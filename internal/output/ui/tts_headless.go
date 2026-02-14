//go:build headless

package ui

import (
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/rimelabs/rime-cli/internal/audio/analyze"
)

func (m *TTSModel) startPlayback(format beep.Format, analyzer *analyze.AmplitudeAnalyzer, body io.ReadCloser, playDone chan struct{}) error {
	go func() {
		sampleBuf := make([][2]float64, 512)
		for {
			_, ok := analyzer.Stream(sampleBuf)
			if !ok {
				break
			}
		}
		body.Close()
		close(playDone)
	}()
	return nil
}
