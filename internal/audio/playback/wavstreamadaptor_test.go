//go:build !headless

package playback

import (
	"bytes"
	"io"
	"testing"

	"github.com/gopxl/beep/v2"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestWavStreamerAdapter_Stream(t *testing.T) {
	wav := testhelpers.MakeValidWAV(100)
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}
	defer adapter.Close()

	samples := make([][2]float64, 10)
	n, ok := adapter.Stream(samples)
	if n == 0 {
		t.Error("Expected Stream to return non-zero samples")
	}
	if !ok {
		t.Error("Expected Stream to return ok=true")
	}
}

func TestWavStreamerAdapter_Err(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}
	defer adapter.Close()

	if err := adapter.Err(); err != nil {
		t.Errorf("Expected no error initially, got: %v", err)
	}
}

func TestWavStreamerAdapter_Len(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}
	defer adapter.Close()

	if adapter.Len() != -1 {
		t.Errorf("Expected Len() to return -1, got: %d", adapter.Len())
	}
}

func TestWavStreamerAdapter_Position(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}
	defer adapter.Close()

	if adapter.Position() != -1 {
		t.Errorf("Expected Position() to return -1, got: %d", adapter.Position())
	}
}

func TestWavStreamerAdapter_Seek(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}
	defer adapter.Close()

	err = adapter.Seek(0)
	if err == nil {
		t.Error("Expected Seek to return an error")
	}
}

func TestWavStreamerAdapter_Close(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	reader := bytes.NewReader(wav)
	decoder, _, err := stream.DecodeStreaming(reader)
	if err != nil {
		t.Fatalf("Failed to decode WAV: %v", err)
	}

	adapter := &wavStreamerAdapter{
		decoder: decoder,
		rc:      io.NopCloser(reader),
	}

	err = adapter.Close()
	if err != nil {
		t.Errorf("Expected Close to succeed, got: %v", err)
	}

	err = adapter.Close()
	if err != nil {
		t.Errorf("Expected Close to be idempotent, got: %v", err)
	}
}

func TestWavStreamerAdapter_ImplementsStreamSeekCloser(t *testing.T) {
	var _ beep.StreamSeekCloser = &wavStreamerAdapter{}
}
