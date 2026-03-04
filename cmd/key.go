package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rimelabs/rime-cli/internal/config"
)

func NewKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "Print the resolved API key",
		Long:  `Print the API key resolved from config or environment, with no trailing newline.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := config.ResolveConfigWithOptions(config.ResolveOptions{
				EnvName:    ConfigEnv,
				ConfigFile: ConfigFile,
			})
			if err != nil {
				return err
			}
			fmt.Print(resolved.APIKey)
			return nil
		},
	}
}
