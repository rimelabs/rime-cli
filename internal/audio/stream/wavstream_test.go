package stream

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestDecodeStreaming_ValidWAV(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wavData)

	decoder, format, err := DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	if decoder == nil {
		t.Fatal("DecodeStreaming returned nil decoder")
	}
	if format.SampleRate == 0 {
		t.Error("format.SampleRate should be non-zero")
	}
	if format.NumChannels == 0 {
		t.Error("format.NumChannels should be non-zero")
	}
	if format.Precision == 0 {
		t.Error("format.Precision should be non-zero")
	}
}

func TestDecodeStreaming_InvalidRIFF(t *testing.T) {
	invalidData := []byte("NOTA RIFF file")
	reader := bytes.NewReader(invalidData)

	_, _, err := DecodeStreaming(reader)
	if err == nil {
		t.Error("DecodeStreaming with invalid RIFF should return error")
	}
}

func TestDecodeStreaming_InvalidWAVE(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(4))
	buf.WriteString("NOTA")

	reader := bytes.NewReader(buf.Bytes())
	_, _, err := DecodeStreaming(reader)
	if err == nil {
		t.Error("DecodeStreaming with invalid WAVE should return error")
	}
}

func TestStreamingDecoder_Stream(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wavData)

	decoder, _, err := DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	samples := make([][2]float64, 10)
	n, ok := decoder.Stream(samples)

	if n == 0 {
		t.Error("Stream() should return samples")
	}
	if !ok {
		t.Error("Stream() should return ok=true for first call")
	}
}

func TestStreamingDecoder_StreamEmpty(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(0)
	reader := bytes.NewReader(wavData)

	decoder, _, err := DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	samples := make([][2]float64, 10)
	n, ok := decoder.Stream(samples)

	if n != 0 {
		t.Errorf("Stream() with empty WAV returned %d samples, expected 0", n)
	}
	if ok {
		t.Error("Stream() with empty WAV should return ok=false")
	}
}

func TestStreamingDecoder_Err(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wavData)

	decoder, _, err := DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	if decoder.Err() != nil {
		t.Error("Err() should return nil initially")
	}
}

func TestReadWavHeader_Valid(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wavData)

	header, err := readWavHeader(reader)
	if err != nil {
		t.Fatalf("readWavHeader failed: %v", err)
	}

	if header.SampleRate != 24000 {
		t.Errorf("SampleRate = %d, expected 24000", header.SampleRate)
	}
	if header.NumChannels != 1 {
		t.Errorf("NumChannels = %d, expected 1", header.NumChannels)
	}
	if header.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, expected 16", header.BitsPerSample)
	}
}

func TestReadWavHeader_InvalidRIFF(t *testing.T) {
	invalidData := []byte("NOTA")
	reader := bytes.NewReader(invalidData)

	_, err := readWavHeader(reader)
	if err == nil {
		t.Error("readWavHeader with invalid RIFF should return error")
	}
}

func TestReadWavHeader_InvalidWAVE(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(4))
	buf.WriteString("NOTA")

	reader := bytes.NewReader(buf.Bytes())
	_, err := readWavHeader(reader)
	if err == nil {
		t.Error("readWavHeader with invalid WAVE should return error")
	}
}

func TestReadWavHeader_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(20))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint32(24000))
	binary.Write(&buf, binary.LittleEndian, uint32(48000))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	reader := bytes.NewReader(buf.Bytes())
	_, err := readWavHeader(reader)
	if err == nil {
		t.Error("readWavHeader with unsupported format should return error")
	}
}

func TestStreamingDecoder_Mono(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wavData)

	decoder, format, err := DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	if format.NumChannels != 1 {
		t.Skip("Test requires mono WAV")
	}

	samples := make([][2]float64, 10)
	n, _ := decoder.Stream(samples)

	if n > 0 {
		for i := 0; i < n; i++ {
			if samples[i][0] != samples[i][1] {
				t.Error("Mono audio should duplicate channel to both L and R")
			}
		}
	}
}

func TestStreamingDecoder_ShortRead(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(10)
	reader := io.LimitReader(bytes.NewReader(wavData), int64(len(wavData)/2))

	_, _, err := DecodeStreaming(reader)
	if err == nil {
		t.Error("DecodeStreaming with truncated WAV should return error")
	}
}
