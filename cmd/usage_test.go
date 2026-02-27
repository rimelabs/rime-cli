package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rimelabs/rime-cli/internal/api"
)

const usageResponse = `{"data":[
	{"day":"2026-02-16","mistChars":155929,"arcanaChars":2357},
	{"day":"2026-02-17","mistChars":158621,"arcanaChars":1313},
	{"day":"2026-02-22","mistChars":86244,"arcanaChars":6687}
]}`

func setupUsageTest(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Setenv(api.EnvOptimizeURL, srv.URL)
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	configDir := tmpDir + "/.rime"
	os.MkdirAll(configDir, 0700)
	os.WriteFile(configDir+"/rime.toml", []byte(`api_key = "test-key"`), 0600)

	Version = "test"
	JSONOutput = false
	ConfigFile = ""
	ConfigEnv = ""
}

func captureUsageOutput(t *testing.T, args ...string) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	cmd := NewUsageCmd()
	cmd.SetArgs(args)
	runErr := cmd.Execute()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 1<<16)
	n, _ := r.Read(buf)
	return string(buf[:n]), runErr
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{155929, "155,929"},
		{1000000, "1,000,000"},
		{713535, "713,535"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestUsage_NoAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("RIME_CLI_API_KEY", "")
	ConfigFile = ""
	ConfigEnv = ""

	cmd := NewUsageCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no API key configured")
	}
	if !strings.Contains(err.Error(), "no API key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUsage_TableOutput(t *testing.T) {
	setupUsageTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(usageResponse))
	})

	out, err := captureUsageOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "2026-02-16") {
		t.Error("output missing date")
	}
	if !strings.Contains(out, "155,929") {
		t.Error("output missing formatted mist chars")
	}
	if !strings.Contains(out, "158,286") { // 155929 + 2357
		t.Error("output missing formatted total")
	}
	for _, col := range []string{"Day", "Mist Chars", "Arcana Chars", "Total"} {
		if !strings.Contains(out, col) {
			t.Errorf("output missing header column %q", col)
		}
	}
}

func TestUsage_JSONOutput(t *testing.T) {
	setupUsageTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(usageResponse))
	})
	JSONOutput = true
	defer func() { JSONOutput = false }()

	out, err := captureUsageOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result api.UsageHistory
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(result.Data) != 3 {
		t.Errorf("expected 3 days, got %d", len(result.Data))
	}
	if result.Data[0].Day != "2026-02-16" {
		t.Errorf("unexpected first day: %s", result.Data[0].Day)
	}
	if result.Data[0].MistChars != 155929 {
		t.Errorf("unexpected mist chars: %d", result.Data[0].MistChars)
	}
}

func TestUsage_CSVOutput(t *testing.T) {
	setupUsageTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(usageResponse))
	})

	out, err := captureUsageOutput(t, "--csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if lines[0] != "day,mist_chars,arcana_chars,total" {
		t.Errorf("unexpected CSV header: %s", lines[0])
	}
	if len(lines) != 4 { // header + 3 data rows
		t.Errorf("expected 4 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[1], "2026-02-16,155929,2357,158286") {
		t.Errorf("unexpected first data row: %s", lines[1])
	}
}

func TestUsage_Unauthorized(t *testing.T) {
	setupUsageTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := captureUsageOutput(t)
	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("unexpected error: %v", err)
	}
}
