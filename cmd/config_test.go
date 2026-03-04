package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rimelabs/rime-cli/internal/config"
)

func setupConfigTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	return tmpDir, func() { os.Setenv("HOME", originalHome) }
}

func writeConfigFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
}

func TestConfigAddCmd_NoConfigFile(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{"staging", "--url", "https://staging.rime.ai/v1/rime-tts", "--key", "test-key"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("Expected error to mention 'config init', got: %v", err)
	}
}

func TestConfigAddCmd_RejectsDefaultName(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{"default", "--url", "https://example.rime.ai/v1/rime-tts", "--key", "k"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when using 'default' as environment name")
	}
	if !strings.Contains(err.Error(), "default") {
		t.Errorf("Expected error to mention 'default', got: %v", err)
	}
}

func TestConfigAddCmd_AddsEnvironment(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "global-key"`+"\n"+`api_url = "https://users.rime.ai/v1/rime-tts"`+"\n")

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{
		"staging",
		"--url", "https://staging.rime.ai/v1/rime-tts",
		"--key", "staging-api-key",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config add failed: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	env, ok := cfg.Env["staging"]
	if !ok {
		t.Fatal("Expected 'staging' environment to be added")
	}
	if env.APIURL != "https://staging.rime.ai/v1/rime-tts" {
		t.Errorf("Expected APIURL 'https://staging.rime.ai/v1/rime-tts', got %q", env.APIURL)
	}
	if env.GetAPIKey() != "staging-api-key" {
		t.Errorf("Expected APIKey 'staging-api-key', got %q", env.GetAPIKey())
	}
}

func TestConfigAddCmd_WithAuthPrefix(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n")

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{
		"prod",
		"--url", "https://prod.rime.ai/v1/rime-tts",
		"--key", "prod-key",
		"--auth-prefix", "Api-Key",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config add failed: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	env, ok := cfg.Env["prod"]
	if !ok {
		t.Fatal("Expected 'prod' environment to be added")
	}
	if env.AuthHeaderPrefix == nil || *env.AuthHeaderPrefix != "Api-Key" {
		t.Errorf("Expected AuthHeaderPrefix 'Api-Key', got %v", env.AuthHeaderPrefix)
	}
}

func TestConfigAddCmd_DefaultURL(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n")

	cmd := NewConfigAddCmd()
	// No --url flag: should use default URL
	cmd.SetArgs([]string{"myenv", "--key", "my-key"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config add failed: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	env, ok := cfg.Env["myenv"]
	if !ok {
		t.Fatal("Expected 'myenv' environment to be added")
	}
	if env.APIURL != "https://users.rime.ai/v1/rime-tts" {
		t.Errorf("Expected default APIURL, got %q", env.APIURL)
	}
}

func TestConfigAddCmd_OverwritesExisting(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n"+`[env.staging]`+"\n"+`api_url = "https://old.rime.ai/v1/rime-tts"`+"\n")

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{
		"staging",
		"--url", "https://new.rime.ai/v1/rime-tts",
		"--key", "new-key",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config add failed: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	env := cfg.Env["staging"]
	if env.APIURL != "https://new.rime.ai/v1/rime-tts" {
		t.Errorf("Expected updated APIURL 'https://new.rime.ai/v1/rime-tts', got %q", env.APIURL)
	}
}

func TestConfigRmCmd_RemovesEnvironment(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n"+`[env.staging]`+"\n"+`api_url = "https://staging.rime.ai/v1/rime-tts"`+"\n")

	cmd := NewConfigRmCmd()
	cmd.SetArgs([]string{"staging", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config rm failed: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if _, ok := cfg.Env["staging"]; ok {
		t.Error("Expected 'staging' environment to be removed")
	}
}

func TestConfigRmCmd_NotFound(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n")

	cmd := NewConfigRmCmd()
	cmd.SetArgs([]string{"nonexistent", "--yes"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("Expected error for nonexistent environment")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestConfigRmCmd_RequiresName(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	cmd := NewConfigRmCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when name argument is missing")
	}
}

func TestConfigEditCmd_NoConfigFile(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	cmd := NewConfigEditCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("Expected error to mention 'config init', got: %v", err)
	}
}

func TestConfigEditCmd_NoEditor(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n")

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	t.Setenv("PATH", t.TempDir()) // empty PATH with no nano/vi

	cmd := NewConfigEditCmd()
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when no editor found")
	}
	if !strings.Contains(err.Error(), "editor") {
		t.Errorf("Expected error to mention 'editor', got: %v", err)
	}
}

func TestConfigEditCmd_UsesEditorEnvVar(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	configPath, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	writeConfigFile(t, configPath, `api_key = "k"`+"\n")

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true") // 'true' exits 0 without doing anything

	cmd := NewConfigEditCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config edit failed: %v", err)
	}
}

func TestConfigAddCmd_RequiresName(t *testing.T) {
	_, cleanup := setupConfigTestDir(t)
	defer cleanup()

	cmd := NewConfigAddCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when name argument is missing")
	}
}
