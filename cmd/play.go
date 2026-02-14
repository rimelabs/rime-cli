//go:build !headless

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rimelabs/rime-cli/internal/audio/playback"
	"github.com/rimelabs/rime-cli/internal/output/ui"
)

func NewPlayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "play FILE",
		Short: "Play a WAV file",
		Long:  "Play a WAV audio file with waveform visualization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fpath := args[0]

			if _, err := os.Stat(fpath); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				absPath, absErr := filepath.Abs(fpath)
				if absErr != nil {
					return fmt.Errorf("file not found: %s", fpath)
				}
				if _, err2 := os.Stat(absPath); err2 != nil {
					return fmt.Errorf("file not found: %s", fpath)
				}
				fpath = absPath
			}

			if Quiet || !term.IsTerminal(int(os.Stdout.Fd())) {
				return playback.RunNonInteractivePlay(fpath)
			}

			p := tea.NewProgram(ui.NewPlayModel(fpath))
			m, err := p.Run()
			if err != nil {
				return err
			}

			playM := m.(ui.PlayModel)
			if playM.Err() != nil {
				return playM.Err()
			}

			return nil
		},
	}

	return cmd
}
