package visualizer

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewTranscript(t *testing.T) {
	text := "hello world"
	duration := 2 * time.Second

	tx := NewTranscript(text, duration)

	if len(tx.words) != 2 {
		t.Errorf("NewTranscript() words = %d, expected 2", len(tx.words))
	}
	if tx.duration != duration {
		t.Errorf("NewTranscript() duration = %v, expected %v", tx.duration, duration)
	}
}

func TestTranscript_RenderEmpty(t *testing.T) {
	tx := NewTranscript("", time.Second)

	output := tx.RenderSingleLine(80)
	if output != "" {
		t.Errorf("RenderSingleLine() with empty text = %q, expected empty", output)
	}
}

func TestTranscript_RenderFull(t *testing.T) {
	tx := NewTranscript("hello world", time.Second)
	tx.SetElapsed(time.Second)

	output := tx.RenderSingleLine(80)
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("RenderSingleLine() should contain words, got %q", output)
	}
}

func TestTranscript_RenderPartial(t *testing.T) {
	tx := NewTranscript("hello world test", 2*time.Second)
	tx.SetElapsed(time.Second)

	output := tx.RenderSingleLine(80)
	if output == "" {
		t.Error("RenderSingleLine() with partial progress should return non-empty")
	}
}

func TestTranscript_SetDuration(t *testing.T) {
	tx := NewTranscript("hello", time.Second)
	newDuration := 2 * time.Second

	tx.SetDuration(newDuration)
	if tx.duration != newDuration {
		t.Errorf("SetDuration() duration = %v, expected %v", tx.duration, newDuration)
	}
}

func TestRenderSingleLine_Empty(t *testing.T) {
	tx := NewTranscript("", time.Second)
	tx.SetElapsed(time.Second)
	if got := tx.RenderSingleLine(40); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRenderSingleLine_WidthConstraint(t *testing.T) {
	text := strings.Repeat("hello world ", 20)
	tx := NewTranscript(text, 10*time.Second)
	tx.SetElapsed(5 * time.Second)

	for _, w := range []int{20, 30, 40, 60} {
		got := tx.RenderSingleLine(w)
		visible := lipglossWidth(got)
		if visible > w {
			t.Errorf("availableWidth=%d: visible width %d exceeds limit, output=%q", w, visible, got)
		}
	}
}

func TestRenderSingleLine_BrightBeforeDim(t *testing.T) {
	// With 50% progress, revealed words should appear before unrevealed in output.
	tx := NewTranscript("one two three four five six", 10*time.Second)
	tx.SetElapsed(5 * time.Second) // ~50% → ~3 words revealed

	got := tx.RenderSingleLine(60)
	// "one two three" (bright) should appear left of "four five six" (dim)
	idxBright := strings.Index(got, "one")
	idxDim := strings.Index(got, "four")
	if idxBright < 0 {
		t.Fatalf("expected 'one' in output, got %q", got)
	}
	if idxDim < 0 {
		t.Fatalf("expected 'four' in output, got %q", got)
	}
	if idxBright > idxDim {
		t.Errorf("bright word 'one' appears after dim word 'four' in %q", got)
	}
}

func TestRenderSingleLine_MiddleTruncation(t *testing.T) {
	// Long text should show first words + " ... " + last words, not scroll.
	words := make([]string, 50)
	for i := range words {
		words[i] = fmt.Sprintf("w%02d", i)
	}
	text := strings.Join(words, " ")
	tx := NewTranscript(text, 10*time.Second)
	tx.SetElapsed(10 * time.Second) // fully revealed — end state

	got := tx.RenderSingleLine(40)
	visible := stripANSI(got)

	if !strings.Contains(visible, "w00") {
		t.Errorf("expected first word w00 visible (beginning of text), got %q", visible)
	}
	if !strings.Contains(visible, "w49") {
		t.Errorf("expected last word w49 visible (end of text), got %q", visible)
	}
	if !strings.Contains(visible, "...") {
		t.Errorf("expected '...' for middle truncation, got %q", visible)
	}
}

func TestRenderSingleLine_ShowsBothEnds(t *testing.T) {
	// At end of playback, middle truncation shows the beginning and end of the text.
	words := make([]string, 30)
	for i := range words {
		words[i] = fmt.Sprintf("w%02d", i)
	}
	text := strings.Join(words, " ")
	tx := NewTranscript(text, 10*time.Second)
	tx.SetElapsed(10 * time.Second) // fully revealed — end state

	availableWidth := 40
	got := tx.RenderSingleLine(availableWidth)
	visible := stripANSI(got)

	if lipglossWidth(got) > availableWidth {
		t.Errorf("visible width %d exceeds %d: %q", lipglossWidth(got), availableWidth, got)
	}
	if !strings.Contains(visible, "w00") {
		t.Errorf("expected first word w00 visible, got %q", visible)
	}
	if !strings.Contains(visible, "w29") {
		t.Errorf("expected last word w29 visible, got %q", visible)
	}
}

// lipglossWidth returns the visible rune count of s after stripping ANSI escapes.
func lipglossWidth(s string) int {
	return len([]rune(stripANSI(s)))
}

func TestRenderSingleLine_MiddleEllipsis(t *testing.T) {
	// At end of playback, "..." appears in the middle (not at an edge).
	words := make([]string, 50)
	for i := range words {
		words[i] = fmt.Sprintf("w%02d", i)
	}
	tx := NewTranscript(strings.Join(words, " "), 10*time.Second)
	tx.SetElapsed(10 * time.Second) // fully revealed — end state

	got := tx.RenderSingleLine(40)
	visible := stripANSI(got)

	if !strings.Contains(visible, "...") {
		t.Errorf("expected '...' for overflow, got: %q", visible)
	}
	// First and last words visible — "..." is in the middle, not at an edge
	if !strings.Contains(visible, "w00") {
		t.Errorf("expected first word visible, got: %q", visible)
	}
	if !strings.Contains(visible, "w49") {
		t.Errorf("expected last word visible, got: %q", visible)
	}
}

func TestRenderSingleLine_ScrollingDuringAnimation(t *testing.T) {
	// While still animating (rc < n), use scrolling window — NOT middle truncation.
	// Early words should scroll off; the first word is NOT visible when scrolled.
	words := make([]string, 50)
	for i := range words {
		words[i] = fmt.Sprintf("w%02d", i)
	}
	tx := NewTranscript(strings.Join(words, " "), 10*time.Second)
	tx.SetElapsed(7 * time.Second) // 70% revealed, still animating

	got := tx.RenderSingleLine(40)
	visible := stripANSI(got)

	// Width constraint.
	if lipglossWidth(got) > 40 {
		t.Errorf("visible width %d exceeds 40: %q", lipglossWidth(got), got)
	}
	// First word has scrolled off — should NOT be visible.
	if strings.Contains(visible, "w00") {
		t.Errorf("expected w00 to have scrolled off during animation, got: %q", visible)
	}
	// Last word is far ahead — should NOT be visible.
	if strings.Contains(visible, "w49") {
		t.Errorf("expected w49 not visible during scrolling, got: %q", visible)
	}
	// Word near the reveal point should be visible.
	if !strings.Contains(visible, "w35") {
		t.Errorf("expected anchor-area word w35 visible, got: %q", visible)
	}
}

func TestRenderSingleLine_NoEllipsisWhenFits(t *testing.T) {
	tx := NewTranscript("one two three", time.Second)
	tx.SetElapsed(500 * time.Millisecond) // ~50%

	got := tx.RenderSingleLine(60)
	visible := stripANSI(got)
	if strings.Contains(visible, "...") {
		t.Errorf("expected no '...' when text fits, got: %q", visible)
	}
}

// stripANSI removes ANSI escape sequences for test assertions.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestEstimateDurationFromText(t *testing.T) {
	duration := EstimateDurationFromText("hello world")
	if duration <= 0 {
		t.Error("EstimateDurationFromText() should return positive duration")
	}

	emptyDuration := EstimateDurationFromText("")
	if emptyDuration != 0 {
		t.Errorf("EstimateDurationFromText(\"\") = %v, expected 0", emptyDuration)
	}
}
