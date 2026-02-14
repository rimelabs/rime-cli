package analyze

import (
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestScaleAmplitudes_Basic(t *testing.T) {
	amps := []float64{0.5, 0.8, 1.0}
	scaled := ScaleAmplitudes(amps, 1.0, 0.0)

	if len(scaled) != len(amps) {
		t.Errorf("ScaleAmplitudes length = %d, expected %d", len(scaled), len(amps))
	}
	if scaled[0] != 0.5 || scaled[1] != 0.8 || scaled[2] != 1.0 {
		t.Errorf("ScaleAmplitudes with scale=1.0 should not change values")
	}
}

func TestScaleAmplitudes_ClampMax(t *testing.T) {
	amps := []float64{0.5, 1.0, 2.0}
	scaled := ScaleAmplitudes(amps, 2.0, 0.0)

	if scaled[0] != 1.0 {
		t.Errorf("ScaleAmplitudes[0] = %f, expected 1.0", scaled[0])
	}
	if scaled[1] != 1.0 {
		t.Errorf("ScaleAmplitudes[1] = %f, expected 1.0", scaled[1])
	}
	if scaled[2] != 1.0 {
		t.Errorf("ScaleAmplitudes[2] = %f, expected 1.0", scaled[2])
	}
}

func TestScaleAmplitudes_MinThreshold(t *testing.T) {
	amps := []float64{0.02, 0.03, 0.0}
	scaled := ScaleAmplitudes(amps, 0.5, 0.1)

	if scaled[0] < 0.1 {
		t.Errorf("ScaleAmplitudes[0] = %f, expected >= 0.1 (amp > 0.01 and scaled < threshold)", scaled[0])
	}
	if scaled[1] < 0.1 {
		t.Errorf("ScaleAmplitudes[1] = %f, expected >= 0.1 (amp > 0.01 and scaled < threshold)", scaled[1])
	}
	if scaled[2] != 0.0 {
		t.Errorf("ScaleAmplitudes[2] = %f, expected 0.0 (zero should not get threshold)", scaled[2])
	}
}

func TestAnalyzeAmplitudes_ValidWAV(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	amplitudes, err := AnalyzeAmplitudes(wavData, 4)
	if err != nil {
		t.Fatalf("AnalyzeAmplitudes() error = %v", err)
	}

	if len(amplitudes) == 0 {
		t.Error("AnalyzeAmplitudes() returned empty array")
	}

	if len(amplitudes) != 4 {
		t.Errorf("AnalyzeAmplitudes() returned %d samples, expected 4", len(amplitudes))
	}
}

func TestAnalyzeAmplitudes_InvalidData(t *testing.T) {
	invalidData := []byte("not a wav file")

	_, err := AnalyzeAmplitudes(invalidData, 4)
	if err == nil {
		t.Error("AnalyzeAmplitudes() with invalid data should return error")
	}
}
