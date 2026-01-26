package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logger     *log.Logger
	logFile    *os.File
	isDebugMode = true // Set to false in production
)

// InitLogger initializes the logging system
func InitLogger() {
	// Create logs directory
	logDir := filepath.Join(".", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return
	}
	
	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("transfer_%s.log", timestamp))
	
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}
	
	logFile = file
	logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	
	Log("Logger initialized")
}

// Log writes a log entry
func Log(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s", timestamp, message)
	
	if logger != nil {
		logger.Println(message)
	}
	
	if isDebugMode {
		fmt.Println(logMessage)
	}
}

// LogError writes an error log entry
func LogError(message string, err error) {
	errorMessage := fmt.Sprintf("ERROR: %s - %v", message, err)
	Log(errorMessage)
}

// LogDebug writes a debug log entry (only in debug mode)
func LogDebug(message string) {
	if isDebugMode {
		Log(fmt.Sprintf("DEBUG: %s", message))
	}
}

// CloseLogger closes the log file
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// SetDebugMode enables or disables debug mode
func SetDebugMode(enabled bool) {
	isDebugMode = enabled
}