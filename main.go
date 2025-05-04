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
	activeNovel *config.NovelInfo // Holds the currently active novel's info and loaded chapters
	configDirty bool              // Flag to track if config needs saving on exit
)

// Map regex names back to actual regex objects (used after loading config)
var regexMap = map[string]*regexp.Regexp{
	"chinese":  novel.ChapterRegexes["chinese"],
	"english":  novel.ChapterRegexes["english"],
	"markdown": novel.ChapterRegexes["markdown"],
}

// Define segment separator: one or more newline characters.
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
	if cfg.Novels == nil {
		cfg.Novels = make(map[string]*config.NovelInfo)
	}

	// --- Setup Signal Handling ---
	setupSignalHandler() // Call the new function to handle signals

	// --- Defer Save on Normal Exit ---
	defer func() {
		if configDirty {
			fmt.Println("\nConfiguration changed, saving on normal exit...")
			saveConfig()
		}
	}()

	// --- Load Active Novel ---
	if cfg.ActiveNovelPath != "" {
		loadActiveNovel()
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

// setupSignalHandler registers handlers for interrupt signals (Ctrl+C) and termination signals.
func setupSignalHandler() {
	sigs := make(chan os.Signal, 1)
	// Register the channel to receive notifications for SIGINT and SIGTERM
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle received signals
	go func() {
		sig := <-sigs // Wait for a signal
		fmt.Printf("\nReceived signal: %s. Exiting...\n", sig)
		if configDirty {
			fmt.Println("Configuration changed, saving before exit...")
			saveConfig() // Save if configuration is dirty
		}
		os.Exit(0) // Exit gracefully after handling the signal
	}()
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
		configDirty = true // Mark config as dirty
		// saveConfig() // Removed immediate save
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

	newNovelInfo := &config.NovelInfo{
		FilePath:             filePath,
		Chapters:             parsedChapters,
		ChapterTitles:        chapterTitles,
		LastReadChapterIndex: 0,
		LastReadSegmentIndex: 0,
		DetectedRegex:        detectedRegexName,
	}

	cfg.Novels[filePath] = newNovelInfo
	cfg.ActiveNovelPath = filePath
	activeNovel = newNovelInfo
	configDirty = true // Mark config as dirty
	// saveConfig() // Removed immediate save
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
		fmt.Printf(" %s %d: %s (%d chapters, last read: Ch %d, Seg %d)\n",
			activeMarker, i+1, filepath.Base(novelInfo.FilePath), len(novelInfo.ChapterTitles),
			novelInfo.LastReadChapterIndex+1, novelInfo.LastReadSegmentIndex)
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

	delete(cfg.Novels, filePath)
	fmt.Printf("Removed novel %d: %s\n", index, filepath.Base(filePath))

	if cfg.ActiveNovelPath == filePath {
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		fmt.Println("The active novel was removed.")
	}
	configDirty = true // Mark config as dirty
	// saveConfig() // Removed immediate save
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

	if cfg.ActiveNovelPath != filePath { // Only mark dirty if actually switching
		// Save progress for the *previous* active novel before switching
		if configDirty {
			fmt.Println("Saving progress for previous novel before switching...")
			saveConfig() // Save immediately if dirty
		}

		cfg.ActiveNovelPath = filePath
		activeNovel = novelToSwitch
		loadActiveNovelChapters()
		configDirty = true // Mark config as dirty because ActiveNovelPath changed
		saveConfig()       // Save immediately after switching to persist the new active path
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
	loadActiveNovelChapters()
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

	targetChapterIndex := activeNovel.LastReadChapterIndex
	startSegmentIndex := activeNovel.LastReadSegmentIndex
	chapterChanged := false // Flag to track if chapter index changed

	if len(args) > 0 {
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 1 || idx > len(activeNovel.Chapters) {
			log.Fatalf("Error: Invalid chapter index '%s'. Please provide a number between 1 and %d.", args[0], len(activeNovel.Chapters))
		}
		newChapterIndex := idx - 1
		if newChapterIndex != targetChapterIndex {
			targetChapterIndex = newChapterIndex
			startSegmentIndex = 0 // Start from segment 0 when chapter is specified
			chapterChanged = true
		}
	}

	// --- Immediate Save on Chapter Change ---
	if chapterChanged {
		fmt.Printf("Switching to Chapter %d, saving progress...\n", targetChapterIndex+1)
		activeNovel.LastReadChapterIndex = targetChapterIndex
		activeNovel.LastReadSegmentIndex = startSegmentIndex // Should be 0 here
		configDirty = true
		saveConfig() // Save immediately
	}
	// --- End Immediate Save ---

	if targetChapterIndex < 0 || targetChapterIndex >= len(activeNovel.Chapters) {
		fmt.Printf("No chapter selected or last read chapter index (%d) is invalid. Reading first chapter.\n", activeNovel.LastReadChapterIndex+1)
		targetChapterIndex = 0
		startSegmentIndex = 0
		// If we defaulted to chapter 0, save this change immediately too
		if activeNovel.LastReadChapterIndex != 0 || activeNovel.LastReadSegmentIndex != 0 {
			activeNovel.LastReadChapterIndex = 0
			activeNovel.LastReadSegmentIndex = 0
			configDirty = true
			saveConfig()
		}
	}

	chapter := activeNovel.Chapters[targetChapterIndex]
	fmt.Printf("--- Reading Chapter %d: %s ---\n", targetChapterIndex+1, chapter.Title)

	segments := segmentSeparator.Split(chapter.Content, -1)
	if len(segments) == 0 {
		fmt.Println("Chapter content appears empty or has no segments.")
		return
	}

	// Adjust startSegmentIndex again just in case it was invalid from config
	if startSegmentIndex < 0 || startSegmentIndex >= len(segments) {
		fmt.Printf("Warning: Last read segment index (%d) is invalid for this chapter. Starting from segment 0.\n", startSegmentIndex)
		startSegmentIndex = 0
		// Save the corrected segment index immediately if it changed
		if activeNovel.LastReadSegmentIndex != 0 {
			activeNovel.LastReadSegmentIndex = 0
			configDirty = true
			saveConfig()
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
			return // Stop reading on TTS start error
		}

		// Update progress in memory *before* waiting for speech
		// Only mark dirty, actual save happens on exit/signal/chapter change
		if activeNovel.LastReadChapterIndex != targetChapterIndex || activeNovel.LastReadSegmentIndex != segIdx {
			activeNovel.LastReadChapterIndex = targetChapterIndex
			activeNovel.LastReadSegmentIndex = segIdx
			configDirty = true // Mark config as dirty only if progress changed
		}
		// saveConfig() // Removed immediate save per segment

		fmt.Println("(Speaking...)")
		err = <-doneChan // Block until speaking is done or an error occurs

		if err != nil {
			log.Printf("Error during TTS for Ch %d, Seg %d: %v", targetChapterIndex+1, segIdx, err)
			// Don't exit the whole program, just stop reading this chapter
			return
		}
		fmt.Println("(Segment finished)")

		// If auto-next is disabled, stop after this segment
		if !cfg.AutoReadNext {
			fmt.Println("Auto-next disabled. Stopping.")
			return
		}

		// If it was the last segment of the chapter, break the segment loop
		// to trigger auto-next chapter logic below
		if segIdx == len(segments)-1 {
			break
		}
		// Otherwise, the loop continues to the next segment
	}

	// --- Handle Auto-Next Chapter ---
	// This code runs if the segment loop completed (all segments read) AND auto-next is enabled
	if cfg.AutoReadNext {
		fmt.Println("Chapter finished. Auto-reading next chapter...")
		nextChapterIndexInternal := targetChapterIndex + 1
		if nextChapterIndexInternal < len(activeNovel.Chapters) {
			// Call handleRead for the next chapter. handleRead will handle the immediate save.
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

	nextChapterIndex := activeNovel.LastReadChapterIndex + 1
	if nextChapterIndex >= len(activeNovel.Chapters) {
		fmt.Println("Already at the last chapter.")
		return
	}
	// Call handleRead, which will detect the chapter change and save immediately.
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

	prevChapterIndex := activeNovel.LastReadChapterIndex - 1
	if prevChapterIndex < 0 {
		fmt.Println("Already at the first chapter.")
		return
	}
	// Call handleRead, which will detect the chapter change and save immediately.
	handleRead([]string{strconv.Itoa(prevChapterIndex + 1)})
}

func handleWhere() {
	if cfg.ActiveNovelPath == "" || activeNovel == nil {
		fmt.Println("No novel is currently active.")
		return
	}
	lastChapIdx := activeNovel.LastReadChapterIndex
	lastSegIdx := activeNovel.LastReadSegmentIndex
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

func loadActiveNovel() {
	info, exists := cfg.Novels[cfg.ActiveNovelPath]
	if !exists {
		fmt.Fprintf(os.Stderr, "Warning: Active novel path '%s' not found in library. Clearing active novel.\n", cfg.ActiveNovelPath)
		cfg.ActiveNovelPath = ""
		activeNovel = nil
		configDirty = true // Mark config as dirty since ActiveNovelPath changed
		// saveConfig() // Removed immediate save - rely on signal/defer
		return
	}
	activeNovel = info
}

func loadActiveNovelChapters() {
	if activeNovel == nil || activeNovel.FilePath == "" {
		return
	}
	if len(activeNovel.Chapters) > 0 && len(activeNovel.Chapters) == len(activeNovel.ChapterTitles) {
		return // Already loaded
	}

	fmt.Printf("Loading chapters for: %s\n", activeNovel.FilePath)
	if _, err := os.Stat(activeNovel.FilePath); os.IsNotExist(err) {
		log.Printf("Error: File for active novel not found: %s", activeNovel.FilePath)
		activeNovel.Chapters = nil
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

	activeNovel.Chapters = parsedChapters
	activeNovel.ChapterTitles = make([]string, len(parsedChapters))
	for i, ch := range parsedChapters {
		activeNovel.ChapterTitles[i] = ch.Title
	}

	fmt.Printf("Loaded %d chapters.\n", len(activeNovel.Chapters))
}

func saveConfig() {
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
		log.Printf("Error saving config to %s: %v", configPath, err)
	} else {
		fmt.Println("Configuration saved.")
		configDirty = false // Reset dirty flag after successful save
	}
}
