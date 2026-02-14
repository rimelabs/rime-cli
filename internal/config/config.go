package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDir = ".rime"
	tokenFile = "cli-api-token"
	EnvAPIKey = "RIME_CLI_API_KEY"
)

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

func TokenFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFile), nil
}

func SaveAPIKey(apiKey string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := TokenFilePath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(apiKey), 0600); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	return nil
}

func LoadAPIKey() (string, error) {
	if key := os.Getenv(EnvAPIKey); key != "" {
		return key, nil
	}

	path, err := TokenFilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("API key not found. Run 'rime login' to authenticate or set %s", EnvAPIKey)
		}
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func APIKeyExists() bool {
	path, err := TokenFilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
