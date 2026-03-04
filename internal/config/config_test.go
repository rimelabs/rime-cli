package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_CLI_API_KEY")

	apiKey := "test-api-key-12345"

	if err := SaveAPIKey(apiKey); err != nil {
		t.Fatalf("SaveAPIKey failed: %v", err)
	}

	resolved, err := ResolveConfig("default", "")
	if err != nil {
		t.Fatalf("ResolveConfig failed: %v", err)
	}

	if resolved.APIKey != apiKey {
		t.Errorf("Expected %q, got %q", apiKey, resolved.APIKey)
	}
}

func TestSaveAPIKey_CreatesConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configFilePath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}

	if _, err := os.Stat(configFilePath); err == nil {
		t.Fatal("Config file should not exist before SaveAPIKey")
	}

	if err := SaveAPIKey("test-key"); err != nil {
		t.Fatalf("SaveAPIKey failed: %v", err)
	}

	if _, err := os.Stat(configFilePath); err != nil {
		t.Fatalf("Config file should be created: %v", err)
	}

	info, err := os.Stat(configFilePath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	if info.Mode().Perm()&0600 != 0600 {
		t.Errorf("Config file should have 0600 permissions, got: %v", info.Mode().Perm())
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

func TestLoadConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should not error when file doesn't exist: %v", err)
	}
	if cfg != nil {
		t.Error("LoadConfig should return nil when file doesn't exist")
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}

	configContent := `api_key = "test-key"
api_url = "https://test.rime.ai/v1/rime-tts"
auth_header_prefix = "Bearer"

[env.test]
api_url = "https://test-env.rime.ai/v1/rime-tts"
api_key = "test-env-key"
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig should return config when file exists")
	}

	if cfg.APIKey != "test-key" {
		t.Errorf("Expected api_key 'test-key', got %q", cfg.APIKey)
	}
	if cfg.APIURL != "https://test.rime.ai/v1/rime-tts" {
		t.Errorf("Expected api_url 'https://test.rime.ai/v1/rime-tts', got %q", cfg.APIURL)
	}

	testEnv, ok := cfg.Env["test"]
	if !ok {
		t.Fatal("Expected 'test' environment to exist")
	}
	if testEnv.APIURL != "https://test-env.rime.ai/v1/rime-tts" {
		t.Errorf("Expected test env api_url 'https://test-env.rime.ai/v1/rime-tts', got %q", testEnv.APIURL)
	}
	if testEnv.GetAPIKey() != "test-env-key" {
		t.Errorf("Expected test env api_key 'test-env-key', got %q", testEnv.GetAPIKey())
	}
	if testEnv.AuthHeaderPrefix != nil {
		t.Errorf("Expected test env auth_header_prefix to be nil (not specified), got %q", *testEnv.AuthHeaderPrefix)
	}
}

func TestResolveEnvironment_Default(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_API_URL")
	os.Unsetenv("RIME_CLI_API_KEY")
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	env, err := cfg.ResolveEnvironment("default")
	if err != nil {
		t.Fatalf("ResolveEnvironment failed: %v", err)
	}

	if env.APIURL != defaultAPIURL {
		t.Errorf("Expected APIURL %q, got %q", defaultAPIURL, env.APIURL)
	}
	if env.AuthHeaderPrefix == nil || *env.AuthHeaderPrefix != defaultAuthPrefix {
		prefix := ""
		if env.AuthHeaderPrefix != nil {
			prefix = *env.AuthHeaderPrefix
		}
		t.Errorf("Expected AuthHeaderPrefix %q, got %q", defaultAuthPrefix, prefix)
	}
}

func TestResolveEnvironment_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_API_URL")
	os.Unsetenv("RIME_CLI_API_KEY")
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}

	configContent := `api_key = "global-key"
api_url = "https://global.rime.ai/v1/rime-tts"
auth_header_prefix = "Api-Key"

[env.custom]
api_url = "https://custom.rime.ai/v1/rime-tts"
api_key = "custom-key"
auth_header_prefix = "Bearer"
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	env, err := cfg.ResolveEnvironment("custom")
	if err != nil {
		t.Fatalf("ResolveEnvironment failed: %v", err)
	}

	if env.APIURL != "https://custom.rime.ai/v1/rime-tts" {
		t.Errorf("Expected APIURL 'https://custom.rime.ai/v1/rime-tts', got %q", env.APIURL)
	}
	if env.GetAPIKey() != "custom-key" {
		t.Errorf("Expected APIKey 'custom-key', got %q", env.GetAPIKey())
	}
	if env.AuthHeaderPrefix == nil || *env.AuthHeaderPrefix != "Bearer" {
		prefix := ""
		if env.AuthHeaderPrefix != nil {
			prefix = *env.AuthHeaderPrefix
		}
		t.Errorf("Expected AuthHeaderPrefix 'Bearer', got %q", prefix)
	}
}

func TestResolveEnvironment_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	_, err = cfg.ResolveEnvironment("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent environment")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestResolveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_API_URL")
	os.Unsetenv("RIME_CLI_API_KEY")
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	resolved, err := ResolveConfig("default", "")
	if err != nil {
		t.Fatalf("ResolveConfig failed: %v", err)
	}

	if resolved.Environment != "default" {
		t.Errorf("Expected Environment 'default', got %q", resolved.Environment)
	}
	if resolved.APIURL != defaultAPIURL {
		t.Errorf("Expected APIURL %q, got %q", defaultAPIURL, resolved.APIURL)
	}
}

func TestResolveConfig_WithOverride(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)
	os.Unsetenv("RIME_API_URL")
	os.Unsetenv("RIME_CLI_API_KEY")
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	overrideURL := "https://override.rime.ai/v1/rime-tts"
	resolved, err := ResolveConfig("default", overrideURL)
	if err != nil {
		t.Fatalf("ResolveConfig failed: %v", err)
	}

	if resolved.APIURL != overrideURL {
		t.Errorf("Expected APIURL %q, got %q", overrideURL, resolved.APIURL)
	}
}

func TestLoadConfigFromPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.toml")

	configContent := `api_key = "custom-file-key"
api_url = "https://custom-file.rime.ai/v1/rime-tts"

[env.staging]
api_url = "https://staging.rime.ai/v1/rime-tts"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfigFromPath should return config")
	}

	if cfg.APIKey != "custom-file-key" {
		t.Errorf("Expected api_key 'custom-file-key', got %q", cfg.APIKey)
	}
	if cfg.APIURL != "https://custom-file.rime.ai/v1/rime-tts" {
		t.Errorf("Expected api_url 'https://custom-file.rime.ai/v1/rime-tts', got %q", cfg.APIURL)
	}
}

func TestLoadConfigFromPath_NotFound(t *testing.T) {
	cfg, err := LoadConfigFromPath("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("LoadConfigFromPath should not error for missing file: %v", err)
	}
	if cfg != nil {
		t.Error("LoadConfigFromPath should return nil for missing file")
	}
}

func TestResolveConfigWithOptions_CustomFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("RIME_API_URL")
	os.Unsetenv("RIME_CLI_API_KEY")
	os.Unsetenv("RIME_AUTH_HEADER_PREFIX")

	configPath := filepath.Join(tmpDir, "myconfig.toml")
	configContent := `api_key = "file-key"
api_url = "https://file.rime.ai/v1/rime-tts"
auth_header_prefix = "Api-Key"

[env.prod]
api_url = "https://prod.rime.ai/v1/rime-tts"
api_key = "prod-key"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	resolved, err := ResolveConfigWithOptions(ResolveOptions{
		EnvName:    "default",
		ConfigFile: configPath,
	})
	if err != nil {
		t.Fatalf("ResolveConfigWithOptions failed: %v", err)
	}

	if resolved.APIKey != "file-key" {
		t.Errorf("Expected APIKey 'file-key', got %q", resolved.APIKey)
	}
	if resolved.APIURL != "https://file.rime.ai/v1/rime-tts" {
		t.Errorf("Expected APIURL 'https://file.rime.ai/v1/rime-tts', got %q", resolved.APIURL)
	}
	if resolved.AuthHeaderPrefix != "Api-Key" {
		t.Errorf("Expected AuthHeaderPrefix 'Api-Key', got %q", resolved.AuthHeaderPrefix)
	}

	resolved, err = ResolveConfigWithOptions(ResolveOptions{
		EnvName:    "prod",
		ConfigFile: configPath,
	})
	if err != nil {
		t.Fatalf("ResolveConfigWithOptions failed for prod env: %v", err)
	}

	if resolved.APIKey != "prod-key" {
		t.Errorf("Expected APIKey 'prod-key', got %q", resolved.APIKey)
	}
	if resolved.APIURL != "https://prod.rime.ai/v1/rime-tts" {
		t.Errorf("Expected APIURL 'https://prod.rime.ai/v1/rime-tts', got %q", resolved.APIURL)
	}
}

func TestResolveConfigWithOptions_MissingCustomFile(t *testing.T) {
	_, err := ResolveConfigWithOptions(ResolveOptions{
		EnvName:    "default",
		ConfigFile: "/nonexistent/config.toml",
	})
	if err == nil {
		t.Error("Expected error for missing custom config file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestSaveEnvironment_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	env := Environment{
		APIURL: "https://staging.rime.ai/v1/rime-tts",
	}
	err := SaveEnvironment("staging", env)
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("Expected error to mention 'config init', got: %v", err)
	}
}

func TestSaveEnvironment_AddsNewEnv(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	initialContent := `api_key = "global-key"
api_url = "https://users.rime.ai/v1/rime-tts"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	apiKey := "staging-key"
	prefix := "Bearer"
	env := Environment{
		APIURL:           "https://staging.rime.ai/v1/rime-tts",
		APIKey:           &apiKey,
		AuthHeaderPrefix: &prefix,
	}
	if err := SaveEnvironment("staging", env); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected config to exist")
	}

	saved, ok := cfg.Env["staging"]
	if !ok {
		t.Fatal("Expected 'staging' environment to exist")
	}
	if saved.APIURL != "https://staging.rime.ai/v1/rime-tts" {
		t.Errorf("Expected APIURL 'https://staging.rime.ai/v1/rime-tts', got %q", saved.APIURL)
	}
	if saved.GetAPIKey() != "staging-key" {
		t.Errorf("Expected APIKey 'staging-key', got %q", saved.GetAPIKey())
	}
	if saved.AuthHeaderPrefix == nil || *saved.AuthHeaderPrefix != "Bearer" {
		t.Errorf("Expected AuthHeaderPrefix 'Bearer', got %v", saved.AuthHeaderPrefix)
	}
	// Global key should be preserved
	if cfg.APIKey != "global-key" {
		t.Errorf("Global API key should be preserved, got %q", cfg.APIKey)
	}
}

func TestSaveEnvironment_OverwritesExistingEnv(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	initialContent := `api_key = "global-key"
api_url = "https://users.rime.ai/v1/rime-tts"

[env.staging]
api_url = "https://old-staging.rime.ai/v1/rime-tts"
api_key = "old-key"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	newKey := "new-staging-key"
	env := Environment{
		APIURL: "https://new-staging.rime.ai/v1/rime-tts",
		APIKey: &newKey,
	}
	if err := SaveEnvironment("staging", env); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	saved, ok := cfg.Env["staging"]
	if !ok {
		t.Fatal("Expected 'staging' environment to exist")
	}
	if saved.APIURL != "https://new-staging.rime.ai/v1/rime-tts" {
		t.Errorf("Expected updated APIURL, got %q", saved.APIURL)
	}
	if saved.GetAPIKey() != "new-staging-key" {
		t.Errorf("Expected updated APIKey 'new-staging-key', got %q", saved.GetAPIKey())
	}
}

func TestRemoveEnvironment_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	err := RemoveEnvironment("staging")
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("Expected error to mention 'config init', got: %v", err)
	}
}

func TestRemoveEnvironment_RemovesEnv(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	content := `api_key = "global-key"
api_url = "https://users.rime.ai/v1/rime-tts"

[env.staging]
api_url = "https://staging.rime.ai/v1/rime-tts"

[env.prod]
api_url = "https://prod.rime.ai/v1/rime-tts"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if err := RemoveEnvironment("staging"); err != nil {
		t.Fatalf("RemoveEnvironment failed: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if _, ok := cfg.Env["staging"]; ok {
		t.Error("Expected 'staging' environment to be removed")
	}
	if _, ok := cfg.Env["prod"]; !ok {
		t.Error("Expected 'prod' environment to be preserved")
	}
	if cfg.APIKey != "global-key" {
		t.Errorf("Expected global API key to be preserved, got %q", cfg.APIKey)
	}
}

func TestRemoveEnvironment_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`api_key = "k"`+"\n"), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	err = RemoveEnvironment("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent environment")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to mention 'not found', got: %v", err)
	}
}

func TestRemoveEnvironment_RejectsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`api_key = "k"`+"\n"), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	err = RemoveEnvironment("default")
	if err == nil {
		t.Fatal("Expected error when trying to remove default environment")
	}
	if !strings.Contains(err.Error(), "cannot remove the default") {
		t.Errorf("Expected error about removing default, got: %v", err)
	}
}

func TestSaveEnvironment_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	configPath, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`api_key = "k"`+"\n"), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	env := Environment{APIURL: "https://test.rime.ai/v1/rime-tts"}
	if err := SaveEnvironment("test", env); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config: %v", err)
	}
	if info.Mode().Perm()&0600 != 0600 {
		t.Errorf("Expected 0600 permissions, got %v", info.Mode().Perm())
	}
}
