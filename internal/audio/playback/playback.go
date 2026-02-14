//go:build !headless

package playback

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
)

func RunNonInteractivePlay(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	contentType := detectformat.DetectFormat(data)
	if contentType == "" {
		return fmt.Errorf("unsupported audio format")
	}

	return PlayAudioData(data, contentType)
}

func PlayAudioData(data []byte, contentType string) error {
	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch contentType {
	case "audio/wav":
		reader := bytes.NewReader(data)
		decoder, f, err := stream.DecodeStreaming(reader)
		if err != nil {
			return err
		}
		streamer = &wavStreamerAdapter{decoder: decoder, rc: io.NopCloser(reader)}
		format = f
	case "audio/mpeg", "audio/mp3":
		reader := bytes.NewReader(data)
		s, f, err := stream.DecodeMP3Streaming(io.NopCloser(reader))
		if err != nil {
			return err
		}
		streamer = s
		format = f
	default:
		return fmt.Errorf("unsupported format: %s", contentType)
	}
	defer streamer.Close()

	err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		return err
	}

	done := make(chan struct{})
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		close(done)
	})))
	<-done

	return nil
}

func IsPlaybackEnabled() bool {
	return true
}
