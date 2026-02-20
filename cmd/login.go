package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/auth"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "invalid API key")
}

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

			fmt.Println(styles.Dim("Verifying API key..."))
			client := api.NewClient(apiKey, cmd.Root().Version)
			if err := client.ValidateAPIKey(); err != nil {
				// 401 means the key itself is bad — don't save it
				if isAuthError(err) {
					return fmt.Errorf("API key appears to be invalid: %w", err)
				}
				// Network or other transient error — save the key and warn
				if err := config.SaveAPIKey(apiKey); err != nil {
					return err
				}
				fmt.Println(styles.Error("Could not verify API key: " + err.Error()))
				fmt.Println(styles.Dim("Key saved anyway. Run 'rime hello' to test when ready."))
				return nil
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
