package analyze

import (
	"testing"
	"time"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestCalculateDuration_ValidWAV(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	duration := CalculateDuration(wavData, 24000, 1, 16)
	expected := time.Second

	if duration != expected {
		t.Errorf("CalculateDuration() = %v, expected %v", duration, expected)
	}
}

func TestCalculateDuration_InvalidParams(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(100)

	if CalculateDuration(wavData, 0, 1, 16) != 0 {
		t.Error("CalculateDuration with sampleRate=0 should return 0")
	}
	if CalculateDuration(wavData, 24000, 0, 16) != 0 {
		t.Error("CalculateDuration with numChannels=0 should return 0")
	}
	if CalculateDuration(wavData, 24000, 1, 0) != 0 {
		t.Error("CalculateDuration with bitsPerSample=0 should return 0")
	}
	if CalculateDuration([]byte{}, 24000, 1, 16) != 0 {
		t.Error("CalculateDuration with data < 44 bytes should return 0")
	}
}

func TestCalculateDuration_Stereo(t *testing.T) {
	wavData := testhelpers.MakeValidWAV(24000)

	duration := CalculateDuration(wavData, 24000, 2, 16)
	expected := 500 * time.Millisecond

	if duration != expected {
		t.Errorf("CalculateDuration(stereo) = %v, expected %v", duration, expected)
	}
}
