package novel

import (
	"bufio"
	"errors"
	"fmt" // Ensure fmt is imported
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

// ChapterRegexes holds the candidate regular expressions for chapter detection.
// It is exported so it can be potentially used or referenced by other packages (like main).
var ChapterRegexes = map[string]*regexp.Regexp{
	"chinese":  regexp.MustCompile(`^\s*第\s*[一二三四五六七八九十百千万零〇\d]+\s*[章卷节回].*$`),
	"english":  regexp.MustCompile(`^\s*Chapter\s+\d+.*$`),
	"markdown": regexp.MustCompile(`^\s*#{1,6}\s+.*$`), // Matches markdown headers H1-H6
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
	buffer := make([]byte, detectBufferSize) // Read up to 1MB
	n, err := io.ReadFull(reader, buffer)
	// io.ReadFull returns io.ErrUnexpectedEOF if less than buffer size is read, which is expected for smaller files.
	// It returns io.EOF only if 0 bytes were read.
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	contentSample := string(buffer[:n])

	scores := make(map[string]int)
	// Use strings.Split is simpler for a fixed buffer than a scanner
	lines := strings.Split(contentSample, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line) // Trim whitespace for matching
		if trimmedLine == "" {
			continue
		}
		for format, re := range ChapterRegexes { // Use exported variable
			if re.MatchString(trimmedLine) { // Match against trimmed line
				scores[format]++
				// Optional: break inner loop if one format matches? Assumes titles are unique.
				// break
			}
		}
	}
	// Scanner error check is not needed when using strings.Split

	bestFormat := ""
	// Start with a minimum score threshold to avoid spurious matches on random lines
	maxScore := 1 // Require at least 2 matches to be considered
	for format, score := range scores {
		if score > maxScore {
			maxScore = score
			bestFormat = format
		} else if score == maxScore && score > 1 {
			// Handle ties? For now, first one wins or could prioritize (e.g. markdown)
			// Or maybe require a significantly higher score?
		}
	}

	if bestFormat == "" {
		// Default or fallback if no clear winner
		// Let's prioritize markdown if score is low, otherwise return error?
		if maxScore <= 1 { // If only 0 or 1 match found for the best format
			// Check if markdown has at least one match, prefer it as default
			if scores["markdown"] >= 1 {
				fmt.Println("Warning: Low confidence in format detection, defaulting to markdown.")
				return ChapterRegexes["markdown"], nil
			}
			// If even markdown doesn't match, return error
			return nil, errors.New("could not reliably detect chapter format, few or no chapter titles found in sample")
		}
		// If a best format was found (score > 1)
		fmt.Printf("Detected format '%s' with score %d\n", bestFormat, maxScore)
		return ChapterRegexes[bestFormat], nil
	}
	// This part should not be reachable if the logic above is correct,
	// but the compiler needs a return path.
	// If bestFormat is "", it means maxScore <= 1. The logic inside the if block handles this.
	// If somehow we exit the loop and bestFormat is set, we return it.
	// This path indicates successful detection.
	return ChapterRegexes[bestFormat], nil
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
