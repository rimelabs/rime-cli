package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/tts"
)

func TestTTS_HappyPath(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	if err := config.SaveAPIKey("test-key"); err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	Version = "test-version"
	client := api.NewClient("test-key", Version)
	opts := &api.TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}

	result, err := client.TTSStream("hello", opts)
	if err != nil {
		t.Fatalf("TTSStream failed: %v", err)
	}
	defer result.Body.Close()

	var audioBuf bytes.Buffer
	_, err = io.Copy(&audioBuf, result.Body)
	if err != nil {
		t.Fatalf("Failed to read audio: %v", err)
	}

	if audioBuf.Len() == 0 {
		t.Error("Expected non-zero audio data")
	}
}

func TestTTS_SaveToFile(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	if err := config.SaveAPIKey("test-key"); err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "out.wav")
	Version = "test-version"

	runOpts := tts.RunOptions{
		Text: "hello",
		TTSOptions: &api.TTSOptions{
			Speaker: "astra",
			ModelID: "arcana",
		},
		Output:  outputFile,
		Play:    false,
		Quiet:   true,
		JSON:    false,
		Version: Version,
	}
	err := tts.RunNonInteractive(runOpts)
	if err != nil {
		t.Fatalf("runNonInteractiveTTS failed: %v", err)
	}

	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty")
	}

	if !bytes.HasPrefix(data, []byte("RIFF")) {
		t.Error("Output file is not a valid WAV file")
	}
}

func TestTTS_StdoutOutput(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	if err := config.SaveAPIKey("test-key"); err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	Version = "test-version"
	client := api.NewClient("test-key", Version)
	opts := &api.TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}

	audioData, err := client.TTS("hello", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	if len(audioData) == 0 {
		t.Error("Expected non-zero audio data")
	}

	if !bytes.HasPrefix(audioData, []byte("RIFF")) {
		t.Error("Audio data is not a valid WAV file")
	}
}

func TestTTS_MissingAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_CLI_API_KEY")

	_, err := config.LoadAPIKey()
	if err == nil {
		t.Error("Expected error when API key is missing")
	}

	if !strings.Contains(err.Error(), "rime login") {
		t.Errorf("Error should mention 'rime login', got: %v", err)
	}
}

func TestTTS_InvalidArguments(t *testing.T) {
	cmd := NewTTSCmd()

	err := cmd.ValidateArgs([]string{})
	if err == nil {
		t.Error("Expected error for zero arguments")
	}

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	if err := config.SaveAPIKey("test-key"); err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	Version = "test-version"
	err = cmd.RunE(cmd, []string{"hello"})
	if err == nil {
		t.Error("Expected error for missing required flags")
	}
	if err != nil && !strings.Contains(err.Error(), "--speaker") && !strings.Contains(err.Error(), "--model-id") {
		t.Errorf("Error should mention required flags, got: %v", err)
	}
}
