package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/auth"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your Rime API key",
		Long:  "Opens your browser to authenticate with Rime and saves your API key locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Opening your browser to authenticate...")
			fmt.Println(styles.Dim("Waiting for authentication... (Ctrl+C to cancel)"))

			apiKey, err := auth.Login(api.GetDashboardURL())
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			if err := config.SaveAPIKey(apiKey); err != nil {
				return err
			}

			fmt.Println(styles.Success("Logged in successfully!"))
			fmt.Println(styles.Dim("Try: rime hello"))
			return nil
		},
	}
}
