package metadata

import (
	"encoding/binary"
	"testing"

	"bytes"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestEmbedAndReadMetadata(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()

	meta := WavMetadata{
		Artist:  "Rime AI TTS",
		Name:    "celeste (arcana) eng",
		Comment: "[celeste-arcana-eng]: Hello world",
	}

	embedded := EmbedMetadata(wav, meta)

	if len(embedded) <= len(wav) {
		t.Error("embedded should be larger than original")
	}

	read := ReadMetadata(embedded)

	if read.Artist != meta.Artist {
		t.Errorf("Artist: got %q, want %q", read.Artist, meta.Artist)
	}
	if read.Name != meta.Name {
		t.Errorf("Name: got %q, want %q", read.Name, meta.Name)
	}
	if read.Comment != meta.Comment {
		t.Errorf("Comment: got %q, want %q", read.Comment, meta.Comment)
	}
}

func TestReadMetadataEmpty(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	meta := ReadMetadata(wav)

	if meta.Artist != "" || meta.Name != "" || meta.Comment != "" {
		t.Error("metadata should be empty for wav without LIST chunk")
	}
}

func TestEmbedMetadataInvalidWav(t *testing.T) {
	notWav := []byte("not a wav file")
	result := EmbedMetadata(notWav, WavMetadata{Artist: "test"})

	if !bytes.Equal(result, notWav) {
		t.Error("should return original data for non-wav")
	}
}

func makeWav(dataSize uint32, riffSize uint32, audioBytes int) []byte {
	buf := make([]byte, 44+audioBytes)

	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], riffSize)
	copy(buf[8:12], "WAVE")

	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)

	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], dataSize)

	return buf
}

func TestFixWavHeader_AlreadyCorrect(t *testing.T) {
	wav := makeWav(100, 136, 100)
	fixed := FixWavHeader(wav)

	riffSize := binary.LittleEndian.Uint32(fixed[4:8])
	dataSize := binary.LittleEndian.Uint32(fixed[40:44])

	if riffSize != 136 || dataSize != 100 {
		t.Errorf("sizes changed: riff=%d data=%d", riffSize, dataSize)
	}
}

func TestFixWavHeader_ZeroDataSize(t *testing.T) {
	wav := makeWav(0, 0, 100)
	fixed := FixWavHeader(wav)

	riffSize := binary.LittleEndian.Uint32(fixed[4:8])
	dataSize := binary.LittleEndian.Uint32(fixed[40:44])

	if dataSize != 100 {
		t.Errorf("expected data size 100, got %d", dataSize)
	}
	if riffSize != 136 {
		t.Errorf("expected riff size 136, got %d", riffSize)
	}
}

func TestFixWavHeader_NotWav(t *testing.T) {
	data := []byte("not a wav file at all")
	fixed := FixWavHeader(data)

	if string(fixed) != string(data) {
		t.Error("non-WAV data should be returned unchanged")
	}
}

func TestFixWavHeader_TooShort(t *testing.T) {
	data := []byte("RIFF")
	fixed := FixWavHeader(data)

	if string(fixed) != string(data) {
		t.Error("short data should be returned unchanged")
	}
}
