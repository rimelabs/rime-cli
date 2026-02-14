package visualizer

import (
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

	output := tx.Render()
	if output != "" {
		t.Errorf("Render() with empty text = %q, expected empty", output)
	}
}

func TestTranscript_RenderFull(t *testing.T) {
	tx := NewTranscript("hello world", time.Second)
	tx.SetElapsed(time.Second)

	output := tx.Render()
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("Render() should contain words, got %q", output)
	}
}

func TestTranscript_RenderPartial(t *testing.T) {
	tx := NewTranscript("hello world test", 2*time.Second)
	tx.SetElapsed(time.Second)

	output := tx.Render()
	if output == "" {
		t.Error("Render() with partial progress should return non-empty")
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
