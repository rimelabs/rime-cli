package tts

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
	"github.com/rimelabs/rime-cli/internal/config"
)

func TestRunNonInteractive_SaveToFile(t *testing.T) {
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

	outputFile := filepath.Join(tmpDir, "output.wav")
	opts := &api.TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}

	runOpts := RunOptions{
		Text:       "hello",
		TTSOptions: opts,
		Output:     outputFile,
		Play:       false,
		Quiet:      true,
		JSON:       false,
		Version:    "test-version",
	}
	err := RunNonInteractive(runOpts)
	if err != nil {
		t.Fatalf("RunNonInteractive failed: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file should be created")
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("RIFF")) {
		t.Error("Output file should be a valid WAV file")
	}
}

func TestRunNonInteractive_MissingAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	opts := &api.TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}

	runOpts := RunOptions{
		Text:       "hello",
		TTSOptions: opts,
		Output:     "",
		Play:       false,
		Quiet:      true,
		JSON:       false,
		Version:    "test-version",
	}
	err := RunNonInteractive(runOpts)
	if err == nil {
		t.Error("RunNonInteractive should return error when API key is missing")
	}
}
