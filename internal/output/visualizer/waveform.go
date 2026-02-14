package visualizer

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	minWidth  = 48
	growChunk = 8
	lookAhead = 8
)

var (
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Waveform manages amplitude samples for visualization with playhead support.
type Waveform struct {
	samples      []float64
	maxSamples   int
	displayWidth int
	playhead     int // sample index of current playback position

	cachedTop         string
	cachedBot         string
	cachedPlayhead    int
	cachedSampleCount int
}

// NewWaveform creates a waveform buffer sized to terminal width.
func NewWaveform() *Waveform {
	width := getTerminalWidth()
	if width < minWidth {
		width = minWidth
	}
	return &Waveform{
		samples:      make([]float64, 0),
		maxSamples:   width * 2,
		displayWidth: width,
		playhead:     0,
	}
}

// SetSamples loads pre-analyzed amplitude samples.
func (w *Waveform) SetSamples(samples []float64) {
	if len(samples) > w.maxSamples {
		samples = samples[:w.maxSamples]
	}
	w.samples = samples
	w.playhead = 0
	w.invalidateCache()

	charCount := (len(w.samples) + 1) / 2
	if charCount > w.displayWidth {
		w.displayWidth = charCount
	}
}

// SetProgress sets playhead based on progress (0.0-1.0).
func (w *Waveform) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	newPlayhead := int(progress * float64(len(w.samples)))
	if newPlayhead != w.playhead {
		w.playhead = newPlayhead
		w.invalidateCache()
	}
}

// RenderTop returns the top row with dim/bright based on playhead.
func (w *Waveform) RenderTop() string {
	if w.cachedTop != "" && w.isCacheValid() {
		return w.cachedTop
	}
	w.cachedTop = w.renderRowWithPlayhead(TopChar)
	w.cachedPlayhead = w.playhead
	w.cachedSampleCount = len(w.samples)
	return w.cachedTop
}

// RenderBot returns the bottom row with dim/bright based on playhead.
func (w *Waveform) RenderBot() string {
	if w.cachedBot != "" && w.isCacheValid() {
		return w.cachedBot
	}
	w.cachedBot = w.renderRowWithPlayhead(BotChar)
	w.cachedPlayhead = w.playhead
	w.cachedSampleCount = len(w.samples)
	return w.cachedBot
}

func (w *Waveform) renderRowWithPlayhead(charFunc func(int, int) rune) string {
	if len(w.samples) == 0 {
		return ""
	}

	charCount := (len(w.samples) + 1) / 2
	var b strings.Builder
	b.Grow(charCount * 8)

	var brightChunk strings.Builder
	var dimChunk strings.Builder

	for i := 0; i < len(w.samples); i += 2 {
		left := QuantizeAmplitude(w.samples[i])
		right := 0
		if i+1 < len(w.samples) {
			right = QuantizeAmplitude(w.samples[i+1])
		}
		ch := string(charFunc(left, right))

		if i < w.playhead {
			if dimChunk.Len() > 0 {
				b.WriteString(dimStyle.Render(dimChunk.String()))
				dimChunk.Reset()
			}
			brightChunk.WriteString(ch)
		} else {
			if brightChunk.Len() > 0 {
				b.WriteString(brightStyle.Render(brightChunk.String()))
				brightChunk.Reset()
			}
			dimChunk.WriteString(ch)
		}
	}

	if brightChunk.Len() > 0 {
		b.WriteString(brightStyle.Render(brightChunk.String()))
	}
	if dimChunk.Len() > 0 {
		b.WriteString(dimStyle.Render(dimChunk.String()))
	}

	if charCount < w.displayWidth {
		padding := strings.Repeat("⠀", w.displayWidth-charCount)
		b.WriteString(dimStyle.Render(padding))
	}

	return b.String()
}

func (w *Waveform) invalidateCache() {
	w.cachedTop = ""
	w.cachedBot = ""
}

func (w *Waveform) isCacheValid() bool {
	return w.cachedPlayhead == w.playhead && w.cachedSampleCount == len(w.samples)
}

// AddSample adds an amplitude sample (for streaming mode).
func (w *Waveform) AddSample(amp float64) {
	if len(w.samples) >= w.maxSamples {
		w.samples = w.samples[1:]
		if w.playhead > 0 {
			w.playhead--
		}
		w.invalidateCache()
	}
	w.samples = append(w.samples, amp)
	w.invalidateCache()

	contentChars := (len(w.samples) + 1) / 2
	maxWidth := w.maxSamples / 2

	neededWidth := contentChars + (lookAhead+1)/2
	if neededWidth > w.displayWidth && w.displayWidth < maxWidth {
		newWidth := w.displayWidth + growChunk
		if newWidth > maxWidth {
			newWidth = maxWidth
		}
		w.displayWidth = newWidth
		w.invalidateCache()
	}
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 40 {
		return 80
	}
	return width
}
