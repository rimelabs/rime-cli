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
	transcriptBrightStyle = lipgloss.NewStyle()
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

func (t *Transcript) revealCount() int {
	n := len(t.words)
	if t.duration > 0 && t.elapsed >= 0 {
		progress := float64(t.elapsed) / float64(t.duration)
		if progress > 1.0 {
			progress = 1.0
		}
		n = int(float64(len(t.words)) * progress)
		if n > len(t.words) {
			n = len(t.words)
		}
	}
	return n
}

// RenderSingleLine renders the transcript as a single line of visible width
// availableWidth using two modes:
//
//   - During animation (not all words revealed): a scrolling window anchored
//     at availableWidth/3 from the left, showing recently-spoken words on the
//     left and upcoming words on the right.
//
//   - At end of playback (all words revealed): middle truncation showing the
//     beginning and end of the full text, e.g. "the quick fox ... the lazy dog".
func (t *Transcript) RenderSingleLine(availableWidth int) string {
	if len(t.words) == 0 || availableWidth <= 0 {
		return ""
	}

	rc := t.revealCount()

	if rc >= len(t.words) {
		return t.renderMiddleTruncated(availableWidth)
	}
	return t.renderScrolling(availableWidth, rc)
}

// renderScrolling renders a sliding window anchored at availableWidth/3.
// Bright (spoken) words fill the left slot, dim (upcoming) words the right.
func (t *Transcript) renderScrolling(availableWidth, rc int) string {
	anchor := availableWidth / 3
	brightWords := collectWordsRev(t.words[:rc], anchor)
	usedLeft := wordsWidth(brightWords)

	sep := 0
	if usedLeft > 0 {
		sep = 1
	}
	dimWords := collectWordsFwd(t.words[rc:], availableWidth-anchor-sep)

	var b strings.Builder
	if pad := anchor - usedLeft; pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}
	for i, w := range brightWords {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(transcriptBrightStyle.Render(w))
	}
	if usedLeft > 0 && len(dimWords) > 0 {
		b.WriteByte(' ')
	}
	for i, w := range dimWords {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(transcriptDimStyle.Render(w))
	}
	return b.String()
}

// renderMiddleTruncated shows the full text if it fits, otherwise shows
// beginning words + " ... " + end words (all bright, since fully revealed).
func (t *Transcript) renderMiddleTruncated(availableWidth int) string {
	if wordsWidth(t.words) <= availableWidth {
		var b strings.Builder
		for i, w := range t.words {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(transcriptBrightStyle.Render(w))
		}
		return b.String()
	}

	const sepStr = " ... "
	const sepWidth = 5
	if availableWidth <= sepWidth {
		return transcriptDimStyle.Render(sepStr[:availableWidth])
	}

	remaining := availableWidth - sepWidth
	beginWidth := remaining / 2
	endWidth := remaining - beginWidth

	beginWords := collectWordsFwd(t.words, beginWidth)
	endWords := collectWordsRev(t.words, endWidth)
	endStartIdx := len(t.words) - len(endWords)
	if endStartIdx < len(beginWords) {
		beginWords = beginWords[:endStartIdx]
	}

	var b strings.Builder
	for i, w := range beginWords {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(transcriptBrightStyle.Render(w))
	}
	b.WriteString(transcriptDimStyle.Render(sepStr))
	for i, w := range endWords {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(transcriptBrightStyle.Render(w))
	}
	return b.String()
}

// wordsWidth returns the total visible width of words joined by single spaces.
func wordsWidth(words []string) int {
	total := 0
	for i, w := range words {
		if i > 0 {
			total++
		}
		total += lipgloss.Width(w)
	}
	return total
}

// collectWordsFwd collects words from the start of words that fit within budget chars.
func collectWordsFwd(words []string, budget int) []string {
	var result []string
	used := 0
	for _, w := range words {
		need := lipgloss.Width(w)
		if used > 0 {
			need++
		}
		if used+need > budget {
			break
		}
		result = append(result, w)
		used += need
	}
	return result
}

// collectWordsRev collects words from the end of words that fit within budget chars,
// returning them in their original (left-to-right) order.
func collectWordsRev(words []string, budget int) []string {
	var rev []string
	used := 0
	for i := len(words) - 1; i >= 0; i-- {
		w := words[i]
		need := lipgloss.Width(w)
		if used > 0 {
			need++
		}
		if used+need > budget {
			break
		}
		rev = append(rev, w)
		used += need
	}
	result := make([]string, len(rev))
	for i, w := range rev {
		result[len(rev)-1-i] = w
	}
	return result
}

func (t *Transcript) Render() string {
	if len(t.words) == 0 {
		return ""
	}

	revealCount := t.revealCount()
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
