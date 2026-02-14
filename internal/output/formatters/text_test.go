package formatters

import "testing"

func TestTruncateText_NoTruncation(t *testing.T) {
	result := TruncateText("hello", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncateText_WithEllipsis(t *testing.T) {
	result := TruncateText("hello world", 8)
	expected := "hello..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTruncateText_MaxLen3(t *testing.T) {
	result := TruncateText("hello", 3)
	expected := "hel"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTruncateText_EmptyString(t *testing.T) {
	result := TruncateText("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
