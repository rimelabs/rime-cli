//go:build !headless

package playback

import (
	"fmt"
	"io"

	"github.com/rimelabs/rime-cli/internal/audio/stream"
)

type wavStreamerAdapter struct {
	decoder *stream.StreamingDecoder
	rc      io.ReadCloser
}

func (w *wavStreamerAdapter) Stream(samples [][2]float64) (n int, ok bool) {
	return w.decoder.Stream(samples)
}

func (w *wavStreamerAdapter) Err() error {
	return w.decoder.Err()
}

// Len returns -1 to indicate unknown length for streaming WAV data.
// This is the key part that enables streaming - returning -1 signals
// to the audio library that the stream length is not known upfront.
func (w *wavStreamerAdapter) Len() int {
	return -1
}

// Position returns -1 to indicate that position tracking is not
// meaningful for streaming WAV data. This is required for streaming
// support - returning -1 signals that position cannot be determined.
func (w *wavStreamerAdapter) Position() int {
	return -1
}

// Seek is not supported for streaming WAV data since we cannot
// rewind or jump to arbitrary positions in a stream.
func (w *wavStreamerAdapter) Seek(p int) error {
	return fmt.Errorf("seek not supported")
}

func (w *wavStreamerAdapter) Close() error {
	if w.rc != nil {
		return w.rc.Close()
	}
	return nil
}
