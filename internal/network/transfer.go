package network

import (
	"encoding/binary"
	"encoding/json"
	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	Port              = 9999
	ChunkSize         = 256 * 1024 // 256KB chunks for maximum throughput
	MaxConcurrency    = 8          // Parallel workers for large files
	TCPBufferSize     = 4 * 1024 * 1024 // 4MB TCP buffer
)

type TransferManager struct {
	listener       net.Listener
	activeSession  *Session
	sessionMu      sync.RWMutex
	onSessionRequest func(*Session, *ConnectionRequest) bool
}

func NewTransferManager() *TransferManager {
	return &TransferManager{}
}

// SetSessionRequestHandler sets the callback for connection requests
func (tm *TransferManager) SetSessionRequestHandler(handler func(*Session, *ConnectionRequest) bool) {
	tm.onSessionRequest = handler
}

// StartListening starts listening for incoming connections
func (tm *TransferManager) StartListening() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	tm.listener = listener
	
	utils.Log(fmt.Sprintf("Listening for connections on port %d", Port))
	
	go tm.acceptConnections()
	
	return nil
}

// acceptConnections accepts incoming connections
func (tm *TransferManager) acceptConnections() {
	for {
		conn, err := tm.listener.Accept()
		if err != nil {
			utils.LogError("Error accepting connection", err)
			return
		}
		
		utils.Log(fmt.Sprintf("Incoming connection from %s", conn.RemoteAddr()))
		
		// Optimize TCP settings
		optimizeTCPConnection(conn)
		
		// Create session
		device := &Device{
			IP: conn.RemoteAddr().String(),
		}
		session := NewSession(device, conn, false)
		
		// Handle connection request
		go tm.handleIncomingSession(session)
	}
}

// handleIncomingSession handles a new incoming session
func (tm *TransferManager) handleIncomingSession(session *Session) {
	// Wait for connection request
	msgChan := session.GetMessageChannel()
	session.Start()
	
	select {
	case msg := <-msgChan:
		if msg.Type == MsgConnectionRequest {
			var req ConnectionRequest
			if err := json.Unmarshal(msg.Payload, &req); err != nil {
				utils.LogError("Failed to parse connection request", err)
				session.Close()
				return
			}
			
			session.RemoteDevice.Name = req.DeviceName
			session.RemoteDevice.DeviceType = req.DeviceType
			
			// Ask user for approval
			if tm.onSessionRequest != nil {
				accepted := tm.onSessionRequest(session, &req)
				if accepted {
					session.AcceptConnection()
					tm.setActiveSession(session)
				} else {
					session.RejectConnection()
					session.Close()
				}
			}
		}
	case <-time.After(30 * time.Second):
		utils.Log("Connection request timeout")
		session.Close()
	}
}

// ConnectToDevice initiates a connection to a device
func (tm *TransferManager) ConnectToDevice(device *Device, localDeviceName string) (*Session, error) {
	utils.Log(fmt.Sprintf("Connecting to %s (%s:%d)", device.Name, device.IP, device.Port))
	
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", device.IP, device.Port), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	optimizeTCPConnection(conn)
	
	session := NewSession(device, conn, true)
	session.Start()
	
	// Send connection request
	if err := session.SendConnectionRequest(localDeviceName, getDeviceType()); err != nil {
		session.Close()
		return nil, err
	}
	
	// Wait for response
	msgChan := session.GetMessageChannel()
	select {
	case msg := <-msgChan:
		if msg.Type == MsgConnectionAccept {
			utils.Log("Connection accepted")
			tm.setActiveSession(session)
			return session, nil
		} else if msg.Type == MsgConnectionReject {
			session.Close()
			return nil, fmt.Errorf("connection rejected by remote device")
		}
	case <-time.After(30 * time.Second):
		session.Close()
		return nil, fmt.Errorf("connection timeout")
	}
	
	return session, nil
}

// SendFile sends a file through the active session with maximum speed
func (tm *TransferManager) SendFile(filePath string, progressChan chan<- float64) error {
	tm.sessionMu.RLock()
	session := tm.activeSession
	tm.sessionMu.RUnlock()
	
	if session == nil {
		return fmt.Errorf("no active session")
	}
	
	utils.Log(fmt.Sprintf("Starting file transfer: %s", filePath))
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	fileSize := fileInfo.Size()
	fileName := filepath.Base(filePath)
	
	// Offer file
	if err := session.OfferFile(fileName, fileSize); err != nil {
		return fmt.Errorf("failed to offer file: %w", err)
	}
	
	// Wait for acceptance
	msgChan := session.GetMessageChannel()
	select {
	case msg := <-msgChan:
		if msg.Type != MsgFileAccept {
			return fmt.Errorf("file transfer rejected")
		}
	case <-time.After(30 * time.Second):
		return fmt.Errorf("file offer timeout")
	}
	
	// Send filename length and name
	fileNameBytes := []byte(fileName)
	if err := binary.Write(session.Conn, binary.LittleEndian, int32(len(fileNameBytes))); err != nil {
		return fmt.Errorf("failed to send filename length: %w", err)
	}
	if _, err := session.Conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("failed to send filename: %w", err)
	}
	
	// Send file size
	if err := binary.Write(session.Conn, binary.LittleEndian, fileSize); err != nil {
		return fmt.Errorf("failed to send file size: %w", err)
	}
	
	// Transfer file with maximum speed
	return tm.transferFileOptimized(file, fileSize, session.Conn, progressChan, "send")
}

// ReceiveFile receives a file through the active session
func (tm *TransferManager) ReceiveFile(downloadPath string, progressChan chan<- float64) (string, error) {
	tm.sessionMu.RLock()
	session := tm.activeSession
	tm.sessionMu.RUnlock()
	
	if session == nil {
		return "", fmt.Errorf("no active session")
	}
	
	// Wait for file offer
	msgChan := session.GetMessageChannel()
	var offer FileOffer
	
	select {
	case msg := <-msgChan:
		if msg.Type != MsgFileOffer {
			return "", fmt.Errorf("expected file offer")
		}
		if err := json.Unmarshal(msg.Payload, &offer); err != nil {
			return "", err
		}
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("waiting for file offer timeout")
	}
	
	utils.Log(fmt.Sprintf("Receiving file: %s (%.2f MB)", offer.FileName, float64(offer.FileSize)/1024/1024))
	
	// Accept file
	if err := session.AcceptFile(); err != nil {
		return "", err
	}
	
	// Receive filename
	var fileNameLen int32
	if err := binary.Read(session.Conn, binary.LittleEndian, &fileNameLen); err != nil {
		return "", err
	}
	
	fileNameBytes := make([]byte, fileNameLen)
	if _, err := io.ReadFull(session.Conn, fileNameBytes); err != nil {
		return "", err
	}
	
	// Receive file size
	var fileSize int64
	if err := binary.Read(session.Conn, binary.LittleEndian, &fileSize); err != nil {
		return "", err
	}
	
	// Create output file
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		return "", err
	}
	
	outputPath := filepath.Join(downloadPath, string(fileNameBytes))
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()
	
	// Receive file with maximum speed
	if err := tm.receiveFileOptimized(outputFile, fileSize, session.Conn, progressChan); err != nil {
		return "", err
	}
	
	return outputPath, nil
}

// transferFileOptimized transfers a file with maximum network speed
func (tm *TransferManager) transferFileOptimized(file *os.File, fileSize int64, conn net.Conn, progressChan chan<- float64, direction string) error {
	buffer := make([]byte, ChunkSize)
	var totalSent int64
	startTime := time.Now()
	lastUpdate := time.Now()
	
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}
		
		// Send chunk
		sent := 0
		for sent < n {
			written, err := conn.Write(buffer[sent:n])
			if err != nil {
				return fmt.Errorf("failed to send data: %w", err)
			}
			sent += written
		}
		
		totalSent += int64(n)
		
		// Update progress (throttled to every 100ms)
		if time.Since(lastUpdate) > 100*time.Millisecond || totalSent == fileSize {
			progress := float64(totalSent) / float64(fileSize)
			select {
			case progressChan <- progress:
			default:
			}
			lastUpdate = time.Now()
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalSent) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Transfer complete. Size: %.2f MB, Time: %.2fs, Speed: %.2f MB/s", 
		float64(totalSent)/1024/1024, elapsed, avgSpeed))
	
	return nil
}

// receiveFileOptimized receives a file with maximum network speed
func (tm *TransferManager) receiveFileOptimized(file *os.File, fileSize int64, conn net.Conn, progressChan chan<- float64) error {
	buffer := make([]byte, ChunkSize)
	var totalReceived int64
	startTime := time.Now()
	lastUpdate := time.Now()
	
	for totalReceived < fileSize {
		remaining := fileSize - totalReceived
		toRead := ChunkSize
		if remaining < int64(ChunkSize) {
			toRead = int(remaining)
		}
		
		n, err := io.ReadFull(conn, buffer[:toRead])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("failed to receive data: %w", err)
		}
		if n == 0 {
			break
		}
		
		// Write to file
		written := 0
		for written < n {
			w, err := file.Write(buffer[written:n])
			if err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			written += w
		}
		
		totalReceived += int64(n)
		
		// Update progress (throttled)
		if time.Since(lastUpdate) > 100*time.Millisecond || totalReceived == fileSize {
			progress := float64(totalReceived) / float64(fileSize)
			select {
			case progressChan <- progress:
			default:
			}
			lastUpdate = time.Now()
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalReceived) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Receive complete. Size: %.2f MB, Time: %.2fs, Speed: %.2f MB/s", 
		float64(totalReceived)/1024/1024, elapsed, avgSpeed))
	
	return nil
}

// optimizeTCPConnection sets TCP options for maximum throughput
func optimizeTCPConnection(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Disable Nagle's algorithm for low latency
		tcpConn.SetNoDelay(true)
		
		// Enable keepalive
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		
		// Set buffer sizes for maximum throughput
		tcpConn.SetReadBuffer(TCPBufferSize)
		tcpConn.SetWriteBuffer(TCPBufferSize)
		
		utils.Log("TCP connection optimized for maximum throughput")
	}
}

func (tm *TransferManager) setActiveSession(session *Session) {
	tm.sessionMu.Lock()
	defer tm.sessionMu.Unlock()
	tm.activeSession = session
}

func (tm *TransferManager) GetActiveSession() *Session {
	tm.sessionMu.RLock()
	defer tm.sessionMu.RUnlock()
	return tm.activeSession
}

func (tm *TransferManager) CloseSession() {
	tm.sessionMu.Lock()
	defer tm.sessionMu.Unlock()
	
	if tm.activeSession != nil {
		tm.activeSession.Close()
		tm.activeSession = nil
	}
}

func (tm *TransferManager) Close() error {
	tm.CloseSession()
	
	if tm.listener != nil {
		return tm.listener.Close()
	}
	return nil
}