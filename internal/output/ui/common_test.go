package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rimelabs/rime-cli/internal/output/visualizer"
)

func textLineFromPanel(lines []string) string {
	for _, line := range lines {
		if strings.Contains(line, "text:") {
			return line
		}
	}
	return ""
}

func TestRenderRightPanel_TextLineDoesNotOverflow(t *testing.T) {
	// A transcript whose rendered output is wider than the panel.
	longText := strings.Repeat("hello world ", 20) // ~240 chars
	tx := visualizer.NewTranscript(longText, 10*time.Second)
	tx.SetElapsed(10 * time.Second) // fully revealed

	width := 40
	lines := RenderRightPanel("speaker: foo  model: bar  lang: eng", width, tx, "0:01", "", nil)

	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > width {
			t.Errorf("line visible width %d exceeds panel width %d: %q", w, width, line)
		}
	}
}

func TestRenderRightPanel_LongTextMiddleTruncated(t *testing.T) {
	// Long text should show first and last words with "..." between them.
	words := make([]string, 40)
	for i := range words {
		words[i] = fmt.Sprintf("w%02d", i)
	}
	tx := visualizer.NewTranscript(strings.Join(words, " "), 10*time.Second)
	tx.SetElapsed(10 * time.Second) // fully revealed — end state

	width := 40
	lines := RenderRightPanel("speaker: foo  model: bar  lang: eng", width, tx, "0:01", "", nil)

	textLine := textLineFromPanel(lines)
	if textLine == "" {
		t.Fatal("expected to find a text: line in the panel output")
	}
	// First word should be visible (beginning of text)
	if !strings.Contains(textLine, "w00") {
		t.Errorf("expected first word w00 visible, got: %q", textLine)
	}
	// Last word should be visible (end of text)
	if !strings.Contains(textLine, "w39") {
		t.Errorf("expected last word w39 visible, got: %q", textLine)
	}
	// visible width must not exceed panel width
	if w := lipgloss.Width(textLine); w > width {
		t.Errorf("text line width %d exceeds panel width %d: %q", w, width, textLine)
	}
}

func TestRenderRightPanel_ShortTextNotTruncated(t *testing.T) {
	shortText := "hi"
	tx := visualizer.NewTranscript(shortText, time.Second)
	tx.SetElapsed(time.Second)

	width := 60
	lines := RenderRightPanel("speaker: foo  model: bar  lang: eng", width, tx, "0:01", "", nil)

	textLine := textLineFromPanel(lines)
	if textLine == "" {
		t.Fatal("expected to find a text: line in the panel output")
	}
	if strings.Contains(textLine, "...") {
		t.Errorf("short text should not be truncated, got: %q", textLine)
	}
	if !strings.Contains(textLine, "hi") {
		t.Errorf("expected text line to contain 'hi', got: %q", textLine)
	}
}

func TestRenderRightPanel_TextLineWidth(t *testing.T) {
	// Verify available width formula: width - lipgloss.Width(label) - 2
	labelWidth := lipgloss.Width(DimStyle.Render("text: "))
	if labelWidth != 6 {
		t.Errorf("label 'text: ' visible width = %d, expected 6", labelWidth)
	}

	longText := strings.Repeat("abcde ", 30)
	tx := visualizer.NewTranscript(longText, time.Second)
	tx.SetElapsed(time.Second)

	for _, width := range []int{30, 40, 50, 60} {
		lines := RenderRightPanel("", width, tx, "0:01", "", nil)
		for _, line := range lines {
			w := lipgloss.Width(line)
			if w > width {
				t.Errorf("width=%d: line visible width %d exceeds panel width: %q", width, w, line)
			}
		}
	}
}
