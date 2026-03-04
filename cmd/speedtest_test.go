package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
	"github.com/rimelabs/rime-cli/internal/config"
)

func TestFormatTTFB(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{500 * time.Microsecond, "500.00µs"},
		{1 * time.Millisecond, "1.00ms"},
		{150 * time.Millisecond, "150.00ms"},
		{999 * time.Millisecond, "999.00ms"},
		{1 * time.Second, "1.00s"},
		{2500 * time.Millisecond, "2.50s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTTFB(tt.duration)
			if got != tt.want {
				t.Errorf("formatTTFB(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestTruncateURL(t *testing.T) {
	tests := []struct {
		url    string
		maxLen int
		want   string
	}{
		{"https://example.com", 50, "https://example.com"},
		{"https://example.com", 10, "https:/..."},
		{"https://very-long-url.example.com/path/to/endpoint", 30, "https://very-long-url.examp..."},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := truncateURL(tt.url, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateURL(%q, %d) = %q, want %q", tt.url, tt.maxLen, got, tt.want)
			}
			if len(got) > tt.maxLen {
				t.Errorf("truncateURL result length %d exceeds maxLen %d", len(got), tt.maxLen)
			}
		})
	}
}

func TestFindFastest(t *testing.T) {
	tests := []struct {
		name    string
		results []SpeedtestResult
		wantEnv string
		wantNil bool
	}{
		{
			name:    "empty results",
			results: []SpeedtestResult{},
			wantNil: true,
		},
		{
			name: "all errors",
			results: []SpeedtestResult{
				{Environment: "env1", Error: "failed"},
				{Environment: "env2", Error: "failed"},
			},
			wantNil: true,
		},
		{
			name: "single result",
			results: []SpeedtestResult{
				{Environment: "env1", TTFB: 100 * time.Millisecond},
			},
			wantEnv: "env1",
		},
		{
			name: "multiple results",
			results: []SpeedtestResult{
				{Environment: "slow", TTFB: 500 * time.Millisecond},
				{Environment: "fast", TTFB: 100 * time.Millisecond},
				{Environment: "medium", TTFB: 250 * time.Millisecond},
			},
			wantEnv: "fast",
		},
		{
			name: "mixed results with errors",
			results: []SpeedtestResult{
				{Environment: "error", Error: "failed"},
				{Environment: "slow", TTFB: 500 * time.Millisecond},
				{Environment: "fast", TTFB: 100 * time.Millisecond},
			},
			wantEnv: "fast",
		},
		{
			name: "zero TTFB ignored",
			results: []SpeedtestResult{
				{Environment: "zero", TTFB: 0},
				{Environment: "valid", TTFB: 100 * time.Millisecond},
			},
			wantEnv: "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFastest(tt.results)
			if tt.wantNil {
				if got != nil {
					t.Errorf("findFastest() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("findFastest() = nil, want non-nil")
			}
			if got.Environment != tt.wantEnv {
				t.Errorf("findFastest().Environment = %q, want %q", got.Environment, tt.wantEnv)
			}
		})
	}
}

func TestSpeedtest_MultipleEndpoints(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server2.Close()

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configContent := `api_key = "test-key"
api_url = "` + server1.URL + `"

[env.slow]
api_url = "` + server2.URL + `"
`
	configDir := tmpDir + "/.rime"
	os.MkdirAll(configDir, 0700)
	os.WriteFile(configDir+"/rime.toml", []byte(configContent), 0600)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	envs := cfg.ListEnvironments()
	if len(envs) != 2 {
		t.Fatalf("Expected 2 environments, got %d", len(envs))
	}

	Version = "test-version"
	Quiet = true
	JSONOutput = false

	cmd := NewSpeedtestCmd()
	var output strings.Builder
	cmd.SetOut(&output)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Speedtest command failed: %v", err)
	}
}

func TestSpeedtest_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	Quiet = false
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no config file exists")
	}
	if !strings.Contains(err.Error(), "no config file found") {
		t.Errorf("Expected 'no config file found' error, got: %v", err)
	}
}

func TestSpeedtest_JSONOutput(t *testing.T) {
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

	configContent := `api_key = "test-key"
api_url = "` + server.URL + `"
`
	configDir := tmpDir + "/.rime"
	os.MkdirAll(configDir, 0700)
	os.WriteFile(configDir+"/rime.toml", []byte(configContent), 0600)

	Version = "test-version"
	Quiet = false
	JSONOutput = true
	ConfigFile = ""

	cmd := NewSpeedtestCmd()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Speedtest command failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "environment") {
		t.Error("JSON output should contain 'environment' field")
	}
	if !strings.Contains(output, "ttfb_ms") {
		t.Error("JSON output should contain 'ttfb_ms' field")
	}
}

// captureRequest returns an HTTP handler that stores the last request body.
func captureRequest(t *testing.T, wavData []byte) (*httptest.Server, func() map[string]interface{}) {
	t.Helper()
	var mu sync.Mutex
	var lastBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		lastBody = body
		mu.Unlock()
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	getReq := func() map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		if lastBody == nil {
			return nil
		}
		var m map[string]interface{}
		json.Unmarshal(lastBody, &m)
		return m
	}
	return server, getReq
}

func setupSpeedtestConfig(t *testing.T, apiURL string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := tmpDir + "/.rime"
	os.MkdirAll(configDir, 0700)
	configContent := "api_key = \"test-key\"\napi_url = \"" + apiURL + "\"\n"
	os.WriteFile(configDir+"/rime.toml", []byte(configContent), 0600)
}

func setupSpeedtestConfigWithEnv(t *testing.T, defaultURL, slowURL string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := tmpDir + "/.rime"
	os.MkdirAll(configDir, 0700)
	configContent := "api_key = \"test-key\"\napi_url = \"" + defaultURL + "\"\n\n[env.slow]\napi_url = \"" + slowURL + "\"\n"
	os.WriteFile(configDir+"/rime.toml", []byte(configContent), 0600)
}

func TestSpeedtest_ModelFlag(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)
	server, getReq := captureRequest(t, wavData)
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--model", "arcanav2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	req := getReq()
	if req == nil {
		t.Fatal("no request captured")
	}
	if got, _ := req["modelId"].(string); got != "arcanav2" {
		t.Errorf("expected modelId=arcanav2, got %q", got)
	}
}

func TestSpeedtest_SpeakerFlag(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)
	server, getReq := captureRequest(t, wavData)
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--speaker", "luna"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	req := getReq()
	if req == nil {
		t.Fatal("no request captured")
	}
	if got, _ := req["speaker"].(string); got != "luna" {
		t.Errorf("expected speaker=luna, got %q", got)
	}
}

func TestSpeedtest_URLFlag(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	// Default config server — should NOT be called when only --url is given
	var configHits atomic.Int32
	configServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer configServer.Close()

	var urlHits atomic.Int32
	urlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer urlServer.Close()

	setupSpeedtestConfig(t, configServer.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--url", urlServer.URL})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if configHits.Load() != 0 {
		t.Error("config server should NOT be called when only --url is given")
	}
	if urlHits.Load() == 0 {
		t.Error("--url server should have been called")
	}
}

func TestSpeedtest_URLFlagWithEnv(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	var defaultHits atomic.Int32
	defaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer defaultServer.Close()

	var urlHits atomic.Int32
	urlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer urlServer.Close()

	setupSpeedtestConfig(t, defaultServer.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--url", urlServer.URL, "--env", "default"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if defaultHits.Load() == 0 {
		t.Error("default env should be called when --env default is specified")
	}
	if urlHits.Load() == 0 {
		t.Error("--url server should have been called")
	}
}

func TestSpeedtest_EnvFilter(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	var defaultHits atomic.Int32
	defaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer defaultServer.Close()

	var slowHits atomic.Int32
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slowHits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer slowServer.Close()

	setupSpeedtestConfigWithEnv(t, defaultServer.URL, slowServer.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--env", "slow"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if defaultHits.Load() != 0 {
		t.Error("default env should NOT have been called when --env slow is specified")
	}
	if slowHits.Load() == 0 {
		t.Error("slow env should have been called")
	}
}

func TestComputeStats(t *testing.T) {
	tests := []struct {
		name    string
		ttfbs   []time.Duration
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:    "single value",
			ttfbs:   []time.Duration{100 * time.Millisecond},
			wantMin: 100 * time.Millisecond,
			wantMax: 100 * time.Millisecond,
		},
		{
			name:    "multiple values",
			ttfbs:   []time.Duration{300 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond},
			wantMin: 100 * time.Millisecond,
			wantMax: 300 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var total time.Duration
			for _, d := range tt.ttfbs {
				total += d
			}
			wantMean := total / time.Duration(len(tt.ttfbs))

			gotMean, gotMin, gotMax := computeStats(tt.ttfbs)
			if gotMean != wantMean {
				t.Errorf("mean: got %v, want %v", gotMean, wantMean)
			}
			if gotMin != tt.wantMin {
				t.Errorf("min: got %v, want %v", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("max: got %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}

func TestSpeedtest_RunsFlag(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = false
	JSONOutput = true
	ConfigFile = ""

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--runs", "3"})
	if err := cmd.Execute(); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("command failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	if hits.Load() != 3 {
		t.Errorf("expected 3 requests, got %d", hits.Load())
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "ttfb_min_ms") {
		t.Error("JSON output should contain 'ttfb_min_ms' when --runs > 1")
	}
	if !strings.Contains(output, "ttfb_max_ms") {
		t.Error("JSON output should contain 'ttfb_max_ms' when --runs > 1")
	}
}

func TestSpeedtest_RunsFlag_Single(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)
	server, _ := captureRequest(t, wavData)
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = false
	JSONOutput = true
	ConfigFile = ""

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewSpeedtestCmd()
	// default --runs 1
	if err := cmd.Execute(); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("command failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// min/max should be absent with single run
	if strings.Contains(output, "ttfb_min_ms") {
		t.Error("JSON output should NOT contain 'ttfb_min_ms' when --runs == 1")
	}
}

func TestSpeedtest_TimeoutFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang long enough to trigger a short timeout
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--timeout", "50ms"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command should not error (timeout reported as result): %v", err)
	}
}

func TestSpeedtest_RunsInvalidFlag(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)
	server, _ := captureRequest(t, wavData)
	defer server.Close()

	setupSpeedtestConfig(t, server.URL)
	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--runs", "0"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for --runs 0")
	}
	if !strings.Contains(err.Error(), "--runs must be at least 1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSpeedtest_URLFlag_NoConfig(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write(wavData)
	}))
	defer server.Close()

	// Empty home dir — no config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	Version = "test-version"
	Quiet = true
	JSONOutput = false
	ConfigFile = ""

	cmd := NewSpeedtestCmd()
	cmd.SetArgs([]string{"--url", server.URL})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success with --url and no config, got: %v", err)
	}
	if hits.Load() == 0 {
		t.Error("--url server should have been called")
	}
}
