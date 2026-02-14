package decode

import (
	"bytes"
	"testing"
)

func TestDecodeAudio_UnsupportedFormat(t *testing.T) {
	data := []byte("test data")
	reader := bytes.NewReader(data)

	_, _, err := DecodeAudio(reader, "audio/unknown")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}
