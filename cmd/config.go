package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(NewConfigInitCmd())
	cmd.AddCommand(NewConfigAddCmd())
	cmd.AddCommand(NewConfigListCmd())
	cmd.AddCommand(NewConfigShowCmd())
	cmd.AddCommand(NewConfigRmCmd())
	cmd.AddCommand(NewConfigEditCmd())
	return cmd
}

func NewConfigAddCmd() *cobra.Command {
	var apiURL string
	var apiKey string
	var authPrefix string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a named environment to configuration",
		Long:  "Interactively adds a named environment to ~/.rime/rime.toml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			name := args[0]

			if name == "default" {
				return fmt.Errorf("cannot add an environment named %q; use 'rime config init' to modify the default", name)
			}

			isTerminal := term.IsTerminal(int(os.Stdin.Fd()))

			reader := bufio.NewReader(os.Stdin)

			if apiURL == "" {
				if isTerminal {
					fmt.Printf("API URL [https://users.rime.ai/v1/rime-tts]: ")
					line, _ := reader.ReadString('\n')
					if trimmed := strings.TrimSpace(line); trimmed != "" {
						apiURL = trimmed
					}
				}
				if apiURL == "" {
					apiURL = "https://users.rime.ai/v1/rime-tts"
				}
			}

			if apiKey == "" && isTerminal {
				fmt.Print("API key (input hidden): ")
				keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err == nil {
					apiKey = strings.TrimSpace(string(keyBytes))
				}
			}

			if authPrefix == "" && isTerminal {
				fmt.Print("Auth header prefix (optional, e.g. Bearer): ")
				line, _ := reader.ReadString('\n')
				authPrefix = strings.TrimSpace(line)
			}

			env := config.Environment{
				APIURL: apiURL,
			}
			if apiKey != "" {
				env.APIKey = &apiKey
			}
			if authPrefix != "" {
				env.AuthHeaderPrefix = &authPrefix
			}

			if err := config.SaveEnvironment(name, env); err != nil {
				return err
			}

			fmt.Println(styles.Successf("Added environment %q to config", name))
			return nil
		},
	}

	cmd.Flags().StringVar(&apiURL, "url", "", "API URL (default: https://users.rime.ai/v1/rime-tts)")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key")
	cmd.Flags().StringVar(&authPrefix, "auth-prefix", "", "Auth header prefix (e.g. Bearer)")
	return cmd
}

func NewConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long:  "Creates a new ~/.rime/rime.toml configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}

			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("Config file already exists at %s\nUse --force to overwrite", path)
			}

			dir, err := config.ConfigDir()
			if err != nil {
				return err
			}

			if err := os.MkdirAll(dir, 0700); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			var apiKey string
			if term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Print("Creating ", path, "\n\n")
				fmt.Print("Paste your API key (or press Enter to skip): ")
				keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err == nil {
					apiKey = strings.TrimSpace(string(keyBytes))
				}
			}

			cfg := fmt.Sprintf(`api_key = %q
api_url = "https://users.rime.ai/v1/rime-tts"

# [env.example]
# api_url = "https://example.rime.ai/v1/rime-tts"
`, apiKey)

			if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Println(styles.Successf("Created %s", path))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config file")
	return cmd
}

func NewConfigListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured environments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.Config
			var err error
			if ConfigFile != "" {
				cfg, err = config.LoadConfigFromPath(ConfigFile)
			} else {
				cfg, err = config.LoadConfig()
			}
			if err != nil {
				return err
			}

			if cfg == nil {
				fmt.Println("No config file found. Run 'rime config init' to create one.")
				return nil
			}

			if jsonOutput {
				return listEnvironmentsJSON(cfg)
			}

			fmt.Printf("%-15s %-50s %s\n", "NAME", "URL", "AUTH")
			fmt.Println(strings.Repeat("-", 80))

			envs := cfg.ListEnvironments()
			for _, name := range envs {
				env, err := cfg.ResolveEnvironment(name)
				if err != nil {
					continue
				}

				displayName := name
				if name == "default" {
					displayName = "default"
				}

				authStatus := ""
				if env.AuthHeaderPrefix != nil {
					authStatus = *env.AuthHeaderPrefix
				}
				if env.GetAPIKey() == "" {
					authStatus = "(no auth)"
				}

				fmt.Printf("%-15s %-50s %s\n", displayName, env.APIURL, authStatus)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func listEnvironmentsJSON(cfg *config.Config) error {
	result := make(map[string]map[string]interface{})

	envs := cfg.ListEnvironments()
	for _, name := range envs {
		env, err := cfg.ResolveEnvironment(name)
		if err != nil {
			continue
		}

		envData := map[string]interface{}{
			"api_url":            env.APIURL,
			"has_api_key":        env.GetAPIKey() != "",
			"auth_header_prefix": env.AuthHeaderPrefix,
		}

		result[name] = envData
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func NewConfigShowCmd() *cobra.Command {
	var jsonOutput bool
	var showKey bool
	var envName string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show resolved configuration",
		Long:  "Shows the fully resolved configuration for the default or specified environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if envName == "" {
				envName = ConfigEnv
			}
			if envName == "" {
				envName = "default"
			}

			resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
				EnvName:    envName,
				ConfigFile: ConfigFile,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return showConfigJSON(resolved, showKey)
			}

			fmt.Printf("Environment:  %s\n", resolved.Environment)
			fmt.Printf("API URL:      %s\n", resolved.APIURL)

			if showKey {
				fmt.Printf("API Key:      %s\n", resolved.APIKey)
			} else {
				if resolved.APIKey != "" {
					displayKey := resolved.APIKey
					if len(displayKey) > 20 {
						displayKey = displayKey[:17] + "..."
					}
					source := ""
					if resolved.APIKeySource != "config" {
						source = fmt.Sprintf(" (redacted, inherited from %s)", resolved.APIKeySource)
					} else {
						source = " (redacted)"
					}
					fmt.Printf("API Key:      %s%s\n", displayKey, source)
				} else {
					fmt.Printf("API Key:      (none)\n")
				}
			}

			fmt.Printf("Auth Prefix:  %s\n", resolved.AuthHeaderPrefix)

			if resolved.APIKey != "" && resolved.AuthHeaderPrefix != "" {
				if showKey {
					fmt.Printf("Auth Header:  Authorization: %s %s\n", resolved.AuthHeaderPrefix, resolved.APIKey)
				} else {
					fmt.Printf("Auth Header:  Authorization: %s %s\n", resolved.AuthHeaderPrefix, "(redacted)")
				}
			} else {
				fmt.Printf("Auth Header:  (none)\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&showKey, "show-key", false, "Show full API key")
	cmd.Flags().StringVarP(&envName, "env", "e", "", "Environment to show")
	return cmd
}

func NewConfigRmCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a named environment from configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			name := args[0]

			if !yes && term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Printf("Remove environment %q? [y/N]: ", name)
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if strings.ToLower(strings.TrimSpace(line)) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := config.RemoveEnvironment(name); err != nil {
				return err
			}
			fmt.Println(styles.Successf("Removed environment %q", name))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func NewConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the configuration file in an editor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("config file not found; run 'rime config init' first")
			}

			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				if p, err := exec.LookPath("nano"); err == nil {
					editor = p
				} else if p, err := exec.LookPath("vi"); err == nil {
					editor = p
				}
			}
			if editor == "" {
				return fmt.Errorf("no editor found; set $VISUAL or $EDITOR")
			}

			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func showConfigJSON(resolved *config.ResolvedConfig, showKey bool) error {
	result := map[string]interface{}{
		"environment":        resolved.Environment,
		"api_url":            resolved.APIURL,
		"has_api_key":        resolved.APIKey != "",
		"api_key_source":     resolved.APIKeySource,
		"auth_header_prefix": resolved.AuthHeaderPrefix,
	}

	if showKey {
		result["api_key"] = resolved.APIKey
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
