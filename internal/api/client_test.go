package api

import (
	"encoding/json"
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

func TestNewClientWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Api-Key custom-key" {
			t.Errorf("Expected 'Api-Key custom-key', got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		APIKey:           "custom-key",
		APIURL:           server.URL,
		AuthHeaderPrefix: "Api-Key",
		Version:          "1.0.0",
	})

	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTS("test", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}
}

func TestNewClientWithOptions_NoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Expected no Authorization header, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		APIKey:           "",
		APIURL:           server.URL,
		AuthHeaderPrefix: "Bearer",
		Version:          "1.0.0",
	})

	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTS("test", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}
}

func TestNewClientWithOptions_EmptyPrefix(t *testing.T) {
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")
	defer os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			t.Errorf("Expected no Authorization header when prefix is explicitly empty, got %q", authHeader)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	authPrefix := ""
	client := NewClient(ClientOptions{
		APIKey:           "",
		APIURL:           server.URL,
		AuthHeaderPrefix: authPrefix,
		Version:          "1.0.0",
	})

	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: "arcana",
	}
	_, err := client.TTS("test", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
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

	client := NewClient(ClientOptions{
		APIKey:  "bad-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "bad-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
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

func TestIsArcanaModel(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{ModelIDArcana, true},
		{ModelIDArcanaV2, true},
		{ModelIDMist, false},
		{ModelIDMistV2, false},
		{"", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		got := IsArcanaModel(tt.modelID)
		if got != tt.want {
			t.Errorf("IsArcanaModel(%q) = %v, want %v", tt.modelID, got, tt.want)
		}
	}
}

func TestIsMistModel(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{ModelIDMist, true},
		{ModelIDMistV2, true},
		{ModelIDArcana, false},
		{ModelIDArcanaV2, false},
		{"", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		got := IsMistModel(tt.modelID)
		if got != tt.want {
			t.Errorf("IsMistModel(%q) = %v, want %v", tt.modelID, got, tt.want)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestValidateModelParams(t *testing.T) {
	tests := []struct {
		name    string
		opts    *TTSOptions
		wantErr string
	}{
		{"nil opts", nil, ""},

		// Arcana-only params on arcana model — valid
		{"temperature valid arcana", &TTSOptions{ModelID: ModelIDArcana, Temperature: ptr(0.5)}, ""},
		{"top-p valid arcana", &TTSOptions{ModelID: ModelIDArcana, TopP: ptr(0.8)}, ""},
		{"repetition-penalty valid arcana", &TTSOptions{ModelID: ModelIDArcana, RepetitionPenalty: ptr(1.5)}, ""},
		{"max-tokens valid arcana", &TTSOptions{ModelID: ModelIDArcana, MaxTokens: ptr(1200)}, ""},

		// Arcana-only params on mist — errors
		{"temperature on mist", &TTSOptions{ModelID: ModelIDMist, Temperature: ptr(0.5)}, "arcana/arcanav2"},
		{"top-p on mistv2", &TTSOptions{ModelID: ModelIDMistV2, TopP: ptr(0.5)}, "arcana/arcanav2"},
		{"repetition-penalty on mist", &TTSOptions{ModelID: ModelIDMist, RepetitionPenalty: ptr(1.5)}, "arcana/arcanav2"},
		{"max-tokens on mistv2", &TTSOptions{ModelID: ModelIDMistV2, MaxTokens: ptr(1000)}, "arcana/arcanav2"},

		// Out-of-range arcana params
		{"temperature too low", &TTSOptions{ModelID: ModelIDArcana, Temperature: ptr(-0.1)}, "between 0 and 1"},
		{"temperature too high", &TTSOptions{ModelID: ModelIDArcana, Temperature: ptr(1.1)}, "between 0 and 1"},
		{"top-p too high", &TTSOptions{ModelID: ModelIDArcana, TopP: ptr(1.5)}, "between 0 and 1"},
		{"repetition-penalty too low", &TTSOptions{ModelID: ModelIDArcana, RepetitionPenalty: ptr(0.5)}, "between 1 and 2"},
		{"repetition-penalty too high", &TTSOptions{ModelID: ModelIDArcana, RepetitionPenalty: ptr(2.5)}, "between 1 and 2"},
		{"max-tokens too low", &TTSOptions{ModelID: ModelIDArcana, MaxTokens: ptr(100)}, "between 200 and 5000"},
		{"max-tokens too high", &TTSOptions{ModelID: ModelIDArcana, MaxTokens: ptr(6000)}, "between 200 and 5000"},

		// Mist-only params on mist — valid
		{"pause-between-brackets on mist", &TTSOptions{ModelID: ModelIDMist, PauseBetweenBrackets: ptr(true)}, ""},
		{"phonemize-between-brackets on mistv2", &TTSOptions{ModelID: ModelIDMistV2, PhonemizeBetweenBrackets: ptr(true)}, ""},
		{"inline-speed-alpha on mist", &TTSOptions{ModelID: ModelIDMist, InlineSpeedAlpha: ptr("1.0,1.2")}, ""},
		{"no-text-normalization on mist", &TTSOptions{ModelID: ModelIDMist, NoTextNormalization: ptr(true)}, ""},
		{"save-oovs on mist", &TTSOptions{ModelID: ModelIDMist, SaveOovs: ptr(true)}, ""},

		// Mist-only params on arcana — errors
		{"pause-between-brackets on arcana", &TTSOptions{ModelID: ModelIDArcana, PauseBetweenBrackets: ptr(true)}, "mist/mistv2"},
		{"phonemize-between-brackets on arcana", &TTSOptions{ModelID: ModelIDArcana, PhonemizeBetweenBrackets: ptr(true)}, "mist/mistv2"},
		{"inline-speed-alpha on arcana", &TTSOptions{ModelID: ModelIDArcana, InlineSpeedAlpha: ptr("1.0")}, "mist/mistv2"},
		{"no-text-normalization on arcanav2", &TTSOptions{ModelID: ModelIDArcanaV2, NoTextNormalization: ptr(true)}, "mist/mistv2"},
		{"save-oovs on arcana", &TTSOptions{ModelID: ModelIDArcana, SaveOovs: ptr(true)}, "mist/mistv2"},

		// SpeedAlpha
		{"speed-alpha valid", &TTSOptions{ModelID: ModelIDArcana, SpeedAlpha: ptr(1.5)}, ""},
		{"speed-alpha zero", &TTSOptions{ModelID: ModelIDArcana, SpeedAlpha: ptr(0.0)}, "greater than 0"},
		{"speed-alpha negative", &TTSOptions{ModelID: ModelIDMist, SpeedAlpha: ptr(-0.5)}, "greater than 0"},

		// SamplingRate - arcana valid
		{"sampling-rate 24000 arcana", &TTSOptions{ModelID: ModelIDArcana, SamplingRate: ptr(24000)}, ""},
		{"sampling-rate 8000 arcana", &TTSOptions{ModelID: ModelIDArcana, SamplingRate: ptr(8000)}, ""},
		{"sampling-rate 96000 arcana", &TTSOptions{ModelID: ModelIDArcana, SamplingRate: ptr(96000)}, ""},
		// SamplingRate - arcana invalid
		{"sampling-rate 12000 arcana", &TTSOptions{ModelID: ModelIDArcana, SamplingRate: ptr(12000)}, "must be one of"},
		// SamplingRate - mist valid
		{"sampling-rate 16000 mist", &TTSOptions{ModelID: ModelIDMist, SamplingRate: ptr(16000)}, ""},
		{"sampling-rate 4000 mist", &TTSOptions{ModelID: ModelIDMist, SamplingRate: ptr(4000)}, ""},
		{"sampling-rate 44100 mist", &TTSOptions{ModelID: ModelIDMist, SamplingRate: ptr(44100)}, ""},
		// SamplingRate - mist invalid
		{"sampling-rate 3999 mist", &TTSOptions{ModelID: ModelIDMist, SamplingRate: ptr(3999)}, "between 4000 and 44100"},
		{"sampling-rate 44101 mist", &TTSOptions{ModelID: ModelIDMist, SamplingRate: ptr(44101)}, "between 4000 and 44100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelParams(tt.opts)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestTTS_SerializesNewParams(t *testing.T) {
	var captured TTSRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
	opts := &TTSOptions{
		Speaker:           "astra",
		ModelID:           ModelIDArcana,
		Temperature:       ptr(0.7),
		TopP:              ptr(0.9),
		RepetitionPenalty: ptr(1.3),
		MaxTokens:         ptr(800),
		SamplingRate:      ptr(24000),
		SpeedAlpha:        ptr(1.2),
	}
	_, err := client.TTS("hello", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	if captured.Temperature == nil || *captured.Temperature != 0.7 {
		t.Errorf("expected Temperature=0.7, got %v", captured.Temperature)
	}
	if captured.TopP == nil || *captured.TopP != 0.9 {
		t.Errorf("expected TopP=0.9, got %v", captured.TopP)
	}
	if captured.RepetitionPenalty == nil || *captured.RepetitionPenalty != 1.3 {
		t.Errorf("expected RepetitionPenalty=1.3, got %v", captured.RepetitionPenalty)
	}
	if captured.MaxTokens == nil || *captured.MaxTokens != 800 {
		t.Errorf("expected MaxTokens=800, got %v", captured.MaxTokens)
	}
	if captured.SamplingRate == nil || *captured.SamplingRate != 24000 {
		t.Errorf("expected SamplingRate=24000, got %v", captured.SamplingRate)
	}
	if captured.SpeedAlpha == nil || *captured.SpeedAlpha != 1.2 {
		t.Errorf("expected SpeedAlpha=1.2, got %v", captured.SpeedAlpha)
	}
}

func TestTTSStream_SerializesNewParams(t *testing.T) {
	var captured TTSRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mp3data"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
	opts := &TTSOptions{
		Speaker:                  "astra",
		ModelID:                  ModelIDMistV2,
		SamplingRate:             ptr(16000),
		SpeedAlpha:               ptr(0.8),
		PauseBetweenBrackets:     ptr(true),
		PhonemizeBetweenBrackets: ptr(false),
		InlineSpeedAlpha:         ptr("1.0,1.2"),
		NoTextNormalization:      ptr(true),
		SaveOovs:                 ptr(false),
	}
	result, err := client.TTSStream("hello", opts)
	if err != nil {
		t.Fatalf("TTSStream failed: %v", err)
	}
	result.Body.Close()

	if captured.SamplingRate == nil || *captured.SamplingRate != 16000 {
		t.Errorf("expected SamplingRate=16000, got %v", captured.SamplingRate)
	}
	if captured.SpeedAlpha == nil || *captured.SpeedAlpha != 0.8 {
		t.Errorf("expected SpeedAlpha=0.8, got %v", captured.SpeedAlpha)
	}
	if captured.PauseBetweenBrackets == nil || *captured.PauseBetweenBrackets != true {
		t.Errorf("expected PauseBetweenBrackets=true, got %v", captured.PauseBetweenBrackets)
	}
	if captured.PhonemizeBetweenBrackets == nil {
		t.Error("expected PhonemizeBetweenBrackets to be present, but it was omitted")
	} else if *captured.PhonemizeBetweenBrackets != false {
		t.Errorf("expected PhonemizeBetweenBrackets=false, got %v", *captured.PhonemizeBetweenBrackets)
	}
	if captured.InlineSpeedAlpha == nil || *captured.InlineSpeedAlpha != "1.0,1.2" {
		t.Errorf("expected InlineSpeedAlpha=1.0,1.2, got %v", captured.InlineSpeedAlpha)
	}
	if captured.NoTextNormalization == nil || *captured.NoTextNormalization != true {
		t.Errorf("expected NoTextNormalization=true, got %v", captured.NoTextNormalization)
	}
	if captured.SaveOovs == nil {
		t.Error("expected SaveOovs to be present, but it was omitted")
	} else if *captured.SaveOovs != false {
		t.Errorf("expected SaveOovs=false, got %v", *captured.SaveOovs)
	}
}

func TestTTS_UnsetParamsOmitted(t *testing.T) {
	var rawBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		rawBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	os.Setenv("RIME_API_URL", server.URL)
	defer os.Unsetenv("RIME_API_URL")

	client := NewClient(ClientOptions{
		APIKey:  "test-key",
		Version: "1.0.0",
	})
	opts := &TTSOptions{
		Speaker: "astra",
		ModelID: ModelIDArcana,
	}
	_, err := client.TTS("hello", opts)
	if err != nil {
		t.Fatalf("TTS failed: %v", err)
	}

	body := string(rawBody)
	for _, field := range []string{"temperature", "top_p", "repetition_penalty", "max_tokens", "samplingRate", "speedAlpha", "pauseBetweenBrackets"} {
		if strings.Contains(body, field) {
			t.Errorf("expected field %q to be omitted when not set, but found in body: %s", field, body)
		}
	}
}
