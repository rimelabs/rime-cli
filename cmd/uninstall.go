package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func NewUninstallCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the Rime CLI and all configuration",
		Long:  "Removes the Rime CLI binary, configuration, and shell PATH setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(yes)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

type uninstallAction struct {
	description string
	execute     func() error
}

func runUninstall(yes bool) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	if isHomebrewInstall(execPath) {
		fmt.Println(styles.Info("Rime was installed via Homebrew."))
		fmt.Println(styles.Dim("To uninstall, run: brew uninstall rime"))
		return nil
	}

	// Derive install dir from binary location: ~/.rime/bin/rime -> ~/.rime
	installDir := filepath.Dir(filepath.Dir(execPath))

	shSourceLine := fmt.Sprintf(`. "%s/env.sh"`, installDir)
	fishSourceLine := fmt.Sprintf(`source "%s/env.fish"`, installDir)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	shellConfigs := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}

	var actions []uninstallAction

	// Check each shell config for our source lines
	for _, configPath := range shellConfigs {
		path := configPath // capture for closure
		sourceLine := shSourceLine
		if strings.HasSuffix(path, "config.fish") {
			sourceLine = fishSourceLine
		}
		line := sourceLine // capture for closure

		if fileContains(path, line) {
			tilde := tildifyPath(home, path)
			actions = append(actions, uninstallAction{
				description: fmt.Sprintf("Remove rime source line from %s", tilde),
				execute: func() error {
					return removeLineFromFile(path, line)
				},
			})
		}
	}

	// Check if install dir exists
	if _, err := os.Stat(installDir); err == nil {
		tilde := tildifyPath(home, installDir)
		actions = append(actions, uninstallAction{
			description: fmt.Sprintf("Remove %s/", tilde),
			execute: func() error {
				return os.RemoveAll(installDir)
			},
		})
	}

	if len(actions) == 0 {
		fmt.Println(styles.Dim("Nothing to remove."))
		return nil
	}

	fmt.Println(styles.Info("This will:"))
	fmt.Println()
	for _, a := range actions {
		fmt.Printf("  %s %s\n", styles.Dim("·"), a.description)
	}
	fmt.Println()

	if !yes {
		fmt.Print("Continue? [y/N] ")

		var answer string
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer = strings.TrimSpace(scanner.Text())
		}

		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			fmt.Println(styles.Dim("Cancelled."))
			return nil
		}
	}

	fmt.Println()
	for _, a := range actions {
		if err := a.execute(); err != nil {
			return fmt.Errorf("failed to %s: %w", a.description, err)
		}
		fmt.Println(styles.Success(a.description))
	}

	fmt.Println()
	fmt.Println(styles.Dim("Rime CLI removed."))

	return nil
}

func fileContains(path, substr string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), substr) {
			return true
		}
	}
	return false
}

// removeLineFromFile removes lines containing substr (and any immediately
// preceding "# rime" comment line) from the file at path.
func removeLineFromFile(path, substr string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	filtered := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.Contains(line, substr) {
			// Also remove a preceding "# rime" comment if present
			if len(filtered) > 0 && strings.TrimSpace(filtered[len(filtered)-1]) == "# rime" {
				filtered = filtered[:len(filtered)-1]
			}
			// And remove the preceding blank line we added during install
			if len(filtered) > 0 && strings.TrimSpace(filtered[len(filtered)-1]) == "" {
				filtered = filtered[:len(filtered)-1]
			}
			continue
		}
		filtered = append(filtered, line)
	}

	return os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0644)
}

func isHomebrewInstall(execPath string) bool {
	return strings.Contains(execPath, "/Cellar/") ||
		strings.Contains(execPath, "/homebrew/") ||
		strings.HasPrefix(execPath, "/opt/homebrew/") ||
		strings.HasPrefix(execPath, "/usr/local/opt/")
}

func tildifyPath(home, path string) string {
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~/" + path[len(home)+1:]
	}
	return path
}
