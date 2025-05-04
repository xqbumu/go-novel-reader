# Go Novel Reader (go-say) - Development Plan

## Project Goals

1.  **Chapter Splitting:** Automatically identify and split chapters in Markdown or TXT novel files using format detection.
2.  **TTS Reading:** Utilize macOS's built-in `say` command to read selected chapters aloud.
3.  **Multi-Novel Library:** Manage a library of multiple novels.
4.  **Individual Progress Saving:** Keep track of the last read chapter for each novel in the library.
5.  **Active Novel:** Maintain the concept of a currently "active" novel for reading commands.

## Technical Choices

*   **Language:** Go
*   **Core Libraries:**
    *   `os/exec`: To execute the `say` command.
    *   `regexp`: For identifying chapter titles.
    *   `encoding/json`: For handling the configuration file.
    *   `flag` or `github.com/spf13/cobra`: For building the command-line interface (CLI).

## Proposed Project Structure

```
go-say/
├── go.mod
├── main.go           # Entry point, handles CLI arguments and interaction
├── novel/
│   └── parser.go     # Reads files, splits into chapters
├── tts/
│   └── speaker.go    # Interfaces with macOS 'say' command
└── config/
    ├── manager.go    # Loads and saves configuration
    └── config.go     # Defines the configuration struct
```

## Core Feature Implementation Ideas

1.  **Chapter Splitting (`novel/parser.go`):**
    *   **Automatic Format Detection:**
        *   Read the first 1MB of the file content.
        *   Define candidate regular expressions for common formats:
            *   Chinese: `^\s*第\s*[一二三四五六七八九十百千万零〇\d]+\s*[章卷节回].*$`
            *   English: `^\s*Chapter\s+\d+.*$`
            *   Markdown: `^\s*#{1,6}\s+.*$`
        *   Analyze the initial content line by line, counting matches for each regex.
        *   Select the regex with the most matches as the detected format for the file. Handle potential ambiguities (e.g., few matches, ties) by defaulting or prompting.
    *   **Full File Splitting:**
        *   Using the detected regex, read the entire file again.
        *   Split the content into chapters, extracting both the title and the body for each chapter.

2.  **TTS Reading (`tts/speaker.go`):**
    *   Accept chapter text as input.
    *   Use `exec.Command("say", chapterText).Run()` to invoke system TTS.
    *   Consider handling potential interruption of reading (e.g., user wants to stop).

3.  **Configuration Management (`config/config.go`):**
    *   Define structs for configuration:
        ```go
        // Represents a single novel's metadata and progress
        type NovelInfo struct {
            FilePath      string          `json:"file_path"`
            Chapters      []novel.Chapter `json:"-"` // Loaded in memory, not saved
            ChapterTitles []string        `json:"chapter_titles"` // Saved for listing
            LastReadIndex int             `json:"last_read_index"`
            DetectedRegex string          `json:"detected_regex,omitempty"`
        }

        // Main application configuration
        type AppConfig struct {
            Novels          map[string]*NovelInfo `json:"novels"` // Map FilePath -> NovelInfo
            ActiveNovelPath string                `json:"active_novel_path"`
        }
        ```
    *   Save configuration as a JSON file (e.g., `~/.config/go-say/config.json`).
    *   Load config on startup. Save config whenever the library or progress changes.

4.  **Main Program Logic (`main.go`):**
    *   Design the CLI commands:
        *   `add <filepath>`: Add a novel to the library, parse chapters, and set as active.
        *   `list`: List all novels in the library with their index, marking the active one.
        *   `remove <index>`: Remove the novel at the specified index (from `list`) from the library.
        *   `switch <index>`: Set the novel at the specified index (from `list`) as active.
        *   `chapters`: List chapters of the currently active novel.
        *   `read [chap_index]`: Read a specific chapter (1-based) of the active novel, or continue from the last read position if index is omitted.
        *   `next`/`prev`: Read the next/previous chapter of the active novel.
        *   `where`: Show the active novel and its last read chapter.
    *   Handle user input, manage the active novel state, load chapters as needed, and orchestrate calls to other modules.

## Development Steps (Completed for v2)

1.  Initialize Go module.
2.  Implement configuration structures (`AppConfig`, `NovelInfo`) and load/save logic.
3.  Implement chapter splitter with automatic format detection (`novel/parser.go`).
4.  Implement TTS interface (`tts/speaker.go`).
5.  Build CLI interaction logic in `main.go` supporting multi-novel management and individual progress tracking.
6.  Add error handling and user feedback.

## Information Needed (Resolved)

*   Common chapter title formats were identified (Chinese, English, Markdown) and automatic detection was implemented.
