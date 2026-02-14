package detectformat

import "testing"

func TestDetectFormat_WAV(t *testing.T) {
	wav := []byte("RIFF")
	wav = append(wav, make([]byte, 4)...)
	wav = append(wav, []byte("WAVE")...)

	format := DetectFormat(wav)
	if format != "audio/wav" {
		t.Errorf("Expected audio/wav, got %s", format)
	}
}

func TestDetectFormat_MP3_SyncWord(t *testing.T) {
	mp3 := []byte{0xFF, 0xFB, 0x90, 0x00}

	format := DetectFormat(mp3)
	if format != "audio/mp3" {
		t.Errorf("Expected audio/mp3, got %s", format)
	}
}

func TestDetectFormat_MP3_ID3(t *testing.T) {
	mp3 := []byte("ID3")
	mp3 = append(mp3, make([]byte, 7)...)

	format := DetectFormat(mp3)
	if format != "audio/mp3" {
		t.Errorf("Expected audio/mp3, got %s", format)
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	unknown := []byte("unknown format")

	format := DetectFormat(unknown)
	if format != "" {
		t.Errorf("Expected empty string, got %s", format)
	}
}

func TestDetectFormat_Empty(t *testing.T) {
	format := DetectFormat([]byte{})
	if format != "" {
		t.Errorf("Expected empty string, got %s", format)
	}
}
