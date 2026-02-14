//go:build !headless

package ui

import (
	"io"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/rimelabs/rime-cli/internal/audio/analyze"
)

func (m *TTSModel) startPlayback(format beep.Format, analyzer *analyze.AmplitudeAnalyzer, body io.ReadCloser, playDone chan struct{}) error {
	err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		body.Close()
		return err
	}

	speaker.Play(beep.Seq(analyzer, beep.Callback(func() {
		body.Close()
		close(playDone)
	})))
	return nil
}
