package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xqbumu/go-say/novel" // Import novel package
)

// --- Main Configuration ---

// NovelInfo holds metadata for a single novel (progress is stored separately).
type NovelInfo struct {
	FilePath      string          `json:"file_path"`
	Chapters      []novel.Chapter `json:"-"`                        // Chapters loaded in memory, not saved to JSON directly
	ChapterTitles []string        `json:"chapter_titles"`           // Save titles to JSON for listing
	DetectedRegex string          `json:"detected_regex,omitempty"` // Store the name of the detected regex ("chinese", "english", "markdown")
}

// AppConfig holds the application's less frequently changing configuration.
type AppConfig struct {
	Novels          map[string]*NovelInfo `json:"novels"` // Map from FilePath to NovelInfo
	ActiveNovelPath string                `json:"active_novel_path"`
	AutoReadNext    bool                  `json:"auto_read_next,omitempty"` // Feature: Auto-read next chapter
}

// DefaultConfigPath returns the default path for the main configuration file.
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "go-say")
	return filepath.Join(appConfigDir, "config.json"), nil
}

// LoadConfig loads the main configuration from the specified path.
func LoadConfig(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &AppConfig{
				Novels:       make(map[string]*NovelInfo),
				AutoReadNext: false,
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
	// Ensure Novels map is initialized after loading
	if cfg.Novels == nil {
		cfg.Novels = make(map[string]*NovelInfo)
	}
	return &cfg, nil
}

// SaveConfig saves the main configuration to the specified path.
func SaveConfig(configPath string, cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0640)
}

// --- Progress Data ---

// ProgressInfo holds the reading progress for a single novel.
type ProgressInfo struct {
	LastReadChapterIndex int `json:"last_read_chapter_index"`
	LastReadSegmentIndex int `json:"last_read_segment_index"`
}

// ProgressData holds the reading progress for all novels.
type ProgressData map[string]*ProgressInfo // Map from FilePath to ProgressInfo

// DefaultProgressPath returns the default path for the progress file.
func DefaultProgressPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "go-say")
	return filepath.Join(appConfigDir, "progress.json"), nil
}

// LoadProgress loads the progress data from the specified path.
// If the file doesn't exist, it returns an initialized map.
func LoadProgress(progressPath string) (ProgressData, error) {
	data, err := os.ReadFile(progressPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty initialized map if file doesn't exist
			return make(ProgressData), nil
		}
		return nil, err
	}

	var progress ProgressData
	err = json.Unmarshal(data, &progress)
	if err != nil {
		// If unmarshalling fails, maybe return an empty map instead of error?
		// Or log a warning and return empty map. Let's return empty for robustness.
		// log.Printf("Warning: Could not unmarshal progress file %s: %v. Starting with empty progress.", progressPath, err)
		return make(ProgressData), nil // Return empty map on error
	}
	// Ensure map is not nil after loading
	if progress == nil {
		progress = make(ProgressData)
	}
	return progress, nil
}

// SaveProgress saves the progress data to the specified path.
func SaveProgress(progressPath string, progress ProgressData) error {
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(progressPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	return os.WriteFile(progressPath, data, 0640)
}
