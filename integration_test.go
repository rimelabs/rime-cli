package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/audio/analyze"
	"github.com/rimelabs/rime-cli/internal/audio/metadata"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
	"github.com/rimelabs/rime-cli/internal/config"
)

func TestTTS_Pipeline(t *testing.T) {
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

	resolved, err := config.ResolveConfig("default", "")
	if err != nil {
		t.Fatalf("Failed to resolve config: %v", err)
	}

	client := api.NewClient(api.ClientOptions{
		APIKey:           resolved.APIKey,
		APIURL:           resolved.APIURL,
		AuthHeaderPrefix: resolved.AuthHeaderPrefix,
		Version:          "test-version",
	})
	opts := &api.TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}

	result, err := client.TTSStream("hello world", opts)
	if err != nil {
		t.Fatalf("TTSStream failed: %v", err)
	}
	defer result.Body.Close()

	if result.TTFB <= 0 {
		t.Error("Expected non-zero TTFB")
	}

	var audioBuf bytes.Buffer
	tee := io.TeeReader(result.Body, &audioBuf)

	decoder, format, err := stream.DecodeStreaming(tee)
	if err != nil {
		t.Fatalf("DecodeStreaming failed: %v", err)
	}

	if format.SampleRate == 0 {
		t.Error("Expected non-zero sample rate")
	}
	if format.NumChannels == 0 {
		t.Error("Expected non-zero channel count")
	}

	analyzer := analyze.NewAmplitudeAnalyzer(decoder)

	sampleBuf := make([][2]float64, 512)
	for {
		n, ok := analyzer.Stream(sampleBuf)
		if !ok {
			break
		}
		if n == 0 {
			break
		}
	}

	if analyzer.Err() != nil {
		t.Fatalf("Analyzer error: %v", analyzer.Err())
	}

	audioData := metadata.FixWavHeader(audioBuf.Bytes())
	if len(audioData) == 0 {
		t.Error("Expected non-zero audio data")
	}

	amps, err := analyze.AnalyzeAmplitudes(audioData, 20)
	if err != nil {
		t.Fatalf("AnalyzeAmplitudes failed: %v", err)
	}

	if len(amps) == 0 {
		t.Error("Expected non-zero amplitude array")
	}

	scaled := analyze.ScaleAmplitudes(amps, 5.0, 0.2)
	if len(scaled) == 0 {
		t.Error("Expected non-zero scaled amplitudes")
	}
}
