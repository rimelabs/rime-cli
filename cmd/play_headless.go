//go:build headless

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewPlayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "play",
		Short: "Play command not available in headless build",
		Long:  "The play command requires audio support. Use a full build or save audio with 'tts -o FILE'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("play command requires audio support. Use a full build or save audio with 'tts -o FILE'")
		},
	}
}
