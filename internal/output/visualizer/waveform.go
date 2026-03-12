package visualizer

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 16
	maxWidth  = 36
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
	columnWidth  int  // if > 0, target terminal column width for two-row rendering (2 samples per column)
	playhead     int  // sample index of current playback position
	progressMode bool // true once SetProgress has been called; enables bright/dim split in RenderSingle

	cachedTop         string
	cachedBot         string
	cachedSingle      string
	cachedPlayhead    int
	cachedSampleCount int
}

// NewWaveform creates a waveform buffer sized to the given width.
// Width is clamped between minWidth and maxWidth.
func NewWaveform(width int) *Waveform {
	if width > maxWidth {
		width = maxWidth
	}
	if width < minWidth {
		width = minWidth
	}
	return &Waveform{
		samples:      make([]float64, 0),
		maxSamples:   width,
		displayWidth: width,
		playhead:     0,
	}
}

// NewWaveformTwoRow creates a waveform for two-row braille rendering.
func NewWaveformTwoRow(columns int) *Waveform {
	if columns < minWidth {
		columns = minWidth
	}
	return &Waveform{
		samples:      make([]float64, 0),
		maxSamples:   columns * 2,
		displayWidth: columns * 2,
		columnWidth:  columns,
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
	w.progressMode = false
	w.invalidateCache()

	charCount := len(w.samples)
	if charCount > w.displayWidth && w.displayWidth < w.maxSamples {
		newWidth := charCount
		if newWidth > w.maxSamples {
			newWidth = w.maxSamples
		}
		w.displayWidth = newWidth
	}
}

// SetProgress sets playhead based on progress (0.0-1.0) and enables the animated
// bright/dim split in RenderSingle.
func (w *Waveform) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	w.progressMode = true
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

// RenderSingle returns a single-row waveform using sqrt amplitude scaling.
// When SetProgress has been called, played samples render bright and upcoming samples dim.
func (w *Waveform) RenderSingle() string {
	if w.cachedSingle != "" && w.isCacheValid() {
		return w.cachedSingle
	}
	w.cachedSingle = w.renderSingleLine(math.Sqrt)
	w.cachedPlayhead = w.playhead
	w.cachedSampleCount = len(w.samples)
	return w.cachedSingle
}

// Width returns the display width of the waveform in terminal columns.
func (w *Waveform) Width() int {
	return w.displayWidth
}

func (w *Waveform) renderRowWithPlayhead(charFunc func(int, int) rune) string {
	if len(w.samples) == 0 {
		return ""
	}

	charCount := (len(w.samples) + 1) / 2
	var b strings.Builder
	b.Grow(charCount * 8)

	var brightChunk strings.Builder

	for i := 0; i < len(w.samples); i += 2 {
		left := QuantizeAmplitudeSqrt(w.samples[i])
		right := 0
		if i+1 < len(w.samples) {
			right = QuantizeAmplitudeSqrt(w.samples[i+1])
		}
		ch := string(charFunc(left, right))
		brightChunk.WriteString(ch)
	}

	if brightChunk.Len() > 0 {
		b.WriteString(brightStyle.Render(brightChunk.String()))
	}

	targetCols := w.displayWidth
	if w.columnWidth > 0 {
		targetCols = w.columnWidth
	}
	if charCount < targetCols {
		padding := strings.Repeat("⠀", targetCols-charCount)
		b.WriteString(dimStyle.Render(padding))
	}

	return b.String()
}

func (w *Waveform) renderSingleLine(transform func(float64) float64) string {
	if len(w.samples) == 0 {
		return ""
	}

	charCount := len(w.samples)
	var b strings.Builder
	b.Grow(charCount * 4)

	var brightChunk strings.Builder
	var dimChunk strings.Builder

	flushBright := func() {
		if brightChunk.Len() > 0 {
			b.WriteString(brightStyle.Render(brightChunk.String()))
			brightChunk.Reset()
		}
	}
	flushDim := func() {
		if dimChunk.Len() > 0 {
			b.WriteString(dimStyle.Render(dimChunk.String()))
			dimChunk.Reset()
		}
	}

	for i := 0; i < len(w.samples); i++ {
		v := transform(w.samples[i])
		var level int
		switch {
		case v < 0.15:
			level = 0
		case v < 0.4:
			level = 1
		case v < 0.5:
			level = 2
		default:
			level = 3
		}
		ch := string(SingleLineChar(level))
		if w.progressMode && i >= w.playhead {
			flushBright()
			dimChunk.WriteString(ch)
		} else {
			flushDim()
			brightChunk.WriteString(ch)
		}
	}
	flushBright()
	flushDim()

	if charCount < w.displayWidth {
		padding := strings.Repeat(" ", w.displayWidth-charCount)
		b.WriteString(dimStyle.Render(padding))
	}

	return b.String()
}

func (w *Waveform) invalidateCache() {
	w.cachedTop = ""
	w.cachedBot = ""
	w.cachedSingle = ""
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
	}
	w.samples = append(w.samples, amp)
	w.invalidateCache()

	contentChars := len(w.samples)
	maxWidth := w.maxSamples

	neededWidth := contentChars + lookAhead
	if neededWidth > w.displayWidth && w.displayWidth < maxWidth {
		newWidth := w.displayWidth + growChunk
		if newWidth > maxWidth {
			newWidth = maxWidth
		}
		w.displayWidth = newWidth
		w.invalidateCache()
	}
}
