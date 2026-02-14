package decode

import (
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
)

func decodeWAV(r io.Reader) (AudioDecoder, beep.Format, error) {
	rc, ok := r.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(r)
	}
	decoder, format, err := stream.DecodeStreaming(r)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return &wavDecoderAdapter{decoder: decoder, rc: rc}, format, nil
}

type wavDecoderAdapter struct {
	decoder *stream.StreamingDecoder
	rc      io.ReadCloser
}

func (w *wavDecoderAdapter) Stream(samples [][2]float64) (n int, ok bool) {
	return w.decoder.Stream(samples)
}

func (w *wavDecoderAdapter) Err() error {
	return w.decoder.Err()
}

func (w *wavDecoderAdapter) Close() error {
	if w.rc != nil {
		return w.rc.Close()
	}
	return nil
}
