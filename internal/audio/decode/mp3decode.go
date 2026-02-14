package decode

import (
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
)

func decodeMP3(r io.Reader) (AudioDecoder, beep.Format, error) {
	rc, ok := r.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(r)
	}
	streamer, format, err := stream.DecodeMP3Streaming(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return &mp3DecoderAdapter{streamer: streamer}, format, nil
}

type mp3DecoderAdapter struct {
	streamer beep.StreamSeekCloser
}

func (m *mp3DecoderAdapter) Stream(samples [][2]float64) (n int, ok bool) {
	return m.streamer.Stream(samples)
}

func (m *mp3DecoderAdapter) Err() error {
	return m.streamer.Err()
}
