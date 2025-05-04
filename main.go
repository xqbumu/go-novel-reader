package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort" // Import sort package
	"strconv"

	// "strings" // Removed as unused

	"github.com/xqbumu/go-say/config"
	"github.com/xqbumu/go-say/novel"
	"github.com/xqbumu/go-say/tts"
)

var (
	cfg         *config.AppConfig
	configPath  string
	activeNovel *config.NovelInfo // Holds the currently active novel's info and loaded chapters
)

// Map regex names back to actual regex objects (used after loading config)
// We reference the exported map from the novel package now.
var regexMap = map[string]*regexp.Regexp{
	"chinese":  novel.ChapterRegexes["chinese"],  // Use exported novel.ChapterRegexes
	"english":  novel.ChapterRegexes["english"],  // Use exported novel.ChapterRegexes
	"markdown": novel.ChapterRegexes["markdown"], // Use exported novel.ChapterRegexes
}

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
	// Ensure Novels map is initialized
	if cfg.Novels == nil {
		cfg.Novels = make(map[string]*config.NovelInfo)
	}

	// --- Load Active Novel ---
	if cfg.ActiveNovelPath != "" {
		loadActiveNovel()
	}

	// --- Command Line Argument Parsing ---
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Manages and reads novels using TTS.\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  add <filepath>      Add a new novel to the library and parse chapters.\n")
		fmt.Fprintf(os.Stderr, "  list                List all novels in the library with their index.\n")
		fmt.Fprintf(os.Stderr, "  remove <index>      Remove the novel at the specified index (from 'list') from the library.\n")
		fmt.Fprintf(os.Stderr, "  switch <index>      Set the novel at the specified index (from 'list') as the active one.\n")
		fmt.Fprintf(os.Stderr, "  chapters            List chapters of the active novel.\n")
		fmt.Fprintf(os.Stderr, "  read [chap_index]   Read chapter of active novel (1-based index). Reads last known chapter if index omitted.\n")
		fmt.Fprintf(os.Stderr, "  next                Read the next chapter of the active novel.\n")
		fmt.Fprintf(os.Stderr, "  prev                Read the previous chapter of the active novel.\n")
		fmt.Fprintf(os.Stderr, "  where               Show the active novel and last read chapter index.\n")
		fmt.Fprintf(os.Stderr, "  config [setting]    View or toggle configuration settings.\n")
		fmt.Fprintf(os.Stderr, "                      Available settings: auto_next\n")
		// fmt.Fprintf(os.Stderr, "  continue            Continue reading from the last position (same as 'read').\n") // Merged with read
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)
	args := flag.Args()[1:]

	// --- Command Handling ---
	switch command {
	case "add":
		handleAdd(args)
	case "list":
		handleListNovels()
	case "remove":
		handleRemove(args)
	case "switch":
		handleSwitch(args)
	case "chapters":
		handleChapters()
	case "read", "continue": // Allow 'continue' as alias for 'read' without args
		handleRead(args)
	case "next":
		handleNext()
	case "prev":
		handlePrev()
	case "where":
		handleWhere()
	case "config":
		handleConfig(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

// --- Command Handler Functions ---

func handleConfig(args []string) {
	if len(args) == 0 {
		// View current settings
		fmt.Println("Current Configuration:")
		fmt.Printf("  auto_next: %t\n", cfg.AutoReadNext)
		return
	}

	setting := args[0]
	switch setting {
	case "auto_next":
		cfg.AutoReadNext = !cfg.AutoReadNext // Toggle the setting
		saveConfig()
		fmt.Printf("Set auto_next to: %t\n", cfg.AutoReadNext)
	default:
		log.Fatalf("Error: Unknown config setting '%s'. Available: auto_next", setting)
	}
}

func handleAdd(args []string) {
	if len(args) < 1 {
		log.Fatal("Error: add command requires a filepath argument.")
	}
	filePath, err := filepath.Abs(args[0]) // Store absolute path
	if err != nil {
		log.Fatalf("Error getting absolute path for %s: %v", args[0], err)
	}

	if _, exists := cfg.Novels[filePath]; exists {
		log.Printf("Novel '%s' already exists in the library.", filePath)
		// Optionally switch to it? For now, just inform.
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("Error: File not found: %s", filePath)
	}

	fmt.Printf("Adding novel: %s\n", filePath)
	detectedFormatRegex, err := novel.DetectFormat(filePath)
	if err != nil {
		log.Fatalf("Error detecting format: %v", err)
	}

	// Find the name of the detected regex
	detectedRegexName := ""
	for name, r := range regexMap {
		if r == detectedFormatRegex {
			detectedRegexName = name
			break
		}
	}
	if detectedRegexName == "" {
		log.Println("Warning: Could not map detected regex back to a known name. Using default.")
		detectedRegexName = "markdown" // Fallback, though DetectFormat should handle this
		detectedFormatRegex = regexMap[detectedRegexName]
	}
	fmt.Printf("Detected format: %s\n", detectedRegexName)

	parsedChapters, err := novel.ParseNovel(filePath, detectedFormatRegex)
	if err != nil {
		log.Fatalf("Error parsing novel: %v", err)
	}

	chapterTitles := make([]string, len(parsedChapters))
	for i, ch := range parsedChapters {
		chapterTitles[i] = ch.Title
	}

	newNovelInfo := &config.NovelInfo{
		FilePath:      filePath,
		Chapters:      parsedChapters, // Keep in memory for now
		ChapterTitles: chapterTitles,
		LastReadIndex: 0, // Start at the beginning
		DetectedRegex: detectedRegexName,
	}

	cfg.Novels[filePath] = newNovelInfo
	// Automatically switch to the newly added novel? Yes.
	cfg.ActiveNovelPath = filePath
	activeNovel = newNovelInfo // Update active novel in memory

	saveConfig()
	fmt.Printf("Successfully added '%s' with %d chapters and set as active.\n", filePath, len(parsedChapters))
}

func handleListNovels() {
	if len(cfg.Novels) == 0 {
		fmt.Println("Library is empty. Use 'add <filepath>' to add a novel.")
		return
	}
	fmt.Println("Novels in library:")
	sortedNovels := getNovelsSorted() // Get sorted list
	for i, novelInfo := range sortedNovels {
		activeMarker := " "
		if novelInfo.FilePath == cfg.ActiveNovelPath {
			activeMarker = "*"
		}
		// Display 1-based index for the user
		fmt.Printf(" %s %d: %s (%d chapters, last read: %d)\n",
			activeMarker, i+1, filepath.Base(novelInfo.FilePath), len(novelInfo.ChapterTitles), novelInfo.LastReadIndex+1)
	}
}

func handleRemove(args []string) {
	if len(args) < 1 {
		log.Fatal("Error: remove command requires an index argument.")
	}
	index, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("Error: Invalid index '%s'. Please provide the number shown by 'list'.", args[0])
	}

	sortedNovels := getNovelsSorted()
	if index < 1 || index > len(sortedNovels) {
		log.Fatalf("Error: Index %d is out of range. Valid range is 1 to %d.", index, len(sortedNovels))
	}

	novelToRemove := sortedNovels[index-1] // Get the NovelInfo using 0-based index
	filePath := novelToRemove.FilePath

	delete(cfg.Novels, filePath) // Delete from the map using the path
	fmt.Printf("Removed novel %d: %s\n", index, filepath.Base(filePath))

	// If the removed novel was the active one, clear the active path
	if cfg.ActiveNovelPath == filePath {
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		fmt.Println("The active novel was removed.")
		// Optionally, switch to another novel if available? For now, just clear.
	}

	saveConfig()
}

func handleSwitch(args []string) {
	if len(args) < 1 {
		log.Fatal("Error: switch command requires an index argument.")
	}
	index, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("Error: Invalid index '%s'. Please provide the number shown by 'list'.", args[0])
	}

	sortedNovels := getNovelsSorted()
	if index < 1 || index > len(sortedNovels) {
		log.Fatalf("Error: Index %d is out of range. Valid range is 1 to %d.", index, len(sortedNovels))
	}

	novelToSwitch := sortedNovels[index-1] // Get the NovelInfo using 0-based index
	filePath := novelToSwitch.FilePath

	cfg.ActiveNovelPath = filePath
	activeNovel = novelToSwitch // Update global active novel pointer
	// Load chapters into memory if not already loaded (or re-load?)
	loadActiveNovelChapters() // Separate function to load chapters for the active novel

	saveConfig()
	fmt.Printf("Switched active novel to: %s\n", filePath)
}

func handleChapters() {
	if activeNovel == nil {
		fmt.Println("No active novel selected. Use 'switch <filepath>' first.")
		return
	}
	if len(activeNovel.ChapterTitles) == 0 {
		fmt.Printf("No chapters found or loaded for '%s'.\n", activeNovel.FilePath)
		return
	}
	fmt.Printf("Chapters for '%s':\n", filepath.Base(activeNovel.FilePath))
	for i, title := range activeNovel.ChapterTitles {
		fmt.Printf("  %d: %s\n", i+1, title) // 1-based index for user
	}
}

func handleRead(args []string) {
	if activeNovel == nil {
		fmt.Println("No active novel selected. Use 'switch <filepath>' first.")
		return
	}
	if len(activeNovel.Chapters) == 0 {
		fmt.Printf("Chapters not loaded for '%s'. Try reloading or re-adding.\n", activeNovel.FilePath)
		// Attempt to load chapters now
		loadActiveNovelChapters()
		if len(activeNovel.Chapters) == 0 {
			return // Still no chapters
		}
	}

	targetIndex := activeNovel.LastReadIndex // Default to last read index

	if len(args) > 0 {
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 1 || idx > len(activeNovel.Chapters) {
			log.Fatalf("Error: Invalid chapter index '%s'. Please provide a number between 1 and %d.", args[0], len(activeNovel.Chapters))
		}
		targetIndex = idx - 1 // Convert to 0-based index
	}

	if targetIndex < 0 || targetIndex >= len(activeNovel.Chapters) {
		fmt.Printf("No chapter selected or last read index (%d) is invalid for '%s'. Reading first chapter.\n", activeNovel.LastReadIndex+1, filepath.Base(activeNovel.FilePath))
		targetIndex = 0
	}

	chapter := activeNovel.Chapters[targetIndex]
	fmt.Printf("Reading Chapter %d: %s (from %s)\n", targetIndex+1, chapter.Title, filepath.Base(activeNovel.FilePath))

	// Combine title and content for reading
	textToRead := fmt.Sprintf("%s\n\n%s", chapter.Title, chapter.Content)

	// --- Start Speaking Asynchronously ---
	doneChan, err := tts.SpeakAsync(textToRead)
	if err != nil {
		log.Printf("Error starting TTS for chapter %d: %v", targetIndex+1, err)
		// Don't update progress if TTS couldn't even start
		return
	}

	// Update progress *before* waiting for speech to finish,
	// so 'where' shows the correct chapter *during* reading.
	activeNovel.LastReadIndex = targetIndex
	saveConfig()

	// --- Wait for Speaking to Finish ---
	fmt.Println("Speaking... (Press Ctrl+C to stop the program)") // Basic feedback
	err = <-doneChan                                              // Block until speaking is done or an error occurs

	if err != nil {
		log.Printf("Error during TTS for chapter %d: %v", targetIndex+1, err)
		// Decide if we should still try auto-next on TTS error? Probably not.
		return
	}

	// --- Handle Auto-Next ---
	fmt.Println("Chapter finished.")
	if cfg.AutoReadNext {
		fmt.Println("Auto-reading next chapter...")
		// Need to find the *next* index relative to the one just read
		nextIdxInternal := targetIndex + 1
		if nextIdxInternal < len(activeNovel.Chapters) {
			// Use a goroutine to avoid blocking main flow if handleNext itself speaks
			// Although handleNext calls handleRead which now uses SpeakAsync...
			// Let's call it directly for now. If it causes issues, rethink.
			handleNext() // handleNext calls handleRead with the next index
		} else {
			fmt.Println("Reached the end of the novel.")
		}
	}
}

func handleNext() {
	if activeNovel == nil {
		fmt.Println("No active novel.")
		return
	}

	// Ensure chapters are loaded before checking bounds or proceeding
	loadActiveNovelChapters()
	if len(activeNovel.Chapters) == 0 { // Check if loading succeeded
		fmt.Println("Failed to load chapters for the active novel.")
		return
	}

	nextIndex := activeNovel.LastReadIndex + 1
	if nextIndex >= len(activeNovel.Chapters) {
		fmt.Println("Already at the last chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(nextIndex + 1)}) // Pass 1-based index
}

func handlePrev() {
	if activeNovel == nil {
		fmt.Println("No active novel.")
		return
	}

	// Ensure chapters are loaded before checking bounds or proceeding
	loadActiveNovelChapters()
	if len(activeNovel.Chapters) == 0 { // Check if loading succeeded
		fmt.Println("Failed to load chapters for the active novel.")
		return
	}

	prevIndex := activeNovel.LastReadIndex - 1
	if prevIndex < 0 {
		fmt.Println("Already at the first chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(prevIndex + 1)}) // Pass 1-based index
}

func handleWhere() {
	if cfg.ActiveNovelPath == "" || activeNovel == nil {
		fmt.Println("No novel is currently active.")
		return
	}
	lastRead := activeNovel.LastReadIndex
	title := ""
	if lastRead >= 0 && lastRead < len(activeNovel.ChapterTitles) {
		title = activeNovel.ChapterTitles[lastRead]
	} else {
		title = "(index out of bounds or no chapters)"
	}

	fmt.Printf("Active novel: %s\nLast read: Chapter %d: %s\n",
		activeNovel.FilePath, lastRead+1, title)
}

// --- Helper Functions ---

// getNovelsSorted returns a slice of NovelInfo pointers sorted alphabetically by FilePath.
func getNovelsSorted() []*config.NovelInfo {
	keys := make([]string, 0, len(cfg.Novels))
	for k := range cfg.Novels {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Sort file paths alphabetically

	sorted := make([]*config.NovelInfo, len(keys))
	for i, k := range keys {
		sorted[i] = cfg.Novels[k]
	}
	return sorted
}

// loadActiveNovel finds the active novel info in the config and sets the global activeNovel pointer.
// It does NOT load the chapter content by default.
func loadActiveNovel() {
	info, exists := cfg.Novels[cfg.ActiveNovelPath]
	if !exists {
		fmt.Fprintf(os.Stderr, "Warning: Active novel path '%s' not found in library. Clearing active novel.\n", cfg.ActiveNovelPath)
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		saveConfig() // Save the cleared active path
		return
	}
	activeNovel = info
	// Optionally pre-load chapters here, or do it lazily in handleRead/handleChapters
	// loadActiveNovelChapters() // Let's load lazily for now
}

// loadActiveNovelChapters ensures the chapter content for the active novel is loaded into memory.
func loadActiveNovelChapters() {
	if activeNovel == nil || activeNovel.FilePath == "" {
		log.Println("Error: Cannot load chapters, no active novel set.")
		return
	}
	// Avoid reloading if chapters are already present
	if len(activeNovel.Chapters) > 0 && len(activeNovel.Chapters) == len(activeNovel.ChapterTitles) {
		// Assume already loaded if chapter count matches title count
		return
	}

	fmt.Printf("Loading chapters for: %s\n", activeNovel.FilePath)
	if _, err := os.Stat(activeNovel.FilePath); os.IsNotExist(err) {
		log.Printf("Error: File for active novel not found: %s", activeNovel.FilePath)
		// Consider removing it from config or marking as invalid
		activeNovel.Chapters = nil // Clear any potentially stale data
		return
	}

	// Get the regex based on the stored name
	regex, ok := regexMap[activeNovel.DetectedRegex]
	if !ok {
		log.Printf("Warning: Unknown regex name '%s' stored for novel. Falling back to markdown.", activeNovel.DetectedRegex)
		regex = regexMap["markdown"]
		// Optionally update the stored regex name in config?
		// activeNovel.DetectedRegex = "markdown"
		// saveConfig()
	}

	parsedChapters, err := novel.ParseNovel(activeNovel.FilePath, regex)
	if err != nil {
		log.Printf("Error parsing novel %s: %v", activeNovel.FilePath, err)
		activeNovel.Chapters = nil // Clear on error
		return
	}

	// Update the active novel info in memory (which points to the map entry)
	activeNovel.Chapters = parsedChapters
	// Update titles just in case they changed (though ParseNovel doesn't modify titles)
	activeNovel.ChapterTitles = make([]string, len(parsedChapters))
	for i, ch := range parsedChapters {
		activeNovel.ChapterTitles[i] = ch.Title
	}

	fmt.Printf("Loaded %d chapters.\n", len(activeNovel.Chapters))
	// No need to call saveConfig() here as we only modified the in-memory struct
	// The chapter content itself isn't saved to config.json
}

func saveConfig() {
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
		// Log error but don't necessarily crash the program
		log.Printf("Error saving config to %s: %v", configPath, err)
	}
}
