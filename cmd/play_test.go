package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/playback"
	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestPlay_HappyPath(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)
	tmpDir := t.TempDir()
	wavFile := filepath.Join(tmpDir, "test.wav")

	if err := os.WriteFile(wavFile, wavData, 0644); err != nil {
		t.Fatalf("Failed to write test WAV file: %v", err)
	}

	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	err := playback.RunNonInteractivePlay(wavFile)
	if err != nil {
		t.Fatalf("RunNonInteractivePlay failed: %v", err)
	}
}

func TestPlay_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.wav")

	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	cmd := NewPlayCmd()
	err := cmd.RunE(cmd, []string{nonExistentFile})
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("file not found")) {
		t.Errorf("Error should mention 'file not found', got: %v", err)
	}
}

func TestPlay_InvalidWAV(t *testing.T) {
	tmpDir := t.TempDir()
	invalidWavFile := filepath.Join(tmpDir, "invalid.wav")

	invalidData := []byte("not a valid WAV file")
	if err := os.WriteFile(invalidWavFile, invalidData, 0644); err != nil {
		t.Fatalf("Failed to write invalid WAV file: %v", err)
	}

	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	err := playback.RunNonInteractivePlay(invalidWavFile)
	if err == nil {
		t.Error("Expected error for invalid WAV file")
	}
}
