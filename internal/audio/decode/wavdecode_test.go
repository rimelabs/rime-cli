package decode

import (
	"bytes"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestDecodeAudio_WAV(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)

	decoder, format, err := DecodeAudio(reader, "audio/wav")
	if err != nil {
		t.Fatalf("DecodeAudio failed: %v", err)
	}

	if format.SampleRate == 0 {
		t.Error("Expected non-zero sample rate")
	}

	if decoder == nil {
		t.Error("Expected non-nil decoder")
	}

	if closer, ok := decoder.(interface{ Close() error }); ok {
		closer.Close()
	}
}
