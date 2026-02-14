package analyze

import (
	"math"
	"testing"
)

type mockStreamer struct {
	samples [][2]float64
	pos     int
}

func (m *mockStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if m.pos >= len(m.samples) {
		return 0, false
	}
	n = copy(samples, m.samples[m.pos:])
	m.pos += n
	ok = m.pos < len(m.samples)
	return n, ok
}

func (m *mockStreamer) Err() error {
	return nil
}

func TestNewAmplitudeAnalyzer(t *testing.T) {
	mock := &mockStreamer{
		samples: [][2]float64{{0.5, 0.5}, {0.8, 0.8}},
	}
	analyzer := NewAmplitudeAnalyzer(mock)

	if analyzer == nil {
		t.Fatal("NewAmplitudeAnalyzer returned nil")
	}
	if analyzer.Amplitude() != 0.0 {
		t.Errorf("Initial amplitude should be 0.0, got %f", analyzer.Amplitude())
	}
}

func TestAmplitudeAnalyzer_Stream(t *testing.T) {
	mock := &mockStreamer{
		samples: [][2]float64{
			{1.0, 1.0},
			{0.0, 0.0},
			{0.5, 0.5},
		},
	}
	analyzer := NewAmplitudeAnalyzer(mock)

	buf := make([][2]float64, 2)
	n, ok := analyzer.Stream(buf)

	if n != 2 {
		t.Errorf("Stream() returned %d samples, expected 2", n)
	}
	if !ok {
		t.Error("Stream() should return ok=true when more samples available")
	}

	amp := analyzer.Amplitude()
	expectedRMS := math.Sqrt((1.0*1.0 + 1.0*1.0 + 0.0*0.0 + 0.0*0.0) / 4.0)
	if math.Abs(amp-expectedRMS) > 0.001 {
		t.Errorf("Amplitude() = %f, expected approximately %f", amp, expectedRMS)
	}
}

func TestAmplitudeAnalyzer_EmptyStream(t *testing.T) {
	mock := &mockStreamer{
		samples: [][2]float64{},
	}
	analyzer := NewAmplitudeAnalyzer(mock)

	buf := make([][2]float64, 10)
	n, ok := analyzer.Stream(buf)

	if n != 0 {
		t.Errorf("Stream() with empty source returned %d samples, expected 0", n)
	}
	if ok {
		t.Error("Stream() with empty source should return ok=false")
	}
}

func TestAmplitudeAnalyzer_Amplitude(t *testing.T) {
	mock := &mockStreamer{
		samples: [][2]float64{{0.8, 0.6}},
	}
	analyzer := NewAmplitudeAnalyzer(mock)

	buf := make([][2]float64, 1)
	analyzer.Stream(buf)

	amp := analyzer.Amplitude()
	if amp <= 0 {
		t.Error("Amplitude() should return positive value after streaming")
	}
}
