package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xqbumu/go-say/novel" // Import novel package
)

// NovelInfo holds metadata and progress for a single novel.
type NovelInfo struct {
	FilePath             string          `json:"file_path"`
	Chapters             []novel.Chapter `json:"-"`                        // Chapters loaded in memory, not saved to JSON directly
	ChapterTitles        []string        `json:"chapter_titles"`           // Save titles to JSON for listing
	LastReadChapterIndex int             `json:"last_read_chapter_index"`  // Renamed for clarity
	LastReadSegmentIndex int             `json:"last_read_segment_index"`  // Index of the last successfully read segment within the chapter
	DetectedRegex        string          `json:"detected_regex,omitempty"` // Store the name of the detected regex ("chinese", "english", "markdown")
}

// AppConfig holds the application's configuration for multiple novels.
type AppConfig struct {
	Novels          map[string]*NovelInfo `json:"novels"` // Map from FilePath to NovelInfo
	ActiveNovelPath string                `json:"active_novel_path"`
	AutoReadNext    bool                  `json:"auto_read_next,omitempty"` // Feature: Auto-read next chapter
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
// If the file doesn't exist, it returns an initialized config.
func LoadConfig(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default initialized config if file doesn't exist
			cfg := &AppConfig{
				Novels:       make(map[string]*NovelInfo),
				AutoReadNext: false, // Default to false
			}
			return cfg, nil
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
