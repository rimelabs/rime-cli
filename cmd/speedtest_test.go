package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
