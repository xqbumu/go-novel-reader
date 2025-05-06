# 📚 go-novel-reader: Your Personal Command-Line Novel Narrator!

[![Go Report Card](https://goreportcard.com/badge/github.com/xqbumu/go-novel-reader)](https://goreportcard.com/report/github.com/xqbumu/go-novel-reader)
[![Go Version](https://img.shields.io/github/go-mod/go-version/xqbumu/go-novel-reader)](https://golang.org/)
<!-- Add License Badge if applicable -->
<!-- [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) -->

[English](./README.md) | [中文](./README.zh-CN.md)

**Tired of staring at the screen to read novels? Let `go-novel-reader` read them aloud for you!**

This is a command-line tool written in Go that reads your locally stored novel files (TXT or Markdown format). It automatically identifies and splits chapters, utilizes macOS's built-in TTS (Text-to-Speech) to "tell the story," and remembers your reading progress for each novel, down to the paragraph!

## ✨ Features

*   **Multi-Novel Library**: Easily add, list, remove, and switch between your novel collection (`add`, `list`, `remove`, `switch`).
*   **Smart Chapter Splitting**: Automatically detects common chapter title formats (Chinese numerals, English "Chapter X", Markdown headers) and splits accordingly.
*   **Smooth TTS Reading**: Calls macOS's `say` command to read selected chapters segment by segment (`read`, `next`, `prev`).
*   **Precise Progress Saving**: Saves the last read chapter and segment index individually for each novel. Pick up right where you left off!
*   **Auto-Continue**: Optional configuration to automatically start the next segment/chapter after finishing the current one (`config auto_next`).
*   **Convenient Navigation**: Quickly check your current reading position and the chapter list (`where`, `chapters`).
*   **Cross-Platform? (macOS Only)**: Currently relies on the macOS `say` command, so it only supports macOS.

## 🖥️ Requirements

*   **macOS**: Required, due to dependency on the `say` command for TTS.
*   **Go**: Go compilation environment (e.g., Go 1.24 or later) needed for building.

## 🚀 Installation

**Option 1: Using `go install` (Recommended)**

If your Go environment is set up correctly (GOPATH, GOBIN in PATH), you can run:
```bash
go install github.com/xqbumu/go-novel-reader@latest
```
This will download, compile, and install the `go-novel-reader` executable into your `$GOPATH/bin` (or `$GOBIN`) directory.

**Option 2: Manual Build**

1.  Clone the repository:
    ```bash
    git clone https://github.com/xqbumu/go-novel-reader.git
    cd go-novel-reader
    ```
2.  Build:
    ```bash
    go build
    ```
    This generates an executable named `go-novel-reader` in the current directory. You can move this file to any location in your system's PATH for global access.

## 💡 Usage

```bash
# Add a new novel to the library and set it as active
./go-novel-reader add /path/to/your/novel.txt

# List all novels in the library and their progress
./go-novel-reader list

# Switch to the second novel in the library
./go-novel-reader switch 2

# List all chapters of the active novel
./go-novel-reader chapters

# Continue reading the active novel from where you left off
./go-novel-reader read

# Start reading the active novel from Chapter 5
./go-novel-reader read 5

# Read the next chapter of the active novel
./go-novel-reader next

# Read the previous chapter of the active novel
./go-novel-reader prev

# Show the active novel and current reading progress
./go-novel-reader where

# View or toggle configuration settings (e.g., auto-continue)
./go-novel-reader config          # View current config
./go-novel-reader config auto_next # Toggle the state of auto_next (true/false)

# Get help information
./go-novel-reader --help
```

## ⚙️ Configuration Files

`go-novel-reader` creates files in your user configuration directory to store information:

*   `~/.config/go-novel-reader/config.json`: Stores the library list, active novel path, and application settings (like `auto_next`).
*   `~/.config/go-novel-reader/progress.json`: Stores the reading progress for each novel (last read chapter and segment index).

You typically don't need to edit these files manually.

## 🔮 Future Ideas

*   Support for more TTS engines?
*   Cross-platform support? (Requires finding alternatives to `say`)
*   More configuration options?

Suggestions and contributions are welcome!

<!-- ## 📜 License

This project is licensed under the [MIT License](LICENSE). -->
