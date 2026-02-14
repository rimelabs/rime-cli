package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	apiKey := "test-api-key-12345"

	if err := SaveAPIKey(apiKey); err != nil {
		t.Fatalf("SaveAPIKey failed: %v", err)
	}

	loaded, err := LoadAPIKey()
	if err != nil {
		t.Fatalf("LoadAPIKey failed: %v", err)
	}

	if loaded != apiKey {
		t.Errorf("Expected %q, got %q", apiKey, loaded)
	}
}

func TestAPIKeyExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	if APIKeyExists() {
		t.Error("APIKeyExists should return false when no key exists")
	}

	if err := SaveAPIKey("test-key"); err != nil {
		t.Fatalf("SaveAPIKey failed: %v", err)
	}

	if !APIKeyExists() {
		t.Error("APIKeyExists should return true when key exists")
	}
}

func TestLoadAPIKeyNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_CLI_API_KEY")

	_, err := LoadAPIKey()
	if err == nil {
		t.Error("Expected error when API key not found")
	}
	if err != nil && !strings.Contains(err.Error(), "rime login") && !strings.Contains(err.Error(), "RIME_CLI_API_KEY") {
		t.Errorf("Error should mention 'rime login' or RIME_CLI_API_KEY, got: %v", err)
	}
}

func TestSaveAPIKey_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir failed: %v", err)
	}

	if _, err := os.Stat(configPath); err == nil {
		t.Fatal("Config directory should not exist before SaveAPIKey")
	}

	if err := SaveAPIKey("test-key"); err != nil {
		t.Fatalf("SaveAPIKey failed: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config directory should be created: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}

	if info.Mode().Perm()&0700 != 0700 {
		t.Errorf("Config directory should have 0700 permissions, got: %v", info.Mode().Perm())
	}
}

func TestConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir failed: %v", err)
	}

	expected := filepath.Join(tmpDir, configDir)
	if dir != expected {
		t.Errorf("Expected %q, got %q", expected, dir)
	}
}
