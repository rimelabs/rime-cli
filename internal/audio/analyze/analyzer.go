package analyze

import (
	"math"
	"sync/atomic"

	"github.com/gopxl/beep/v2"
)

// AmplitudeAnalyzer wraps a streamer and computes RMS amplitude.
type AmplitudeAnalyzer struct {
	source    beep.Streamer
	amplitude atomic.Value
}

// NewAmplitudeAnalyzer wraps a streamer to analyze amplitude.
func NewAmplitudeAnalyzer(source beep.Streamer) *AmplitudeAnalyzer {
	a := &AmplitudeAnalyzer{source: source}
	a.amplitude.Store(float64(0))
	return a
}

// Stream implements beep.Streamer, computing RMS amplitude.
func (a *AmplitudeAnalyzer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = a.source.Stream(samples)
	if n > 0 {
		var sum float64
		for i := 0; i < n; i++ {
			sum += samples[i][0]*samples[i][0] + samples[i][1]*samples[i][1]
		}
		rms := math.Sqrt(sum / float64(n*2))
		a.amplitude.Store(rms)
	}
	return n, ok
}

func (a *AmplitudeAnalyzer) Err() error {
	return a.source.Err()
}

// Amplitude returns the current RMS amplitude (0.0-1.0 typically).
func (a *AmplitudeAnalyzer) Amplitude() float64 {
	return a.amplitude.Load().(float64)
}
