package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/xqbumu/go-say/config"
	"github.com/xqbumu/go-say/novel"
	"github.com/xqbumu/go-say/tts"
)

var (
	cfg        *config.AppConfig
	configPath string
	chapters   []novel.Chapter
)

func main() {
	// --- Configuration Loading ---
	var err error
	configPath, err = config.DefaultConfigPath()
	if err != nil {
		log.Fatalf("Error getting default config path: %v", err)
	}
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// --- Command Line Argument Parsing ---
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  open <filepath>   Open a novel file and parse chapters.\n")
		fmt.Fprintf(os.Stderr, "  list              List chapters of the currently open novel.\n")
		fmt.Fprintf(os.Stderr, "  read [index]      Read the chapter at the given index (1-based). Reads last known chapter if index is omitted.\n")
		fmt.Fprintf(os.Stderr, "  next              Read the next chapter.\n")
		fmt.Fprintf(os.Stderr, "  prev              Read the previous chapter.\n")
		fmt.Fprintf(os.Stderr, "  where             Show the last read chapter index.\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// --- Load current novel if path exists in config ---
	if cfg.LastNovelPath != "" {
		loadCurrentNovel() // Load chapters from the path in config
	}

	// --- Command Handling ---
	switch command {
	case "open":
		handleOpen(flag.Args()[1:])
	case "list":
		handleList()
	case "read":
		handleRead(flag.Args()[1:])
	case "next":
		handleNext()
	case "prev":
		handlePrev()
	case "where":
		handleWhere()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

// --- Command Handler Functions ---

func handleOpen(args []string) {
	if len(args) < 1 {
		log.Fatal("Error: open command requires a filepath argument.")
	}
	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("Error: File not found: %s", filePath)
	}

	fmt.Printf("Opening novel: %s\n", filePath)
	detectedFormat, err := novel.DetectFormat(filePath)
	if err != nil {
		log.Fatalf("Error detecting format: %v", err)
	}
	fmt.Printf("Detected format: %p\n", detectedFormat) // Just showing the regex pointer for now

	newChapters, err := novel.ParseNovel(filePath, detectedFormat)
	if err != nil {
		log.Fatalf("Error parsing novel: %v", err)
	}

	chapters = newChapters
	cfg.LastNovelPath = filePath
	cfg.LastChapterIndex = 0 // Reset index when opening new book
	saveConfig()
	fmt.Printf("Successfully opened and parsed %d chapters.\n", len(chapters))
	handleList() // Show chapters after opening
}

func handleList() {
	if len(chapters) == 0 {
		fmt.Println("No novel open or no chapters found. Use 'open <filepath>' first.")
		return
	}
	fmt.Println("Chapters:")
	for i, ch := range chapters {
		fmt.Printf("  %d: %s\n", i+1, ch.Title) // 1-based index for user
	}
}

func handleRead(args []string) {
	if len(chapters) == 0 {
		fmt.Println("No novel open. Use 'open <filepath>' first.")
		return
	}

	targetIndex := cfg.LastChapterIndex // Default to last read index

	if len(args) > 0 {
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 1 || idx > len(chapters) {
			log.Fatalf("Error: Invalid chapter index '%s'. Please provide a number between 1 and %d.", args[0], len(chapters))
		}
		targetIndex = idx - 1 // Convert to 0-based index
	}

	if targetIndex < 0 || targetIndex >= len(chapters) {
		fmt.Println("No chapter selected or last read index is invalid. Reading first chapter.")
		targetIndex = 0
	}

	chapter := chapters[targetIndex]
	fmt.Printf("Reading Chapter %d: %s\n", targetIndex+1, chapter.Title)

	// Combine title and content for reading
	textToRead := fmt.Sprintf("%s\n\n%s", chapter.Title, chapter.Content)

	err := tts.Speak(textToRead)
	if err != nil {
		log.Printf("Error speaking chapter %d: %v", targetIndex+1, err)
		// Don't update config if speaking fails? Or update anyway? Let's update.
	}

	// Update and save config only if speaking was attempted (even if it failed)
	cfg.LastChapterIndex = targetIndex
	saveConfig()
}

func handleNext() {
	if len(chapters) == 0 {
		fmt.Println("No novel open.")
		return
	}
	nextIndex := cfg.LastChapterIndex + 1
	if nextIndex >= len(chapters) {
		fmt.Println("Already at the last chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(nextIndex + 1)}) // Pass 1-based index
}

func handlePrev() {
	if len(chapters) == 0 {
		fmt.Println("No novel open.")
		return
	}
	prevIndex := cfg.LastChapterIndex - 1
	if prevIndex < 0 {
		fmt.Println("Already at the first chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(prevIndex + 1)}) // Pass 1-based index
}

func handleWhere() {
	if cfg.LastNovelPath == "" || len(chapters) == 0 {
		fmt.Println("No novel is currently open or loaded.")
		return
	}
	if cfg.LastChapterIndex < 0 || cfg.LastChapterIndex >= len(chapters) {
		fmt.Printf("Currently open: %s (No specific chapter read yet or index invalid)\n", cfg.LastNovelPath)
	} else {
		fmt.Printf("Currently open: %s\nLast read: Chapter %d: %s\n",
			cfg.LastNovelPath, cfg.LastChapterIndex+1, chapters[cfg.LastChapterIndex].Title)
	}
}

// --- Helper Functions ---

func loadCurrentNovel() {
	if _, err := os.Stat(cfg.LastNovelPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: Last novel file not found: %s\n", cfg.LastNovelPath)
		cfg.LastNovelPath = "" // Clear invalid path
		saveConfig()
		return
	}

	fmt.Printf("Loading novel: %s\n", cfg.LastNovelPath)
	detectedFormat, err := novel.DetectFormat(cfg.LastNovelPath)
	if err != nil {
		log.Printf("Error detecting format for %s: %v", cfg.LastNovelPath, err)
		// Proceed without chapters, or handle error differently?
		return
	}
	chapters, err = novel.ParseNovel(cfg.LastNovelPath, detectedFormat)
	if err != nil {
		log.Printf("Error parsing novel %s: %v", cfg.LastNovelPath, err)
		// Clear chapters if parsing fails
		chapters = nil
		return
	}
	fmt.Printf("Loaded %d chapters.\n", len(chapters))
}

func saveConfig() {
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
		// Log error but don't necessarily crash the program
		log.Printf("Error saving config to %s: %v", configPath, err)
	}
}
