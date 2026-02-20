package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/api"
	"github.com/rimelabs/rime-cli/internal/output/styles"
	"github.com/rimelabs/rime-cli/internal/output/ui"
	"github.com/rimelabs/rime-cli/internal/tts"
)

func getGreeting() string {
	hour := time.Now().Hour()
	switch {
	case hour < 12:
		return "morning"
	case hour < 17:
		return "afternoon"
	default:
		return "evening"
	}
}

func NewHelloCmd() *cobra.Command {
	var output string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "hello",
		Short: "Quick demo with a friendly greeting",
		Long:  "Plays a quick TTS demo using the Astra voice",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			greeting := getGreeting()
			text := fmt.Sprintf("good %s from Rime AI!", greeting)

			opts := &api.TTSOptions{
				Speaker: "astra",
				ModelID: "arcana",
				Lang:    "eng",
			}

			shouldPlay := output == ""

			if Quiet || JSONOutput || !term.IsTerminal(int(os.Stdout.Fd())) {
				runOpts := tts.RunOptions{
					Text:       text,
					TTSOptions: opts,
					Output:     output,
					Play:       shouldPlay,
					Quiet:      Quiet,
					JSON:       JSONOutput,
					Version:    Version,
					BaseURL:    apiURL,
					ConfigEnv:  ConfigEnv,
					ConfigFile: ConfigFile,
				}
				return tts.RunNonInteractive(runOpts)
			}

			if shouldPlay {
				fmt.Fprintln(os.Stderr, styles.Dim("Playing audio (use -o to save)"))
			}

			p := tea.NewProgram(ui.NewTTSModel(text, opts, output, shouldPlay, Version, apiURL, ConfigEnv, ConfigFile))
			m, err := p.Run()
			if err != nil {
				return err
			}

			ttsM := m.(ui.TTSModel)
			if ttsM.Err() != nil {
				return ttsM.Err()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (plays by default)")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "API URL (default: $RIME_API_URL or https://users.rime.ai/v1/rime-tts)")

	return cmd
}
