package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func NewUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Show removal instructions",
		Long:  "Prints instructions for removing the Rime CLI and configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("could not determine executable path: %w", err)
			}
			configDir, err := config.ConfigDir()
			if err != nil {
				return fmt.Errorf("could not determine config directory path: %w", err)
			}

			fmt.Println(styles.Info("To uninstall Rime CLI, remove the following:"))
			fmt.Println()
			fmt.Printf("  Binary:  %s\n", execPath)
			fmt.Printf("  Config:  %s\n", configDir)
			fmt.Println()
			fmt.Println(styles.Dim("Run these commands:"))
			fmt.Printf("  rm %s\n", execPath)
			fmt.Printf("  rm -rf %s\n", configDir)

			return nil
		},
	}

	return cmd
}
