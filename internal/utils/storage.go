package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateDownloadPath checks if a download path is valid and writable
func ValidateDownloadPath(path string) error {
	if path == "" {
		return fmt.Errorf("download path is empty")
	}
	
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("cannot create download directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("cannot access download path: %w", err)
	}
	
	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("download path is not a directory")
	}
	
	// Check write permissions by creating a temp file
	testFile := filepath.Join(path, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("download path is not writable: %w", err)
	}
	file.Close()
	os.Remove(testFile)
	
	return nil
}

// GetAvailableSpace returns available disk space in bytes for the given path
func GetAvailableSpace(path string) (uint64, error) {
	// This is platform-specific and would require syscall
	// For now, return a placeholder
	// In production, use golang.org/x/sys/unix or windows packages
	return 0, fmt.Errorf("not implemented")
}

// EnsureDownloadPath ensures the download path exists and is writable
func EnsureDownloadPath(path string) error {
	if err := ValidateDownloadPath(path); err != nil {
		// If validation fails, try to create it
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create download path: %w", err)
		}
	}
	return nil
}

// GetDefaultDownloadPath returns the default download path for the platform
func GetDefaultDownloadPath() string {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./downloads"
	}
	
	// Use platform-specific downloads folder
	return filepath.Join(homeDir, "Downloads", "FileTransfer")
}

// ListDownloadedFiles returns a list of files in the download directory
func ListDownloadedFiles(downloadPath string) ([]string, error) {
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read download directory: %w", err)
	}
	
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	
	return files, nil
}

// GetDownloadHistory returns recent download history with file info
type DownloadInfo struct {
	FileName   string
	FilePath   string
	FileSize   int64
	Downloaded string // timestamp
}

func GetDownloadHistory(downloadPath string, limit int) ([]DownloadInfo, error) {
	entries, err := os.ReadDir(downloadPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read download directory: %w", err)
	}
	
	var history []DownloadInfo
	count := 0
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		history = append(history, DownloadInfo{
			FileName:   entry.Name(),
			FilePath:   filepath.Join(downloadPath, entry.Name()),
			FileSize:   info.Size(),
			Downloaded: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return history, nil
}

// CleanOldDownloads removes files older than specified days
func CleanOldDownloads(downloadPath string, daysOld int) error {
	// Implementation would check file modification time
	// and remove files older than daysOld days
	// This is left as an exercise for production use
	return fmt.Errorf("not implemented")
}