package decode

import (
	"fmt"
	"io"

	"github.com/gopxl/beep/v2"
)

type AudioDecoder interface {
	Stream(samples [][2]float64) (n int, ok bool)
	Err() error
}

func DecodeAudio(r io.Reader, contentType string) (AudioDecoder, beep.Format, error) {
	switch contentType {
	case "audio/wav":
		return decodeWAV(r)
	case "audio/mpeg", "audio/mp3":
		return decodeMP3(r)
	default:
		return nil, beep.Format{}, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
