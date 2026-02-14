package formatters

import (
	"testing"
	"time"
)

func TestFormatDuration_Seconds(t *testing.T) {
	result := FormatDuration(30 * time.Second)
	expected := "0:30"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	result := FormatDuration(125 * time.Second)
	expected := "2:05"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	result := FormatDuration(1 * time.Hour)
	expected := "60:00"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	result := FormatDuration(0)
	expected := "0:00"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
