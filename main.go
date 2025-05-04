package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal" // Import signal package
	"path/filepath"
	"regexp"
	"sort" // Import sort package
	"strconv"
	"strings" // Import strings for splitting
	"syscall" // Import syscall for SIGTERM

	"github.com/xqbumu/go-say/config"
	"github.com/xqbumu/go-say/novel"
	"github.com/xqbumu/go-say/tts"
)

var (
	cfg         *config.AppConfig
	configPath  string
	configDirty bool // Flag to track if main config needs saving

	progressData  config.ProgressData
	progressPath  string
	progressDirty bool // Flag to track if progress data needs saving

	activeNovel *config.NovelInfo // Holds the currently active novel's *metadata*
)

// Map regex names back to actual regex objects
var regexMap = map[string]*regexp.Regexp{
	"chinese":  novel.ChapterRegexes["chinese"],
	"english":  novel.ChapterRegexes["english"],
	"markdown": novel.ChapterRegexes["markdown"],
}

// Define segment separator
var segmentSeparator = regexp.MustCompile(`\n+`)

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

	// --- Progress Data Loading ---
	progressPath, err = config.DefaultProgressPath()
	if err != nil {
		log.Fatalf("Error getting default progress path: %v", err)
	}
	progressData, err = config.LoadProgress(progressPath)
	if err != nil {
		log.Fatalf("Error loading progress data: %v", err) // Fatal on progress load error too
	}

	// --- Setup Signal Handling ---
	setupSignalHandler()

	// --- Defer Save on Normal Exit ---
	defer func() {
		saveOnExit() // Call combined save function
	}()

	// --- Load Active Novel Metadata ---
	if cfg.ActiveNovelPath != "" {
		loadActiveNovelMetadata() // Renamed function
	}

	// --- Command Line Argument Parsing ---
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Manages and reads novels using macOS TTS.\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  add <filepath>      Add a new novel, parse chapters, and set as active.\n")
		fmt.Fprintf(os.Stderr, "  list                List novels in the library with index and last read chapter/segment.\n")
		fmt.Fprintf(os.Stderr, "  remove <index>      Remove the novel at the specified index (from 'list').\n")
		fmt.Fprintf(os.Stderr, "  switch <index>      Set the novel at the specified index (from 'list') as active.\n")
		fmt.Fprintf(os.Stderr, "  chapters            List chapters of the active novel.\n")
		fmt.Fprintf(os.Stderr, "  read [chap_index]   Read active novel segment by segment. Starts from specified chapter (1-based index)\n")
		fmt.Fprintf(os.Stderr, "                      or continues from the last read chapter/segment if index is omitted.\n")
		fmt.Fprintf(os.Stderr, "  next                Read the next chapter of the active novel (starts from segment 0).\n")
		fmt.Fprintf(os.Stderr, "  prev                Read the previous chapter of the active novel (starts from segment 0).\n")
		fmt.Fprintf(os.Stderr, "  where               Show the active novel and the last read chapter/segment index.\n")
		fmt.Fprintf(os.Stderr, "  config [setting]    View or toggle configuration settings.\n")
		fmt.Fprintf(os.Stderr, "                      Available settings: auto_next (toggle auto-read next segment/chapter)\n")
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
	case "read", "continue":
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

// setupSignalHandler registers handlers for interrupt and termination signals.
func setupSignalHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("\nReceived signal: %s. Exiting...\n", sig)
		saveOnExit() // Call combined save function
		os.Exit(0)
	}()
}

// saveOnExit checks dirty flags and saves config/progress if needed.
func saveOnExit() {
	if progressDirty {
		fmt.Println("Progress changed, saving before exit...")
		saveProgress()
	}
	if configDirty {
		fmt.Println("Configuration changed, saving before exit...")
		saveConfig()
	}
}

// --- Command Handler Functions ---

func handleConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("Current Configuration:")
		fmt.Printf("  auto_next: %t\n", cfg.AutoReadNext)
		return
	}
	setting := args[0]
	switch setting {
	case "auto_next":
		cfg.AutoReadNext = !cfg.AutoReadNext
		configDirty = true // Mark main config as dirty
		fmt.Printf("Set auto_next to: %t\n", cfg.AutoReadNext)
	default:
		log.Fatalf("Error: Unknown config setting '%s'. Available: auto_next", setting)
	}
}

func handleAdd(args []string) {
	if len(args) < 1 {
		log.Fatal("Error: add command requires a filepath argument.")
	}
	filePath, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatalf("Error getting absolute path for %s: %v", args[0], err)
	}

	if _, exists := cfg.Novels[filePath]; exists {
		log.Printf("Novel '%s' already exists in the library.", filePath)
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
	detectedRegexName := ""
	for name, r := range regexMap {
		if r == detectedFormatRegex {
			detectedRegexName = name
			break
		}
	}
	if detectedRegexName == "" {
		log.Println("Warning: Could not map detected regex back to a known name. Using default.")
		detectedRegexName = "markdown"
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

	// Create metadata entry
	newNovelInfo := &config.NovelInfo{
		FilePath:      filePath,
		Chapters:      parsedChapters, // Keep chapters in memory for active novel
		ChapterTitles: chapterTitles,
		DetectedRegex: detectedRegexName,
	}
	cfg.Novels[filePath] = newNovelInfo
	cfg.ActiveNovelPath = filePath
	activeNovel = newNovelInfo // Set active novel metadata
	configDirty = true         // Mark main config dirty (ActiveNovelPath changed)

	// Create progress entry
	if _, exists := progressData[filePath]; !exists {
		progressData[filePath] = &config.ProgressInfo{LastReadChapterIndex: 0, LastReadSegmentIndex: 0}
		progressDirty = true // Mark progress dirty
	}

	fmt.Printf("Successfully added '%s' with %d chapters and set as active.\n", filePath, len(parsedChapters))
}

func handleListNovels() {
	if len(cfg.Novels) == 0 {
		fmt.Println("Library is empty. Use 'add <filepath>' to add a novel.")
		return
	}
	fmt.Println("Novels in library:")
	sortedNovels := getNovelsSorted()
	for i, novelInfo := range sortedNovels {
		activeMarker := " "
		if novelInfo.FilePath == cfg.ActiveNovelPath {
			activeMarker = "*"
		}
		// Get progress info for this novel
		progInfo, ok := progressData[novelInfo.FilePath]
		if !ok {
			// Should not happen if add creates progress, but handle defensively
			progInfo = &config.ProgressInfo{LastReadChapterIndex: 0, LastReadSegmentIndex: 0}
		}
		fmt.Printf(" %s %d: %s (%d chapters, last read: Ch %d, Seg %d)\n",
			activeMarker, i+1, filepath.Base(novelInfo.FilePath), len(novelInfo.ChapterTitles),
			progInfo.LastReadChapterIndex+1, progInfo.LastReadSegmentIndex) // Use progressData
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

	novelToRemove := sortedNovels[index-1]
	filePath := novelToRemove.FilePath

	// Remove from main config
	delete(cfg.Novels, filePath)
	configDirty = true
	fmt.Printf("Removed novel metadata %d: %s\n", index, filepath.Base(filePath))

	// Remove from progress data
	if _, exists := progressData[filePath]; exists {
		delete(progressData, filePath)
		progressDirty = true
		fmt.Printf("Removed novel progress data for: %s\n", filepath.Base(filePath))
	}

	if cfg.ActiveNovelPath == filePath {
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		fmt.Println("The active novel was removed.")
		// configDirty is already true
	}
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

	novelToSwitch := sortedNovels[index-1]
	filePath := novelToSwitch.FilePath

	if cfg.ActiveNovelPath != filePath {
		// Save progress for the *previous* active novel if dirty
		if progressDirty {
			fmt.Println("Saving progress for previous novel before switching...")
			saveProgress()
		}
		// Save config for the *previous* active novel if dirty (e.g., auto_next changed)
		if configDirty {
			fmt.Println("Saving config for previous state before switching...")
			saveConfig()
		}

		cfg.ActiveNovelPath = filePath
		activeNovel = novelToSwitch // Update active novel metadata pointer
		loadActiveNovelChapters()   // Load chapters for the new active novel
		configDirty = true          // Mark config dirty because ActiveNovelPath changed
		saveConfig()                // Save immediately to persist the new active path
		fmt.Printf("Switched active novel to: %s\n", filePath)
	} else {
		fmt.Printf("Novel '%s' is already active.\n", filePath)
	}
}

func handleChapters() {
	if activeNovel == nil {
		fmt.Println("No active novel selected. Use 'switch <index>' first.")
		return
	}
	loadActiveNovelChapters() // Ensure chapters are loaded into activeNovel.Chapters
	if len(activeNovel.ChapterTitles) == 0 {
		fmt.Printf("No chapters found or loaded for '%s'.\n", activeNovel.FilePath)
		return
	}
	fmt.Printf("Chapters for '%s':\n", filepath.Base(activeNovel.FilePath))
	for i, title := range activeNovel.ChapterTitles {
		fmt.Printf("  %d: %s\n", i+1, title)
	}
}

func handleRead(args []string) {
	if activeNovel == nil {
		fmt.Println("No active novel selected. Use 'switch <index>' first.")
		return
	}
	loadActiveNovelChapters()
	if len(activeNovel.Chapters) == 0 {
		fmt.Printf("Chapters not loaded for '%s'.\n", activeNovel.FilePath)
		return
	}

	// Get current progress for the active novel
	currentProgress, ok := progressData[activeNovel.FilePath]
	if !ok {
		// Initialize if missing (should not happen if 'add' worked)
		currentProgress = &config.ProgressInfo{LastReadChapterIndex: 0, LastReadSegmentIndex: 0}
		progressData[activeNovel.FilePath] = currentProgress
		progressDirty = true
	}

	targetChapterIndex := currentProgress.LastReadChapterIndex
	startSegmentIndex := currentProgress.LastReadSegmentIndex
	chapterChanged := false

	if len(args) > 0 {
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 1 || idx > len(activeNovel.Chapters) {
			log.Fatalf("Error: Invalid chapter index '%s'. Please provide a number between 1 and %d.", args[0], len(activeNovel.Chapters))
		}
		newChapterIndex := idx - 1
		if newChapterIndex != targetChapterIndex {
			targetChapterIndex = newChapterIndex
			startSegmentIndex = 0 // Reset segment on chapter change
			chapterChanged = true
		}
	}

	// Immediate Save on Chapter Change
	if chapterChanged {
		fmt.Printf("Switching to Chapter %d, saving progress...\n", targetChapterIndex+1)
		currentProgress.LastReadChapterIndex = targetChapterIndex
		currentProgress.LastReadSegmentIndex = startSegmentIndex // Should be 0
		progressDirty = true
		saveProgress() // Save progress immediately
	}

	// Validate targetChapterIndex (could be from loaded progress or args)
	if targetChapterIndex < 0 || targetChapterIndex >= len(activeNovel.Chapters) {
		fmt.Printf("Last read chapter index (%d) is invalid. Reading first chapter.\n", targetChapterIndex+1)
		targetChapterIndex = 0
		startSegmentIndex = 0
		if currentProgress.LastReadChapterIndex != 0 || currentProgress.LastReadSegmentIndex != 0 {
			currentProgress.LastReadChapterIndex = 0
			currentProgress.LastReadSegmentIndex = 0
			progressDirty = true
			saveProgress() // Save corrected progress
		}
	}

	chapter := activeNovel.Chapters[targetChapterIndex]
	fmt.Printf("--- Reading Chapter %d: %s ---\n", targetChapterIndex+1, chapter.Title)

	segmentsReadInSession := 0
	segments := segmentSeparator.Split(chapter.Content, -1)
	if len(segments) == 0 {
		fmt.Println("Chapter content appears empty or has no segments.")
		return
	}

	// Validate startSegmentIndex
	if startSegmentIndex < 0 || startSegmentIndex >= len(segments) {
		fmt.Printf("Warning: Last read segment index (%d) is invalid for this chapter. Starting from segment 0.\n", startSegmentIndex)
		startSegmentIndex = 0
		if currentProgress.LastReadSegmentIndex != 0 {
			currentProgress.LastReadSegmentIndex = 0
			progressDirty = true
			saveProgress() // Save corrected progress
		}
	}

	for segIdx := startSegmentIndex; segIdx < len(segments); segIdx++ {
		segmentText := strings.TrimSpace(segments[segIdx])
		if segmentText == "" {
			continue
		}

		fmt.Printf("\n[Segment %d/%d]\n%s\n", segIdx+1, len(segments), segmentText)

		doneChan, err := tts.SpeakAsync(segmentText)
		if err != nil {
			log.Printf("Error starting TTS for Ch %d, Seg %d: %v", targetChapterIndex+1, segIdx, err)
			return
		}

		// Update progress in memory *before* waiting
		if currentProgress.LastReadChapterIndex != targetChapterIndex || currentProgress.LastReadSegmentIndex != segIdx {
			currentProgress.LastReadChapterIndex = targetChapterIndex
			currentProgress.LastReadSegmentIndex = segIdx
			progressDirty = true // Mark progress dirty
		}

		fmt.Println("(Speaking...)")
		err = <-doneChan

		if err != nil {
			log.Printf("Error during TTS for Ch %d, Seg %d: %v", targetChapterIndex+1, segIdx, err)
			return
		}
		fmt.Println("(Segment finished)")
		segmentsReadInSession++

		// Periodic Save
		if segmentsReadInSession%20 == 0 && progressDirty {
			fmt.Printf("(Auto-saving progress after %d segments...)\n", segmentsReadInSession)
			saveProgress() // Save progress data
		}

		if !cfg.AutoReadNext {
			fmt.Println("Auto-next disabled. Stopping.")
			return
		}
		if segIdx == len(segments)-1 {
			break
		}
	}

	// Auto-Next Chapter
	if cfg.AutoReadNext {
		fmt.Println("Chapter finished. Auto-reading next chapter...")
		nextChapterIndexInternal := targetChapterIndex + 1
		if nextChapterIndexInternal < len(activeNovel.Chapters) {
			// handleRead will detect chapter change and save progress
			handleRead([]string{strconv.Itoa(nextChapterIndexInternal + 1)})
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
	loadActiveNovelChapters()
	if len(activeNovel.Chapters) == 0 {
		fmt.Println("Failed to load chapters for the active novel.")
		return
	}
	// Get current progress
	currentProgress, ok := progressData[activeNovel.FilePath]
	if !ok {
		log.Printf("Error: Progress data not found for active novel %s", activeNovel.FilePath)
		return
	}
	nextChapterIndex := currentProgress.LastReadChapterIndex + 1
	if nextChapterIndex >= len(activeNovel.Chapters) {
		fmt.Println("Already at the last chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(nextChapterIndex + 1)})
}

func handlePrev() {
	if activeNovel == nil {
		fmt.Println("No active novel.")
		return
	}
	loadActiveNovelChapters()
	if len(activeNovel.Chapters) == 0 {
		fmt.Println("Failed to load chapters for the active novel.")
		return
	}
	// Get current progress
	currentProgress, ok := progressData[activeNovel.FilePath]
	if !ok {
		log.Printf("Error: Progress data not found for active novel %s", activeNovel.FilePath)
		return
	}
	prevChapterIndex := currentProgress.LastReadChapterIndex - 1
	if prevChapterIndex < 0 {
		fmt.Println("Already at the first chapter.")
		return
	}
	handleRead([]string{strconv.Itoa(prevChapterIndex + 1)})
}

func handleWhere() {
	if cfg.ActiveNovelPath == "" || activeNovel == nil {
		fmt.Println("No novel is currently active.")
		return
	}
	// Get progress info
	progInfo, ok := progressData[activeNovel.FilePath]
	if !ok {
		fmt.Printf("Active novel: %s\nProgress data not found.\n", activeNovel.FilePath)
		return
	}
	lastChapIdx := progInfo.LastReadChapterIndex
	lastSegIdx := progInfo.LastReadSegmentIndex
	title := ""
	if lastChapIdx >= 0 && lastChapIdx < len(activeNovel.ChapterTitles) {
		title = activeNovel.ChapterTitles[lastChapIdx]
	} else {
		title = "(chapter index out of bounds)"
	}
	fmt.Printf("Active novel: %s\nLast read: Chapter %d (%s), Segment %d\n",
		activeNovel.FilePath, lastChapIdx+1, title, lastSegIdx)
}

// --- Helper Functions ---

func getNovelsSorted() []*config.NovelInfo {
	keys := make([]string, 0, len(cfg.Novels))
	for k := range cfg.Novels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make([]*config.NovelInfo, len(keys))
	for i, k := range keys {
		sorted[i] = cfg.Novels[k]
	}
	return sorted
}

// loadActiveNovelMetadata loads only the metadata for the active novel.
func loadActiveNovelMetadata() {
	info, exists := cfg.Novels[cfg.ActiveNovelPath]
	if !exists {
		fmt.Fprintf(os.Stderr, "Warning: Active novel path '%s' not found in library. Clearing active novel.\n", cfg.ActiveNovelPath)
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		configDirty = true // Mark config dirty
		return
	}
	activeNovel = info
	// Chapters are loaded lazily by loadActiveNovelChapters
}

// loadActiveNovelChapters ensures the chapter content for the active novel is loaded into memory.
func loadActiveNovelChapters() {
	if activeNovel == nil || activeNovel.FilePath == "" {
		return
	}
	// Check if chapters are already loaded (simple check based on slice length)
	if len(activeNovel.Chapters) > 0 && len(activeNovel.Chapters) == len(activeNovel.ChapterTitles) {
		return
	}

	fmt.Printf("Loading chapters for: %s\n", activeNovel.FilePath)
	if _, err := os.Stat(activeNovel.FilePath); os.IsNotExist(err) {
		log.Printf("Error: File for active novel not found: %s", activeNovel.FilePath)
		activeNovel.Chapters = nil // Clear potentially stale chapter data
		return
	}

	regex, ok := regexMap[activeNovel.DetectedRegex]
	if !ok {
		log.Printf("Warning: Unknown regex name '%s' stored for novel. Falling back to markdown.", activeNovel.DetectedRegex)
		regex = regexMap["markdown"]
	}

	parsedChapters, err := novel.ParseNovel(activeNovel.FilePath, regex)
	if err != nil {
		log.Printf("Error parsing novel %s: %v", activeNovel.FilePath, err)
		activeNovel.Chapters = nil
		return
	}

	activeNovel.Chapters = parsedChapters // Store loaded chapters in the activeNovel struct
	// Ensure ChapterTitles matches the loaded chapters (though ParseNovel doesn't change titles)
	if len(activeNovel.ChapterTitles) != len(parsedChapters) {
		log.Printf("Warning: Chapter title count mismatch after loading for %s. Rebuilding titles.", activeNovel.FilePath)
		activeNovel.ChapterTitles = make([]string, len(parsedChapters))
		for i, ch := range parsedChapters {
			activeNovel.ChapterTitles[i] = ch.Title
		}
		configDirty = true // Mark config dirty as ChapterTitles changed
	}

	fmt.Printf("Loaded %d chapters.\n", len(activeNovel.Chapters))
}

// saveConfig saves the main application configuration.
func saveConfig() {
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
		log.Printf("Error saving config to %s: %v", configPath, err)
	} else {
		fmt.Println("Configuration saved.")
		configDirty = false // Reset dirty flag
	}
}

// saveProgress saves the reading progress data.
func saveProgress() {
	err := config.SaveProgress(progressPath, progressData)
	if err != nil {
		log.Printf("Error saving progress to %s: %v", progressPath, err)
	} else {
		fmt.Println("Progress saved.")
		progressDirty = false // Reset dirty flag
	}
}
