package visualizer

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	defaultWordsPerMinute = 150
	minTextWidth          = 20
	maxTextWidth          = 80
)

var (
	transcriptDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	transcriptBrightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
)

type Transcript struct {
	words    []string
	duration time.Duration
	elapsed  time.Duration
	maxWidth int

	cachedOutput      string
	cachedRevealCount int
}

func NewTranscript(text string, duration time.Duration) *Transcript {
	words := strings.Fields(text)
	width := getTerminalWidth()
	if width < minTextWidth {
		width = maxTextWidth
	}
	textWidth := width
	if textWidth > maxTextWidth {
		textWidth = maxTextWidth
	}
	return &Transcript{
		words:    words,
		duration: duration,
		maxWidth: textWidth,
	}
}

func (t *Transcript) SetDuration(duration time.Duration) {
	if t.duration != duration {
		t.duration = duration
		t.invalidateCache()
	}
}

func (t *Transcript) SetElapsed(elapsed time.Duration) {
	t.elapsed = elapsed
}

func (t *Transcript) Render() string {
	if len(t.words) == 0 {
		return ""
	}

	revealCount := len(t.words)
	if t.duration > 0 && t.elapsed >= 0 {
		progress := float64(t.elapsed) / float64(t.duration)
		if progress > 1.0 {
			progress = 1.0
		}
		revealCount = int(float64(len(t.words)) * progress)
		if revealCount > len(t.words) {
			revealCount = len(t.words)
		}
	}

	if t.cachedOutput != "" && t.cachedRevealCount == revealCount {
		return t.cachedOutput
	}

	t.cachedOutput = t.wrapWords(revealCount)
	t.cachedRevealCount = revealCount
	return t.cachedOutput
}

func (t *Transcript) invalidateCache() {
	t.cachedOutput = ""
	t.cachedRevealCount = -1
}

func (t *Transcript) wrapWords(revealCount int) string {
	if t.maxWidth < minTextWidth {
		t.maxWidth = maxTextWidth
	}

	if len(t.words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder
	var brightChunk strings.Builder
	var dimChunk strings.Builder
	lineLen := 0

	for i, word := range t.words {
		wordLen := len(word)
		needsSpace := currentLine.Len() > 0 || brightChunk.Len() > 0 || dimChunk.Len() > 0

		spaceLen := 0
		if needsSpace {
			spaceLen = 1
		}

		if lineLen+spaceLen+wordLen > t.maxWidth && (currentLine.Len() > 0 || brightChunk.Len() > 0 || dimChunk.Len() > 0) {
			if brightChunk.Len() > 0 {
				currentLine.WriteString(transcriptBrightStyle.Render(brightChunk.String()))
				brightChunk.Reset()
			}
			if dimChunk.Len() > 0 {
				currentLine.WriteString(transcriptDimStyle.Render(dimChunk.String()))
				dimChunk.Reset()
			}
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			lineLen = 0
			needsSpace = false
		}

		if needsSpace {
			if i < revealCount {
				if dimChunk.Len() > 0 {
					currentLine.WriteString(transcriptDimStyle.Render(dimChunk.String()))
					dimChunk.Reset()
				}
				brightChunk.WriteString(" ")
				brightChunk.WriteString(word)
			} else {
				if brightChunk.Len() > 0 {
					currentLine.WriteString(transcriptBrightStyle.Render(brightChunk.String()))
					brightChunk.Reset()
				}
				dimChunk.WriteString(" ")
				dimChunk.WriteString(word)
			}
			lineLen += spaceLen + wordLen
		} else {
			if i < revealCount {
				if dimChunk.Len() > 0 {
					currentLine.WriteString(transcriptDimStyle.Render(dimChunk.String()))
					dimChunk.Reset()
				}
				brightChunk.WriteString(word)
			} else {
				if brightChunk.Len() > 0 {
					currentLine.WriteString(transcriptBrightStyle.Render(brightChunk.String()))
					brightChunk.Reset()
				}
				dimChunk.WriteString(word)
			}
			lineLen += wordLen
		}
	}

	if brightChunk.Len() > 0 {
		currentLine.WriteString(transcriptBrightStyle.Render(brightChunk.String()))
	}
	if dimChunk.Len() > 0 {
		currentLine.WriteString(transcriptDimStyle.Render(dimChunk.String()))
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

func EstimateDurationFromText(text string) time.Duration {
	words := strings.Fields(text)
	wordCount := len(words)
	if wordCount == 0 {
		return 0
	}
	estimatedMinutes := float64(wordCount) / defaultWordsPerMinute
	return time.Duration(estimatedMinutes * float64(time.Minute))
}
