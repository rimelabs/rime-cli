package cmd

import (
	"github.com/spf13/cobra"
)

var Quiet bool
var JSONOutput bool
var Version string

func NewRootCmd(version string) *cobra.Command {
	Version = version
	root := &cobra.Command{
		Use:           "rime",
		Short:         "Rime TTS CLI",
		Long:          "Command-line interface for Rime text-to-speech synthesis",
		Version:       version,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Suppress non-essential output")
	root.PersistentFlags().BoolVar(&JSONOutput, "json", false, "Output results as JSON")

	root.AddCommand(NewLoginCmd())
	root.AddCommand(NewLogoutCmd())
	root.AddCommand(NewCurlCmd())
	root.AddCommand(NewTTSCmd())
	root.AddCommand(NewHelloCmd())
	root.AddCommand(NewPlayCmd())
	root.AddCommand(NewUninstallCmd())

	return root
}
