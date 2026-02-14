package formatters

import "testing"

func TestFormatBytes_Bytes(t *testing.T) {
	result := FormatBytes(512)
	expected := "512B"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatBytes_Kilobytes(t *testing.T) {
	result := FormatBytes(1536)
	expected := "1.5KB"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatBytes_Megabytes(t *testing.T) {
	result := FormatBytes(1572864)
	expected := "1.5MB"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatBytes_Zero(t *testing.T) {
	result := FormatBytes(0)
	expected := "0B"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
