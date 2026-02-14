package playback

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestRunNonInteractivePlay_ValidWAV(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)
	tmpDir := t.TempDir()
	wavFile := filepath.Join(tmpDir, "test.wav")

	if err := os.WriteFile(wavFile, wavData, 0644); err != nil {
		t.Fatalf("Failed to write test WAV file: %v", err)
	}

	err := RunNonInteractivePlay(wavFile)
	if err != nil {
		t.Fatalf("RunNonInteractivePlay failed: %v", err)
	}
}

func TestRunNonInteractivePlay_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.wav")

	err := RunNonInteractivePlay(nonExistentFile)
	if err == nil {
		t.Error("RunNonInteractivePlay should return error for non-existent file")
	}
}

func TestRunNonInteractivePlay_InvalidWAV(t *testing.T) {
	tmpDir := t.TempDir()
	invalidWavFile := filepath.Join(tmpDir, "invalid.wav")

	invalidData := []byte("not a valid WAV file")
	if err := os.WriteFile(invalidWavFile, invalidData, 0644); err != nil {
		t.Fatalf("Failed to write invalid WAV file: %v", err)
	}

	err := RunNonInteractivePlay(invalidWavFile)
	if err == nil {
		t.Error("RunNonInteractivePlay should return error for invalid WAV")
	}
}
