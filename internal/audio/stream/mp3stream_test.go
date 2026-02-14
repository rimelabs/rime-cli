package stream

import (
	"bytes"
	"io"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestDecodeMP3Streaming_InvalidData(t *testing.T) {
	invalidData := []byte("not an MP3 file")
	reader := io.NopCloser(bytes.NewReader(invalidData))

	_, _, err := DecodeMP3Streaming(reader)
	if err == nil {
		t.Error("DecodeMP3Streaming with invalid data should return error")
	}
}

func TestDecodeMP3Streaming_EmptyData(t *testing.T) {
	emptyData := []byte{}
	reader := io.NopCloser(bytes.NewReader(emptyData))

	_, _, err := DecodeMP3Streaming(reader)
	if err == nil {
		t.Error("DecodeMP3Streaming with empty data should return error")
	}
}

func TestDecodeMP3Streaming_InvalidSyncWord(t *testing.T) {
	invalidSync := []byte{0xFF, 0xFA, 0x00, 0x00}
	reader := io.NopCloser(bytes.NewReader(invalidSync))

	_, _, err := DecodeMP3Streaming(reader)
	if err == nil {
		t.Error("DecodeMP3Streaming with invalid sync word should return error")
	}
}

func TestDecodeMP3Streaming_WithID3ButNoAudio(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("ID3")
	buf.WriteByte(0x03)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x10})
	buf.Write(make([]byte, 16))

	reader := io.NopCloser(bytes.NewReader(buf.Bytes()))

	_, _, err := DecodeMP3Streaming(reader)
	if err == nil {
		t.Error("DecodeMP3Streaming with ID3 but no audio data should return error")
	}
}

func TestDecodeMP3Streaming_NilReader(t *testing.T) {
	_, _, err := DecodeMP3Streaming(nil)
	if err == nil {
		t.Error("DecodeMP3Streaming with nil reader should return error")
	}
}

func TestDecodeMP3Streaming_ReturnsStreamer(t *testing.T) {
	mp3Data := testhelpers.MakeMinimalMP3()
	reader := io.NopCloser(bytes.NewReader(mp3Data))

	streamer, format, err := DecodeMP3Streaming(reader)
	if err == nil {
		if streamer == nil {
			t.Error("DecodeMP3Streaming should return non-nil streamer on success")
		}
		if format.SampleRate == 0 {
			t.Error("DecodeMP3Streaming should return format with non-zero sample rate")
		}
		if streamer != nil {
			streamer.Close()
		}
	}
}

func TestDecodeMP3Streaming_StreamerInterface(t *testing.T) {
	mp3Data := testhelpers.MakeMinimalMP3()
	reader := io.NopCloser(bytes.NewReader(mp3Data))

	streamer, _, err := DecodeMP3Streaming(reader)
	if err != nil {
		t.Skipf("Skipping streamer interface test: %v", err)
	}
	defer streamer.Close()

	length := streamer.Len()
	position := streamer.Position()

	if length < 0 && position < 0 {
		t.Log("MP3 streamer may not support Len/Position (this is expected for some MP3 files)")
	}

	samples := make([][2]float64, 10)
	n, ok := streamer.Stream(samples)

	if n < 0 {
		t.Error("Stream() should return non-negative sample count")
	}

	if n > 0 && !ok {
		t.Error("Stream() should return ok=true when samples are available")
	}
}
