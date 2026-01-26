package utils

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration into human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatSpeed formats transfer speed in MB/s
func FormatSpeed(bytesPerSecond float64) string {
	mbps := bytesPerSecond / 1024 / 1024
	if mbps < 1 {
		kbps := bytesPerSecond / 1024
		return fmt.Sprintf("%.2f KB/s", kbps)
	}
	return fmt.Sprintf("%.2f MB/s", mbps)
}

// CalculateETA calculates estimated time of arrival
func CalculateETA(totalBytes, transferredBytes int64, startTime time.Time) time.Duration {
	if transferredBytes == 0 {
		return 0
	}
	
	elapsed := time.Since(startTime)
	bytesPerSecond := float64(transferredBytes) / elapsed.Seconds()
	remainingBytes := totalBytes - transferredBytes
	
	if bytesPerSecond == 0 {
		return 0
	}
	
	eta := time.Duration(float64(remainingBytes)/bytesPerSecond) * time.Second
	return eta
}

// SanitizeFilename removes or replaces invalid characters from a filename
func SanitizeFilename(filename string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename
	
	for _, char := range invalid {
		sanitized = replaceAll(sanitized, char, "_")
	}
	
	return sanitized
}

func replaceAll(s, old, new string) string {
	result := ""
	for _, ch := range s {
		if string(ch) == old {
			result += new
		} else {
			result += string(ch)
		}
	}
	return result
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation func() error, maxRetries int) error {
	var err error
	backoff := time.Second
	
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}
		
		if i < maxRetries-1 {
			Log(fmt.Sprintf("Operation failed, retrying in %v... (attempt %d/%d)", 
				backoff, i+1, maxRetries))
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}