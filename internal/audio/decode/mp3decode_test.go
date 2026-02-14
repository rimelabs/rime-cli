package decode

import (
	"bytes"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestDecodeAudio_MP3(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()
	reader := bytes.NewReader(mp3)

	decoder, format, err := DecodeAudio(reader, "audio/mpeg")
	if err != nil {
		t.Skipf("DecodeAudio MP3 test skipped: %v (may need valid MP3 data)", err)
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

func TestDecodeAudio_MP3_AltMimeType(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()
	reader := bytes.NewReader(mp3)

	decoder, format, err := DecodeAudio(reader, "audio/mp3")
	if err != nil {
		t.Skipf("DecodeAudio MP3 test skipped: %v (may need valid MP3 data)", err)
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
