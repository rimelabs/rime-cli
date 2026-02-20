package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func NewLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove your saved API key",
		Long:  "Removes the locally saved API key, requiring you to run 'rime login' again",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.TokenFilePath()
			if err != nil {
				return fmt.Errorf("could not determine token path: %w", err)
			}

			if err := os.Remove(path); err != nil {
				if os.IsNotExist(err) {
					fmt.Println(styles.Dim("Not logged in."))
					return nil
				}
				return fmt.Errorf("failed to remove API key: %w", err)
			}

			fmt.Println(styles.Success("Logged out."))
			return nil
		},
	}
}
