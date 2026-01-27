package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
)

const (
	Port           = 9999
	ChunkSize      = 64 * 1024 // 64KB chunks for optimal transfer
	MaxConcurrency = 4         // Number of concurrent transfer workers
)

type TransferManager struct {
	listener net.Listener
}

func NewTransferManager() *TransferManager {
	return &TransferManager{}
}

// SendFile sends a file to the specified IP address
func (tm *TransferManager) SendFile(filePath, targetIP string, progressChan chan<- float64) error {
	utils.Log(fmt.Sprintf("Starting file transfer: %s to %s", filePath, targetIP))
	
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
	
	// Connect to receiver
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, Port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to receiver: %w", err)
	}
	defer conn.Close()
	
	// Set TCP options for better performance
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	
	// Send filename length and name
	fileNameBytes := []byte(fileName)
	if err := binary.Write(conn, binary.LittleEndian, int32(len(fileNameBytes))); err != nil {
		return fmt.Errorf("failed to send filename length: %w", err)
	}
	if _, err := conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("failed to send filename: %w", err)
	}
	
	// Send file size
	if err := binary.Write(conn, binary.LittleEndian, fileSize); err != nil {
		return fmt.Errorf("failed to send file size: %w", err)
	}
	
	// Send file data in chunks
	buffer := make([]byte, ChunkSize)
	var totalSent int64
	startTime := time.Now()
	
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}
		
		// Send chunk
		if _, err := conn.Write(buffer[:n]); err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
		
		totalSent += int64(n)
		progress := float64(totalSent) / float64(fileSize)
		
		// Update progress (non-blocking)
		select {
		case progressChan <- progress:
		default:
		}
		
		// Log progress periodically
		if totalSent%(ChunkSize*100) == 0 {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(totalSent) / elapsed / 1024 / 1024 // MB/s
			utils.Log(fmt.Sprintf("Progress: %.2f%% (%.2f MB/s)", progress*100, speed))
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalSent) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Transfer complete. Average speed: %.2f MB/s", avgSpeed))
	
	return nil
}

// ReceiveFile listens for incoming file transfers
func (tm *TransferManager) ReceiveFile(downloadPath string, progressChan chan<- float64) (string, error) {
	utils.Log("Starting to listen for incoming files")
	
	// Create listener if not exists
	if tm.listener == nil {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
		if err != nil {
			return "", fmt.Errorf("failed to start listener: %w", err)
		}
		tm.listener = listener
	}
	
	// Accept connection
	conn, err := tm.listener.Accept()
	if err != nil {
		return "", fmt.Errorf("failed to accept connection: %w", err)
	}
	defer conn.Close()
	
	utils.Log(fmt.Sprintf("Connection accepted from %s", conn.RemoteAddr()))
	
	// Set TCP options
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
	}
	
	// Receive filename length
	var fileNameLen int32
	if err := binary.Read(conn, binary.LittleEndian, &fileNameLen); err != nil {
		return "", fmt.Errorf("failed to receive filename length: %w", err)
	}
	
	// Receive filename
	fileNameBytes := make([]byte, fileNameLen)
	if _, err := io.ReadFull(conn, fileNameBytes); err != nil {
		return "", fmt.Errorf("failed to receive filename: %w", err)
	}
	fileName := string(fileNameBytes)
	
	// Receive file size
	var fileSize int64
	if err := binary.Read(conn, binary.LittleEndian, &fileSize); err != nil {
		return "", fmt.Errorf("failed to receive file size: %w", err)
	}
	
	utils.Log(fmt.Sprintf("Receiving file: %s (%.2f MB)", fileName, float64(fileSize)/1024/1024))
	
	// Create downloads directory if not exists
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create downloads directory: %w", err)
	}
	
	// Create output file
	outputPath := filepath.Join(downloadPath, fileName)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()
	
	// Receive file data
	buffer := make([]byte, ChunkSize)
	var totalReceived int64
	startTime := time.Now()
	
	for totalReceived < fileSize {
		n, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to receive data: %w", err)
		}
		if n == 0 {
			break
		}
		
		// Write to file
		if _, err := outputFile.Write(buffer[:n]); err != nil {
			return "", fmt.Errorf("failed to write to file: %w", err)
		}
		
		totalReceived += int64(n)
		progress := float64(totalReceived) / float64(fileSize)
		
		// Update progress (non-blocking)
		select {
		case progressChan <- progress:
		default:
		}
		
		// Log progress periodically
		if totalReceived%(ChunkSize*100) == 0 {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(totalReceived) / elapsed / 1024 / 1024
			utils.Log(fmt.Sprintf("Progress: %.2f%% (%.2f MB/s)", progress*100, speed))
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalReceived) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Receive complete. Average speed: %.2f MB/s", avgSpeed))
	
	return outputPath, nil
}

// TransferFile transfers a single file
func (tm *TransferManager) TransferFile(filePath, targetIP string, progressChan chan<- float64) error {
	utils.Log(fmt.Sprintf("Transferring file: %s", filePath))
	
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
	
	// Connect to receiver
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, Port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to receiver: %w", err)
	}
	defer conn.Close()
	
	// Set TCP options for better performance
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	
	// Send filename length and name
	fileNameBytes := []byte(fileName)
	if err := binary.Write(conn, binary.LittleEndian, int32(len(fileNameBytes))); err != nil {
		return fmt.Errorf("failed to send filename length: %w", err)
	}
	if _, err := conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("failed to send filename: %w", err)
	}
	
	// Send file size
	if err := binary.Write(conn, binary.LittleEndian, fileSize); err != nil {
		return fmt.Errorf("failed to send file size: %w", err)
	}
	
	// Send file data in chunks
	buffer := make([]byte, ChunkSize)
	var totalSent int64
	startTime := time.Now()
	
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}
		
		// Send chunk
		if _, err := conn.Write(buffer[:n]); err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
		
		totalSent += int64(n)
		progress := float64(totalSent) / float64(fileSize)
		
		// Update progress (non-blocking)
		select {
		case progressChan <- progress:
		default:
		}
		
		// Log progress periodically
		if totalSent%(ChunkSize*100) == 0 {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(totalSent) / elapsed / 1024 / 1024 // MB/s
			utils.Log(fmt.Sprintf("Progress: %.2f%% (%.2f MB/s)", progress*100, speed))
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalSent) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Transfer complete. Average speed: %.2f MB/s", avgSpeed))
	
	return nil
}

// TransferDirectory transfers all files in a directory
func (tm *TransferManager) TransferDirectory(dirPath, targetIP string, progressChan chan<- float64) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // Skip directories, handle files only
		}
		return tm.SendFile(path, targetIP, progressChan) // Transfer each file
	})
}

// SendMultipleFiles sends multiple files to the specified IP address
func (tm *TransferManager) SendMultipleFiles(filePaths []string, targetIP string, progressChan chan<- float64) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files to send")
	}
	
	utils.Log(fmt.Sprintf("Starting transfer of %d files to %s", len(filePaths), targetIP))
	
	// Calculate total size
	totalSize := int64(0)
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}
		totalSize += info.Size()
	}
	
	// Connect once to receiver
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, Port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to receiver: %w", err)
	}
	defer conn.Close()
	
	// Set TCP options for better performance
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	
	// Send number of files
	if err := binary.Write(conn, binary.LittleEndian, int32(len(filePaths))); err != nil {
		return fmt.Errorf("failed to send file count: %w", err)
	}
	
	startTime := time.Now()
	totalSent := int64(0)
	
	// Send each file
	for idx, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		
		fileInfo, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
		}
		
		fileSize := fileInfo.Size()
		fileName := filepath.Base(filePath)
		fileNameBytes := []byte(fileName)
		
		utils.Log(fmt.Sprintf("Sending file %d/%d: %s", idx+1, len(filePaths), fileName))
		
		// Send filename length and name
		if err := binary.Write(conn, binary.LittleEndian, int32(len(fileNameBytes))); err != nil {
			file.Close()
			return fmt.Errorf("failed to send filename length: %w", err)
		}
		if _, err := conn.Write(fileNameBytes); err != nil {
			file.Close()
			return fmt.Errorf("failed to send filename: %w", err)
		}
		
		// Send file size
		if err := binary.Write(conn, binary.LittleEndian, fileSize); err != nil {
			file.Close()
			return fmt.Errorf("failed to send file size: %w", err)
		}
		
		// Send file data in chunks
		buffer := make([]byte, ChunkSize)
		fileSent := int64(0)
		
		for {
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				file.Close()
				return fmt.Errorf("failed to read file: %w", err)
			}
			if n == 0 {
				break
			}
			
			// Send chunk
			if _, err := conn.Write(buffer[:n]); err != nil {
				file.Close()
				return fmt.Errorf("failed to send data: %w", err)
			}
			
			fileSent += int64(n)
			totalSent += int64(n)
			progress := float64(totalSent) / float64(totalSize)
			
			// Update progress (non-blocking)
			select {
			case progressChan <- progress:
			default:
			}
		}
		
		file.Close()
		
		// Log file progress periodically
		if totalSent%(ChunkSize*100) == 0 {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(totalSent) / elapsed / 1024 / 1024 // MB/s
			currentProgress := float64(totalSent) / float64(totalSize)
			utils.Log(fmt.Sprintf("Progress: %.2f%% (%.2f MB/s)", currentProgress*100, speed))
		}
	}
	
	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalSent) / elapsed / 1024 / 1024
	utils.Log(fmt.Sprintf("Transfer of %d files complete. Average speed: %.2f MB/s", len(filePaths), avgSpeed))
	
	return nil
}

// Close closes the listener
func (tm *TransferManager) Close() error {
	if tm.listener != nil {
		return tm.listener.Close()
	}
	return nil
}