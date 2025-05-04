# Go Novel Reader (go-say) - Development Plan

## Project Goals

1.  **Chapter Splitting:** Automatically identify and split chapters in Markdown or TXT novel files using format detection.
2.  **TTS Reading:** Utilize macOS's built-in `say` command to read selected chapters aloud.
3.  **Multi-Novel Library:** Manage a library of multiple novels.
4.  **Individual Progress Saving:** Keep track of the last successfully read **chapter and segment (paragraph)** for each novel in the library.
5.  **Active Novel:** Maintain the concept of a currently "active" novel for reading commands.
6.  **Auto-Next Chapter:** Optionally configure the application to automatically read the next chapter upon completion of the current one.

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
    └── config.go     # Defines config/progress structs, load/save logic
```
(Removed manager.go as load/save logic is now within config.go)

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

3.  **Configuration and Progress Management (`config/config.go`):**
    *   Define structs:
        ```go
        // Holds metadata (less frequently changed)
        type NovelInfo struct {
            FilePath      string          `json:"file_path"`
            Chapters      []novel.Chapter `json:"-"`
            ChapterTitles []string        `json:"chapter_titles"`
            DetectedRegex string          `json:"detected_regex,omitempty"`
        }
        // Main application config (less frequently changed)
        type AppConfig struct {
            Novels          map[string]*NovelInfo `json:"novels"`
            ActiveNovelPath string                `json:"active_novel_path"`
            AutoReadNext    bool                  `json:"auto_read_next,omitempty"`
        }
        // Holds progress data (frequently changed)
        type ProgressInfo struct {
            LastReadChapterIndex int `json:"last_read_chapter_index"`
            LastReadSegmentIndex int `json:"last_read_segment_index"`
        }
        // Map FilePath -> ProgressInfo
        type ProgressData map[string]*ProgressInfo
        ```
    *   Use two separate files:
        *   `config.json`: Stores `AppConfig` (novel list, active novel path, settings). Saved less frequently (on exit/signal if dirty, on `switch`).
        *   `progress.json`: Stores `ProgressData` (chapter/segment progress for each novel). Saved more frequently (on exit/signal if dirty, on chapter change, periodically).
    *   Load both files on startup. Update relevant data structures in memory. Use dirty flags (`configDirty`, `progressDirty`) to track changes.

4.  **Main Program Logic (`main.go`):**
    *   Design the CLI commands:
        *   `add <filepath>`: Add a novel to the library, parse chapters, and set as active.
        *   `list`: List all novels in the library with their index and last read chapter/segment, marking the active one.
        *   `remove <index>`: Remove the novel at the specified index (from `list`) from the library.
        *   `switch <index>`: Set the novel at the specified index (from `list`) as active.
        *   `chapters`: List chapters of the currently active novel.
        *   `read [chap_index]`: Read a specific chapter (1-based) starting from its first segment, or continue from the last read chapter/segment if index is omitted. Reads segment by segment. If `AutoReadNext` is enabled, automatically proceed to the next segment/chapter upon completion.
        *   `next`/`prev`: Read the next/previous chapter of the active novel, starting from its first segment.
        *   `where`: Show the active novel and its last read chapter and segment.
        *   `config [setting]`: View or toggle configuration settings (currently `auto_next`).
    *   Handle user input, manage the active novel state, load chapters as needed, and orchestrate calls to other modules. Implement segment-based reading loop in `read` with asynchronous TTS. Update progress in `progressData` and mark `progressDirty`. Update settings/active path in `cfg` and mark `configDirty`. Implement signal handling and deferred exit to save both files if dirty. Implement immediate saving of `progress.json` on chapter change/periodic timer, and `config.json` on novel switch.

## Development Steps (Completed for v8)

1.  Initialize Go module.
2.  Separate configuration (`AppConfig`, `NovelInfo` in `config.json`) and progress data (`ProgressData`, `ProgressInfo` in `progress.json`). Implement load/save logic for both.
3.  Implement chapter splitter with automatic format detection (`novel/parser.go`).
4.  Implement asynchronous TTS interface (`tts/speaker.go` with `SpeakAsync`).
5.  Build CLI interaction logic in `main.go` using separate config/progress data, supporting multi-novel management, segment-level progress tracking, `config` command, and auto-next feature.
6.  Implement robust saving logic for both files based on dirty flags and specific events (exit, signal, chapter change, periodic, novel switch).
7.  Add error handling and user feedback.

## Information Needed (Resolved)

*   Common chapter title formats were identified (Chinese, English, Markdown) and automatic detection was implemented.
