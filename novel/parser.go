package novel

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

// Chapter represents a single chapter of the novel.
type Chapter struct {
	Title   string
	Content string
}

// Regex candidates for chapter detection
var chapterRegexes = map[string]*regexp.Regexp{
	"chinese":  regexp.MustCompile(`^\s*第\s*[一二三四五六七八九十百千万零〇\d]+\s*[章卷节回].*$`),
	"english":  regexp.MustCompile(`^\s*Chapter\s+\d+.*$`),
	"markdown": regexp.MustCompile(`^\s*#{1,6}\s+.*$`),
}

const detectBufferSize = 1 * 1024 * 1024 // 1MB for format detection

// DetectFormat attempts to automatically detect the chapter title format.
func DetectFormat(filePath string) (*regexp.Regexp, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, detectBufferSize)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	contentSample := string(buffer[:n])

	scores := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(contentSample))
	for scanner.Scan() {
		line := scanner.Text()
		for format, re := range chapterRegexes {
			if re.MatchString(line) {
				scores[format]++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	bestFormat := ""
	maxScore := 0
	for format, score := range scores {
		// Basic heuristic: require at least a few matches to be confident
		if score > maxScore && score > 1 {
			maxScore = score
			bestFormat = format
		}
	}

	if bestFormat == "" {
		// Default or fallback if no clear winner - let's default to markdown for now
		// Alternatively, could return an error asking user to specify
		// return nil, errors.New("could not reliably detect chapter format")
		return chapterRegexes["markdown"], nil // Defaulting to Markdown
	}

	return chapterRegexes[bestFormat], nil
}

// ParseNovel reads a novel file and splits it into chapters based on the provided regex.
func ParseNovel(filePath string, chapterRegex *regexp.Regexp) ([]Chapter, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chapters []Chapter
	var currentContent strings.Builder
	var currentTitle string

	scanner := bufio.NewScanner(file)
	firstChapter := true

	for scanner.Scan() {
		line := scanner.Text()
		if chapterRegex.MatchString(line) {
			// Found a new chapter title
			if !firstChapter {
				// Save the previous chapter's content
				chapters = append(chapters, Chapter{
					Title:   strings.TrimSpace(currentTitle),
					Content: strings.TrimSpace(currentContent.String()),
				})
			}
			// Start new chapter
			currentTitle = line
			currentContent.Reset()
			firstChapter = false
		} else {
			// Append line to current chapter content
			if !firstChapter { // Don't add content before the first chapter title
				currentContent.WriteString(line)
				currentContent.WriteString("\n") // Add newline back
			}
		}
	}

	// Add the last chapter
	if !firstChapter {
		chapters = append(chapters, Chapter{
			Title:   strings.TrimSpace(currentTitle),
			Content: strings.TrimSpace(currentContent.String()),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(chapters) == 0 {
		return nil, errors.New("no chapters found using the detected format")
	}

	return chapters, nil
}
