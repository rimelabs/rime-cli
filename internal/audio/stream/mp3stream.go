package stream

import (
	"errors"
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
)

type StreamingMP3Decoder struct {
	streamer beep.StreamSeekCloser
	format   beep.Format
}

func DecodeMP3Streaming(r io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	if r == nil {
		return nil, beep.Format{}, errors.New("reader cannot be nil")
	}
	streamer, format, err := mp3.Decode(r)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return streamer, format, nil
}
