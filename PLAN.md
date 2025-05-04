# Go Novel Reader (go-say) - Development Plan

## Project Goals

1.  **Chapter Splitting:** Automatically identify and split chapters in Markdown or TXT novel files.
2.  **TTS Reading:** Utilize macOS's built-in `say` command to read selected chapters aloud.
3.  **Progress Saving:** Keep track of the last opened novel and the last read chapter.

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

3.  **Configuration Management (`config/manager.go`, `config/config.go`):**
    *   Define a struct for configuration:
        ```go
        type AppConfig struct {
            LastNovelPath     string `json:"last_novel_path"`
            LastChapterIndex int    `json:"last_chapter_index"` // Or maybe chapter title/identifier
        }
        ```
    *   Save configuration as a JSON file (e.g., `~/.config/go-say/config.json`).
    *   Load config on startup, update and save when switching books or chapters.

4.  **Main Program Logic (`main.go`):**
    *   Design the CLI commands:
        *   `go-say open <filepath>`: Open a new novel.
        *   `go-say list`: Display chapter list for selection.
        *   `go-say read [chapter_number]`: Read a specific chapter (or the last read one).
        *   `go-say next`/`prev`: Read the next/previous chapter.
        *   `go-say where`: Show current reading progress.
    *   Handle user input and orchestrate calls to other modules.

## Development Steps

1.  Initialize the Go module (`go mod init github.com/xqbumu/go-say`).
2.  Implement configuration reading/writing.
3.  **Implement the chapter splitter based on the provided chapter format.**
4.  Implement the TTS interface.
5.  Build the CLI interaction logic.
6.  Add error handling and user feedback.

## Information Needed

*   **What is the common format for chapter titles in your novels?** (e.g., `第X章 标题`, `Chapter X`, `# Title`) This is needed to implement the chapter splitting correctly.
