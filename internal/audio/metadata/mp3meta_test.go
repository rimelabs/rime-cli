package metadata

import (
	"bytes"
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestEmbedAndReadMP3Metadata(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()

	meta := MP3Metadata{
		Artist:  "Rime AI TTS",
		Title:   "Test Title",
		Comment: "[celeste-arcana-eng]: Hello world",
	}

	embedded, err := EmbedMP3Metadata(mp3, meta)
	if err != nil {
		t.Fatalf("EmbedMP3Metadata failed: %v", err)
	}

	if len(embedded) <= len(mp3) {
		t.Error("embedded should be larger than original")
	}

	read := ReadMP3Metadata(embedded)

	if read.Artist != meta.Artist {
		t.Errorf("Artist: got %q, want %q", read.Artist, meta.Artist)
	}
	if read.Title != meta.Title {
		t.Errorf("Title: got %q, want %q", read.Title, meta.Title)
	}
	if read.Comment != meta.Comment {
		t.Errorf("Comment: got %q, want %q", read.Comment, meta.Comment)
	}
}

func TestReadMP3Metadata_NoTags(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()
	meta := ReadMP3Metadata(mp3)

	if meta.Artist != "" || meta.Title != "" || meta.Comment != "" {
		t.Error("metadata should be empty for mp3 without ID3 tags")
	}
}

func TestEmbedMP3Metadata_Empty(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()
	meta := MP3Metadata{}

	embedded, err := EmbedMP3Metadata(mp3, meta)
	if err != nil {
		t.Fatalf("EmbedMP3Metadata failed: %v", err)
	}

	if len(embedded) != len(mp3) {
		t.Error("empty metadata should not change file size")
	}
}

func TestFindMP3AudioStart_WithID3(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("ID3")
	buf.WriteByte(0x03)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x64})
	buf.Write(make([]byte, 100))
	buf.Write([]byte{0xFF, 0xFB, 0x90, 0x00})

	data := buf.Bytes()
	start := findMP3AudioStart(data)

	if start == 0 {
		t.Error("Expected non-zero audio start position")
	}
}

func TestFindMP3AudioStart_NoID3(t *testing.T) {
	mp3 := []byte{0xFF, 0xFB, 0x90, 0x00}
	start := findMP3AudioStart(mp3)

	if start != 0 {
		t.Errorf("Expected 0 for MP3 without ID3, got %d", start)
	}
}
