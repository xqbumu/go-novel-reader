package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig holds the application's configuration.
type AppConfig struct {
	LastNovelPath    string `json:"last_novel_path"`
	LastChapterIndex int    `json:"last_chapter_index"`
}

// DefaultConfigPath returns the default path for the configuration file.
// It typically resides in ~/.config/go-say/config.json
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "go-say")
	return filepath.Join(appConfigDir, "config.json"), nil
}

// LoadConfig loads the configuration from the specified path.
// If the file doesn't exist, it returns a default config.
func LoadConfig(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &AppConfig{}, nil
		}
		return nil, err
	}

	var cfg AppConfig
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig saves the configuration to the specified path.
// It creates the directory if it doesn't exist.
func SaveConfig(configPath string, cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0640)
}
