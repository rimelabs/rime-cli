package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTTS_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("Expected User-Agent header")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-audio-data"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	audio, err := client.TTS("hello world", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	if string(audio) != "fake-audio-data" {
		t.Errorf("Expected fake-audio-data, got %s", string(audio))
	}
}

func TestTTS_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid api key"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("bad-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTS("hello", opts)
	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}

func TestGetAPIURL_EnvOverride(t *testing.T) {
	originalURL := os.Getenv("RIME_API_URL")
	defer os.Setenv("RIME_API_URL", originalURL)

	os.Unsetenv("RIME_API_URL")
	if url := GetAPIURL(); url != defaultAPIBaseURL {
		t.Errorf("Expected %s, got %s", defaultAPIBaseURL, url)
	}

	os.Setenv("RIME_API_URL", "https://custom.api.url/custom/path")
	expectedURL := "https://custom.api.url/custom/path"
	if url := GetAPIURL(); url != expectedURL {
		t.Errorf("Expected %s, got %s", expectedURL, url)
	}
}

func TestTTS_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "celeste",
		ModelID: "arcana",
		Lang:    "eng",
	}
	_, err := client.TTS("test", opts)
	if err != nil {
		t.Fatalf("TTS with options failed: %v", err)
	}
}

func TestTTS_NetworkError(t *testing.T) {
	os.Setenv("RIME_API_URL", "http://localhost:99999")
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	client.client.Timeout = 100 * time.Millisecond
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTS("hello", opts)
	if err == nil {
		t.Error("Expected error for network failure")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Error should mention request failure, got: %v", err)
	}
}

func TestTTS_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("truncated"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	audio, err := client.TTS("hello", opts)
	if err != nil {
		t.Logf("Got expected error for malformed response: %v", err)
		return
	}
	if len(audio) != 9 {
		t.Errorf("Expected truncated data, got %d bytes", len(audio))
	}
}

func TestTTSStream_Success(t *testing.T) {
	wavData := []byte("RIFF")
	wavData = append(wavData, make([]byte, 40)...)
	wavData = append(wavData, []byte("WAVEfmt ")...)
	wavData = append(wavData, make([]byte, 20)...)
	wavData = append(wavData, []byte("data")...)
	wavData = append(wavData, make([]byte, 100)...)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	result, err := client.TTSStream("hello", opts)
	if err != nil {
		t.Fatalf("TTSStream failed: %v", err)
	}
	defer result.Body.Close()

	if result.TTFB <= 0 {
		t.Error("Expected non-zero TTFB")
	}

	data, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-zero audio data")
	}
}

func TestTTSStream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid api key"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("bad-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTSStream("hello", opts)
	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("Error should mention authentication, got: %v", err)
	}
}

func TestGetDashboardURL_EnvOverride(t *testing.T) {
	originalURL := os.Getenv("RIME_DASHBOARD_URL")
	defer os.Setenv("RIME_DASHBOARD_URL", originalURL)

	os.Unsetenv("RIME_DASHBOARD_URL")
	if url := GetDashboardURL(); url != defaultDashboardURL {
		t.Errorf("Expected %s, got %s", defaultDashboardURL, url)
	}

	os.Setenv("RIME_DASHBOARD_URL", "https://custom.dashboard.url")
	if url := GetDashboardURL(); url != "https://custom.dashboard.url" {
		t.Errorf("Expected https://custom.dashboard.url, got %s", url)
	}
}

func TestIsValidLang(t *testing.T) {
	tests := []struct {
		lang    string
		modelID string
		want    bool
	}{
		{"eng", ModelIDArcana, true},
		{"en", ModelIDArcana, true},
		{"ara", ModelIDArcana, true},
		{"ar", ModelIDArcana, true},
		{"jpn", ModelIDArcana, true},
		{"ja", ModelIDArcana, true},
		{"heb", ModelIDArcana, true},
		{"tam", ModelIDArcana, true},
		{"xyz", ModelIDArcana, false},
		{"", ModelIDArcana, false},

		{"eng", ModelIDArcanaV2, true},
		{"en", ModelIDArcanaV2, true},
		{"spa", ModelIDArcanaV2, true},
		{"es", ModelIDArcanaV2, true},
		{"ger", ModelIDArcanaV2, true},
		{"de", ModelIDArcanaV2, true},
		{"fra", ModelIDArcanaV2, true},
		{"fr", ModelIDArcanaV2, true},
		{"hin", ModelIDArcanaV2, true},
		{"hi", ModelIDArcanaV2, true},
		{"ara", ModelIDArcanaV2, false},
		{"jpn", ModelIDArcanaV2, false},
		{"ja", ModelIDArcanaV2, false},
		{"heb", ModelIDArcanaV2, false},
		{"tam", ModelIDArcanaV2, false},
		{"xyz", ModelIDArcanaV2, false},
		{"", ModelIDArcanaV2, false},

		{"eng", ModelIDMistV2, true},
		{"en", ModelIDMistV2, true},
		{"fra", ModelIDMistV2, true},
		{"spa", ModelIDMistV2, true},
		{"de", ModelIDMistV2, true},
		{"ara", ModelIDMistV2, false},
		{"jpn", ModelIDMistV2, false},
		{"ja", ModelIDMistV2, false},
		{"heb", ModelIDMistV2, false},
		{"por", ModelIDMistV2, false},

		{"eng", ModelIDMist, true},
		{"ara", ModelIDMist, false},
		{"jpn", ModelIDMist, false},
	}
	for _, tt := range tests {
		got := IsValidLang(tt.lang, tt.modelID)
		if got != tt.want {
			t.Errorf("IsValidLang(%q, %q) = %v, want %v", tt.lang, tt.modelID, got, tt.want)
		}
	}
}

func TestValidLangsForModel(t *testing.T) {
	arcanaLangs := ValidLangsForModel(ModelIDArcana)
	if len(arcanaLangs) != 22 {
		t.Errorf("expected 22 arcana lang codes (11 iso3 + 11 iso1), got %d: %v", len(arcanaLangs), arcanaLangs)
	}

	arcanaV2Langs := ValidLangsForModel(ModelIDArcanaV2)
	if len(arcanaV2Langs) != 10 {
		t.Errorf("expected 10 arcanav2 lang codes (5 iso3 + 5 iso1), got %d: %v", len(arcanaV2Langs), arcanaV2Langs)
	}

	mistLangs := ValidLangsForModel(ModelIDMistV2)
	if len(mistLangs) != 8 {
		t.Errorf("expected 8 mist lang codes (4 iso3 + 4 iso1), got %d: %v", len(mistLangs), mistLangs)
	}

	for i := 1; i < len(arcanaLangs); i++ {
		if arcanaLangs[i] < arcanaLangs[i-1] {
			t.Errorf("arcana langs not sorted: %v", arcanaLangs)
			break
		}
	}

	for i := 1; i < len(arcanaV2Langs); i++ {
		if arcanaV2Langs[i] < arcanaV2Langs[i-1] {
			t.Errorf("arcanav2 langs not sorted: %v", arcanaV2Langs)
			break
		}
	}
}

func TestGetAudioFormat_WAV(t *testing.T) {
	format := GetAudioFormat("arcana")
	if format != "audio/wav" {
		t.Errorf("Expected audio/wav, got %s", format)
	}

	format = GetAudioFormat("arcanav2")
	if format != "audio/wav" {
		t.Errorf("Expected audio/wav for arcanav2, got %s", format)
	}

	format = GetAudioFormat("")
	if format != "audio/wav" {
		t.Errorf("Expected audio/wav for empty model, got %s", format)
	}
}

func TestGetAudioFormat_MP3(t *testing.T) {
	format := GetAudioFormat("mistv2")
	if format != "audio/mp3" {
		t.Errorf("Expected audio/mp3, got %s", format)
	}
}

func TestTTS_MP3Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "audio/mp3" {
			t.Errorf("Expected Accept: audio/mp3, got %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-mp3-data"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "mistv2",
	}
	audio, err := client.TTS("hello world", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	if string(audio) != "fake-mp3-data" {
		t.Errorf("Expected fake-mp3-data, got %s", string(audio))
	}
}

func TestTTSStream_MP3Request(t *testing.T) {
	mp3Data := []byte("ID3")
	mp3Data = append(mp3Data, make([]byte, 7)...)
	mp3Data = append(mp3Data, []byte{0xFF, 0xFB, 0x90, 0x00}...)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "audio/mp3" {
			t.Errorf("Expected Accept: audio/mp3, got %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(mp3Data)
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "mistv2",
	}
	result, err := client.TTSStream("hello", opts)
	if err != nil {
		t.Fatalf("TTSStream failed: %v", err)
	}
	defer result.Body.Close()

	if result.ContentType != "audio/mpeg" {
		t.Errorf("Expected Content-Type audio/mpeg, got %s", result.ContentType)
	}
}

func TestIsValidModelID(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{ModelIDArcana, true},
		{ModelIDArcanaV2, true},
		{ModelIDMistV2, true},
		{ModelIDMist, true},
		{"invalid", false},
		{"", false},
		{"ARCANA", false},
		{"arcana ", false},
	}
	for _, tt := range tests {
		got := IsValidModelID(tt.modelID)
		if got != tt.want {
			t.Errorf("IsValidModelID(%q) = %v, want %v", tt.modelID, got, tt.want)
		}
	}
}

func TestTTS_ArcanaV2Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "audio/wav" {
			t.Errorf("Expected Accept: audio/wav, got %s", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-wav-data"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient("test-key", "1.0.0")
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcanav2",
		Lang:    "eng",
	}
	audio, err := client.TTS("hello world", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	if string(audio) != "fake-wav-data" {
		t.Errorf("Expected fake-wav-data, got %s", string(audio))
	}
}
