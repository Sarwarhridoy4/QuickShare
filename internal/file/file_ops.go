package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ValidateFilePath checks if a file path is valid and accessible
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path is empty")
	}
	
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}
	
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}
	
	// Check read permissions
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	file.Close()
	
	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CalculateChecksum calculates SHA256 checksum of a file
func CalculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SplitFile splits a large file into chunks (for advanced use cases)
func SplitFile(path string, chunkSize int64) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
//	fileInfo, err := file.Stat()
//	if err != nil {
//		return nil, err
//	}
	
	var chunks []string
	buffer := make([]byte, chunkSize)
	chunkIndex := 0
	
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		
		chunkPath := fmt.Sprintf("%s.part%d", path, chunkIndex)
		chunkFile, err := os.Create(chunkPath)
		if err != nil {
			return nil, err
		}
		
		if _, err := chunkFile.Write(buffer[:n]); err != nil {
			chunkFile.Close()
			return nil, err
		}
		
		chunkFile.Close()
		chunks = append(chunks, chunkPath)
		chunkIndex++
	}
	
	return chunks, nil
}

// MergeChunks merges file chunks back into a single file
func MergeChunks(chunkPaths []string, outputPath string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	
	for _, chunkPath := range chunkPaths {
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		
		if _, err := io.Copy(outputFile, chunkFile); err != nil {
			chunkFile.Close()
			return err
		}
		
		chunkFile.Close()
	}
	
	return nil
}

// GetSafeFilename generates a safe filename by avoiding collisions
func GetSafeFilename(dir, filename string) string {
	fullPath := filepath.Join(dir, filename)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fullPath
	}
	
	ext := filepath.Ext(filename)
	nameWithoutExt := filename[:len(filename)-len(ext)]
	
	counter := 1
	for {
		newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		fullPath = filepath.Join(dir, newFilename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fullPath
		}
		counter++
	}
}

// FormatFileSize formats bytes into human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}