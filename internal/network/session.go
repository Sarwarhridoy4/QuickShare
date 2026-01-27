package network

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
)

// SessionState represents the current state of a session
type SessionState int

const (
	SessionPending SessionState = iota
	SessionActive
	SessionClosed
)

// MessageType represents different message types in the protocol
type MessageType byte

const (
	MsgConnectionRequest MessageType = iota
	MsgConnectionAccept
	MsgConnectionReject
	MsgFileOffer
	MsgFileAccept
	MsgFileReject
	MsgFileData
	MsgFileComplete
	MsgProgressUpdate
	MsgKeepAlive
)

// Session represents an active connection between two devices
type Session struct {
	RemoteDevice   *Device
	Conn           net.Conn
	State          SessionState
	IsInitiator    bool
	TransferQueue  []*FileTransferTask
	CurrentTask    *FileTransferTask
	mu             sync.RWMutex
	stopChan       chan bool
	progressChan   chan TransferProgress
	messageChan    chan SessionMessage
}

// FileTransferTask represents a file to be transferred
type FileTransferTask struct {
	FileName       string
	FilePath       string
	FileSize       int64
	Direction      string // "send" or "receive"
	BytesProcessed int64
	StartTime      time.Time
	Status         string
}

// TransferProgress represents progress of a file transfer
type TransferProgress struct {
	FileName       string
	TotalBytes     int64
	TransferredBytes int64
	Speed          float64 // bytes per second
	Percentage     float64
	Direction      string
}

// SessionMessage represents a protocol message
type SessionMessage struct {
	Type    MessageType
	Payload []byte
}

// ConnectionRequest is sent to initiate a connection
type ConnectionRequest struct {
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
}

// FileOffer is sent to offer a file for transfer
type FileOffer struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// NewSession creates a new session
func NewSession(device *Device, conn net.Conn, isInitiator bool) *Session {
	return &Session{
		RemoteDevice:  device,
		Conn:          conn,
		State:         SessionPending,
		IsInitiator:   isInitiator,
		TransferQueue: make([]*FileTransferTask, 0),
		stopChan:      make(chan bool),
		progressChan:  make(chan TransferProgress, 10),
		messageChan:   make(chan SessionMessage, 10),
	}
}

// Start begins the session message handler
func (s *Session) Start() {
	s.mu.Lock()
	s.State = SessionActive
	s.mu.Unlock()
	
	utils.Log(fmt.Sprintf("Session started with %s (%s)", s.RemoteDevice.Name, s.RemoteDevice.IP))
	
	// Start message receiver
	go s.receiveMessages()
	
	// Start keepalive sender
	go s.sendKeepAlive()
}

// SendConnectionRequest sends a connection request
func (s *Session) SendConnectionRequest(deviceName, deviceType string) error {
	req := ConnectionRequest{
		DeviceName: deviceName,
		DeviceType: deviceType,
	}
	
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	return s.sendMessage(MsgConnectionRequest, payload)
}

// AcceptConnection accepts a connection request
func (s *Session) AcceptConnection() error {
	return s.sendMessage(MsgConnectionAccept, []byte("OK"))
}

// RejectConnection rejects a connection request
func (s *Session) RejectConnection() error {
	return s.sendMessage(MsgConnectionReject, []byte("Rejected"))
}

// OfferFile offers a file for transfer
func (s *Session) OfferFile(fileName string, fileSize int64) error {
	offer := FileOffer{
		FileName: fileName,
		FileSize: fileSize,
	}
	
	payload, err := json.Marshal(offer)
	if err != nil {
		return err
	}
	
	return s.sendMessage(MsgFileOffer, payload)
}

// AcceptFile accepts a file offer
func (s *Session) AcceptFile() error {
	return s.sendMessage(MsgFileAccept, []byte("OK"))
}

// sendMessage sends a message over the session
func (s *Session) sendMessage(msgType MessageType, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Write message type
	if err := binary.Write(s.Conn, binary.LittleEndian, msgType); err != nil {
		return err
	}
	
	// Write payload length
	if err := binary.Write(s.Conn, binary.LittleEndian, int32(len(payload))); err != nil {
		return err
	}
	
	// Write payload
	if _, err := s.Conn.Write(payload); err != nil {
		return err
	}
	
	return nil
}

// receiveMessages receives and processes messages
func (s *Session) receiveMessages() {
	defer func() {
		if r := recover(); r != nil {
			utils.Log(fmt.Sprintf("Recovered from panic in receiveMessages: %v", r))
		}
		utils.Log("Message receiver stopped")
	}()
	
	for {
		// Check if we should stop
		select {
		case <-s.stopChan:
			utils.Log("Stopping message receiver - stop signal received")
			return
		default:
		}
		
		// Check session state
		s.mu.RLock()
		state := s.State
		s.mu.RUnlock()
		
		if state != SessionActive {
			utils.Log(fmt.Sprintf("Stopping message receiver - session not active (state: %v)", state))
			return
		}
		
		// Set read deadline to allow checking stopChan periodically
		s.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		
		var msgType MessageType
		if err := binary.Read(s.Conn, binary.LittleEndian, &msgType); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout is expected, continue to check stopChan
				continue
			}
			if err != io.EOF {
				utils.LogError("Error reading message type", err)
			} else {
				utils.Log("Connection closed by remote peer")
			}
			s.Close()
			return
		}
		
		var payloadLen int32
		if err := binary.Read(s.Conn, binary.LittleEndian, &payloadLen); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			utils.LogError("Error reading payload length", err)
			s.Close()
			return
		}
		
		if payloadLen < 0 || payloadLen > 10*1024*1024 {
			utils.Log(fmt.Sprintf("Invalid payload length: %d", payloadLen))
			s.Close()
			return
		}
		
		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			s.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			if _, err := io.ReadFull(s.Conn, payload); err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					utils.LogError("Timeout reading payload", err)
				} else {
					utils.LogError("Error reading payload", err)
				}
				s.Close()
				return
			}
		}
		
		msg := SessionMessage{
			Type:    msgType,
			Payload: payload,
		}
		
		utils.LogDebug(fmt.Sprintf("Received message type: %v, payload size: %d", msgType, payloadLen))
		
		select {
		case s.messageChan <- msg:
		case <-time.After(1 * time.Second):
			utils.Log("Message channel blocked, dropping message")
		}
	}
}

// sendKeepAlive sends periodic keepalive messages
func (s *Session) sendKeepAlive() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			state := s.State
			s.mu.RUnlock()
			
			if state != SessionActive {
				return
			}
			
			if err := s.sendMessage(MsgKeepAlive, []byte("ping")); err != nil {
				utils.LogError("Failed to send keepalive", err)
				s.Close()
				return
			}
			utils.LogDebug("Keepalive sent")
			
		case <-s.stopChan:
			return
		}
	}
}

// GetMessageChannel returns the message channel
func (s *Session) GetMessageChannel() <-chan SessionMessage {
	return s.messageChan
}

// GetProgressChannel returns the progress channel
func (s *Session) GetProgressChannel() <-chan TransferProgress {
	return s.progressChan
}

// SendProgress sends a progress update
func (s *Session) SendProgress(progress TransferProgress) {
	select {
	case s.progressChan <- progress:
	default:
	}
}

// AddToQueue adds a file to the transfer queue
func (s *Session) AddToQueue(task *FileTransferTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TransferQueue = append(s.TransferQueue, task)
}

// Close closes the session
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.State == SessionClosed {
		return
	}
	
	utils.Log(fmt.Sprintf("Closing session with %s", s.RemoteDevice.IP))
	
	s.State = SessionClosed
	
	// Close stop channel first
	select {
	case <-s.stopChan:
		// Already closed
	default:
		close(s.stopChan)
	}
	
	// Give goroutines time to exit gracefully
	time.Sleep(100 * time.Millisecond)
	
	// Close connection
	if s.Conn != nil {
		s.Conn.Close()
	}
	
	utils.Log(fmt.Sprintf("Session closed with %s", s.RemoteDevice.IP))
}