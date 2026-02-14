package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/config"
	"github.com/rimelabs/rime-cli/internal/output/styles"
)

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func NewLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your Rime API key",
		Long:  "Opens the Rime dashboard to get your API key and saves it locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dashboardURL := api.GetDashboardURL() + "/tokens/copy"
			fmt.Printf("Opening %s in your browser...\n\n", dashboardURL)
			time.Sleep(500 * time.Millisecond)
			openBrowser(dashboardURL)

			fmt.Print("Paste your API key: ")
			keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}

			key := strings.TrimSpace(string(keyBytes))
			if key == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			if err := config.SaveAPIKey(key); err != nil {
				return err
			}

			fmt.Println(styles.Success("API key saved successfully!"))
			fmt.Println(styles.Dim("Try: rime hello"))
			return nil
		},
	}

	return cmd
}
