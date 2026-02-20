package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pelletier/go-toml/v2"
)

const (
	configDir           = ".rime"
	configFile          = "rime.toml"
	EnvAPIKey           = "RIME_CLI_API_KEY"
	EnvAPIURL           = "RIME_API_URL"
	EnvAuthHeaderPrefix = "RIME_AUTH_HEADER_PREFIX"
	defaultAPIURL       = "https://users.rime.ai/v1/rime-tts"
	defaultAuthPrefix   = "Bearer"
)

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

func SaveAPIKey(apiKey string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	if cfg == nil {
		cfg = &Config{
			Env: make(map[string]Environment),
		}
	}

	cfg.APIKey = apiKey
	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	}
	if cfg.AuthHeaderPrefix == nil {
		prefix := defaultAuthPrefix
		cfg.AuthHeaderPrefix = &prefix
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

type Environment struct {
	APIKey           *string `toml:"api_key,omitempty"`
	APIURL           string  `toml:"api_url"`
	AuthHeaderPrefix *string `toml:"auth_header_prefix,omitempty"`
}

func (e *Environment) GetAPIKey() string {
	if e.APIKey == nil {
		return ""
	}
	return *e.APIKey
}

type Config struct {
	APIKey           string                 `toml:"api_key"`
	APIURL           string                 `toml:"api_url"`
	AuthHeaderPrefix *string                `toml:"auth_header_prefix,omitempty"`
	Env              map[string]Environment `toml:"env"`
}

func LoadConfig() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	return LoadConfigFromPath(path)
}

func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Env == nil {
		cfg.Env = make(map[string]Environment)
	}

	return &cfg, nil
}

func (c *Config) ResolveEnvironment(name string) (*Environment, error) {
	if name == "" {
		name = "default"
	}

	if name != "default" && c == nil {
		return nil, fmt.Errorf("environment %q not found in config", name)
	}

	env := Environment{
		APIURL:           defaultAPIURL,
		AuthHeaderPrefix: &[]string{defaultAuthPrefix}[0],
	}

	if c != nil {
		if c.APIURL != "" {
			env.APIURL = c.APIURL
		}
		if c.AuthHeaderPrefix != nil {
			env.AuthHeaderPrefix = c.AuthHeaderPrefix
		}
		if c.APIKey != "" {
			env.APIKey = &c.APIKey
		}

		if name != "default" {
			if envCfg, ok := c.Env[name]; ok {
				if envCfg.APIURL != "" {
					env.APIURL = envCfg.APIURL
				}
				if envCfg.AuthHeaderPrefix != nil {
					env.AuthHeaderPrefix = envCfg.AuthHeaderPrefix
				}
				if envCfg.APIKey != nil {
					env.APIKey = envCfg.APIKey
				}
			} else {
				return nil, fmt.Errorf("environment %q not found in config", name)
			}
		}
	}

	if apiURL := os.Getenv(EnvAPIURL); apiURL != "" {
		env.APIURL = apiURL
	}
	if apiKey := os.Getenv(EnvAPIKey); apiKey != "" {
		env.APIKey = &apiKey
	}
	if prefix := os.Getenv(EnvAuthHeaderPrefix); prefix != "" {
		env.AuthHeaderPrefix = &prefix
	}

	return &env, nil
}

func (c *Config) ListEnvironments() []string {
	envs := []string{"default"}
	if c == nil {
		return envs
	}

	for name := range c.Env {
		envs = append(envs, name)
	}
	sort.Strings(envs[1:])
	return envs
}

type ResolvedConfig struct {
	Environment      string
	APIURL           string
	APIKey           string
	AuthHeaderPrefix string
	APIKeySource     string
}

type ResolveOptions struct {
	EnvName        string
	APIURLOverride string
	ConfigFile     string
}

func ResolveConfig(envName string, apiURLOverride string) (*ResolvedConfig, error) {
	return ResolveConfigWithOptions(ResolveOptions{
		EnvName:        envName,
		APIURLOverride: apiURLOverride,
	})
}

func ResolveConfigWithOptions(opts ResolveOptions) (*ResolvedConfig, error) {
	var cfg *Config
	var err error

	if opts.ConfigFile != "" {
		cfg, err = LoadConfigFromPath(opts.ConfigFile)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			return nil, fmt.Errorf("config file not found: %s", opts.ConfigFile)
		}
	} else {
		cfg, err = LoadConfig()
		if err != nil {
			return nil, err
		}
	}

	envName := opts.EnvName
	if envName == "" {
		envName = "default"
	}

	env, err := cfg.ResolveEnvironment(envName)
	if err != nil {
		return nil, err
	}

	if opts.APIURLOverride != "" {
		env.APIURL = opts.APIURLOverride
	}

	apiKey := env.GetAPIKey()

	apiKeySource := "config"
	if apiKey == "" {
		apiKeySource = "none"
	} else if os.Getenv(EnvAPIKey) != "" {
		apiKeySource = "environment"
	}

	authPrefix := ""
	if env.AuthHeaderPrefix != nil {
		authPrefix = *env.AuthHeaderPrefix
	}

	return &ResolvedConfig{
		Environment:      envName,
		APIURL:           env.APIURL,
		APIKey:           apiKey,
		AuthHeaderPrefix: authPrefix,
		APIKeySource:     apiKeySource,
	}, nil
}
