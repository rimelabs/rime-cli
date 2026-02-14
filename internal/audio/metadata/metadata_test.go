package metadata

import (
	"testing"

	"github.com/rimelabs/rime-cli/internal/audio/testhelpers"
)

func TestGetParsedCommentFromFile_WAV(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	meta := WavMetadata{
		Comment: "[celeste-arcana-eng]: Hello world",
	}
	wavWithMeta := EmbedMetadata(wav, meta)

	parsed, ok := GetParsedCommentFromFile(wavWithMeta)
	if !ok {
		t.Error("Expected parsed comment")
	}
	if parsed.Speaker != "celeste" {
		t.Errorf("Expected speaker celeste, got %s", parsed.Speaker)
	}
	if parsed.ModelID != "arcana" {
		t.Errorf("Expected modelID arcana, got %s", parsed.ModelID)
	}
	if parsed.Text != "Hello world" {
		t.Errorf("Expected text 'Hello world', got %s", parsed.Text)
	}
}

func TestGetParsedCommentFromFile_MP3(t *testing.T) {
	mp3 := testhelpers.MakeMinimalMP3()
	meta := MP3Metadata{
		Comment: "[celeste-arcana-eng]: Hello world",
	}
	mp3WithMeta, err := EmbedMP3Metadata(mp3, meta)
	if err != nil {
		t.Fatalf("EmbedMP3Metadata failed: %v", err)
	}

	parsed, ok := GetParsedCommentFromFile(mp3WithMeta)
	if !ok {
		t.Error("Expected parsed comment")
	}
	if parsed.Speaker != "celeste" {
		t.Errorf("Expected speaker celeste, got %s", parsed.Speaker)
	}
	if parsed.ModelID != "arcana" {
		t.Errorf("Expected modelID arcana, got %s", parsed.ModelID)
	}
	if parsed.Text != "Hello world" {
		t.Errorf("Expected text 'Hello world', got %s", parsed.Text)
	}
}

func TestGetParsedCommentFromFile_NoMetadata(t *testing.T) {
	wav := testhelpers.MakeMinimalWAV()
	parsed, ok := GetParsedCommentFromFile(wav)
	if ok {
		t.Error("Expected no parsed comment for file without metadata")
	}
	if parsed != nil {
		t.Error("Expected nil parsed comment")
	}
}

func TestParseComment_Valid(t *testing.T) {
	comment := "[celeste-arcana-eng]: Hello world"
	parsed, ok := ParseComment(comment)

	if !ok {
		t.Error("ParseComment should return ok=true for valid comment")
	}
	if parsed == nil {
		t.Fatal("ParseComment returned nil")
	}
	if parsed.Speaker != "celeste" {
		t.Errorf("Speaker = %q, expected %q", parsed.Speaker, "celeste")
	}
	if parsed.ModelID != "arcana" {
		t.Errorf("ModelID = %q, expected %q", parsed.ModelID, "arcana")
	}
	if parsed.Language != "eng" {
		t.Errorf("Language = %q, expected %q", parsed.Language, "eng")
	}
	if parsed.Text != "Hello world" {
		t.Errorf("Text = %q, expected %q", parsed.Text, "Hello world")
	}
}

func TestParseComment_Invalid(t *testing.T) {
	invalidComments := []string{
		"not a valid format",
		"[celeste-arcana]: missing language",
		"[celeste]: missing model and language",
		"no brackets",
	}

	for _, comment := range invalidComments {
		parsed, ok := ParseComment(comment)
		if ok {
			t.Errorf("ParseComment(%q) should return ok=false", comment)
		}
		if parsed != nil {
			t.Errorf("ParseComment(%q) should return nil", comment)
		}
	}
}

func TestParseComment_WithSpaces(t *testing.T) {
	comment := "[astra-arcana-eng]:   Multiple   words   here"
	parsed, ok := ParseComment(comment)

	if !ok {
		t.Error("ParseComment should handle spaces")
	}
	if parsed.Text != "Multiple   words   here" {
		t.Errorf("Text should preserve spaces (leading whitespace consumed by regex), got %q", parsed.Text)
	}
}
