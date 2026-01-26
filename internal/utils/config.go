package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	DefaultPort      int    `json:"default_port"`
	ChunkSize        int    `json:"chunk_size"`
	MaxConcurrency   int    `json:"max_concurrency"`
	DownloadPath     string `json:"download_path"`
	AutoAccept       bool   `json:"auto_accept"`
	EnableEncryption bool   `json:"enable_encryption"`
}

var defaultConfig = Config{
	DefaultPort:      9999,
	ChunkSize:        65536, // 64KB
	MaxConcurrency:   4,
	DownloadPath:     "./downloads",
	AutoAccept:       false,
	EnableEncryption: false,
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		return &defaultConfig, SaveConfig(&defaultConfig)
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &defaultConfig, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &defaultConfig, err
	}
	
	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	
	// Create config directory if not exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	return filepath.Join(".", "config", "app_config.json")
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() Config {
	return defaultConfig
}