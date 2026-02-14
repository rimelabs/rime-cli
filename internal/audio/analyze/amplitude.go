package analyze

import (
	"bytes"
	"io"
	"math"

	"github.com/rimelabs/rime-cli/internal/audio/decode"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
)

func AnalyzeAmplitudesFromReader(r io.Reader, contentType string, samplesPerSecond int) ([]float64, error) {
	decoder, format, err := decode.DecodeAudio(r, contentType)
	if err != nil {
		return nil, err
	}

	samplesPerChunk := int(format.SampleRate) / samplesPerSecond
	if samplesPerChunk < 1 {
		samplesPerChunk = 1
	}

	var amplitudes []float64
	buf := make([][2]float64, samplesPerChunk)

	for {
		n, ok := decoder.Stream(buf)
		if n == 0 {
			break
		}

		var sum float64
		for i := 0; i < n; i++ {
			sum += buf[i][0]*buf[i][0] + buf[i][1]*buf[i][1]
		}
		rms := math.Sqrt(sum / float64(n*2))
		amplitudes = append(amplitudes, rms)

		if !ok {
			break
		}
	}

	if closer, ok := decoder.(io.Closer); ok {
		closer.Close()
	}

	return amplitudes, nil
}

func AnalyzeAmplitudes(data []byte, samplesPerSecond int) ([]float64, error) {
	contentType := detectformat.DetectFormat(data)
	if contentType == "" {
		contentType = "audio/wav"
	}
	return AnalyzeAmplitudesFromReader(bytes.NewReader(data), contentType, samplesPerSecond)
}

func ScaleAmplitudes(amplitudes []float64, scale float64, minThreshold float64) []float64 {
	result := make([]float64, len(amplitudes))
	for i, amp := range amplitudes {
		scaled := amp * scale
		if amp > 0.01 && scaled < minThreshold {
			scaled = minThreshold
		}
		if scaled > 1.0 {
			scaled = 1.0
		}
		result[i] = scaled
	}
	return result
}
