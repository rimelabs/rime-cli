package metadata

import (
	"regexp"

	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
)

type ParsedComment struct {
	Speaker  string
	ModelID  string
	Language string
	Text     string
}

var commentFormatRegex = regexp.MustCompile(`^\[([^-]+)-([^-]+)-([^\]]+)\]:\s*(.+)$`)

func ParseComment(comment string) (*ParsedComment, bool) {
	matches := commentFormatRegex.FindStringSubmatch(comment)
	if len(matches) != 5 {
		return nil, false
	}
	return &ParsedComment{
		Speaker:  matches[1],
		ModelID:  matches[2],
		Language: matches[3],
		Text:     matches[4],
	}, true
}

func GetParsedCommentFromFile(data []byte) (*ParsedComment, bool) {
	contentType := detectformat.DetectFormat(data)

	var comment string
	switch contentType {
	case "audio/mpeg", "audio/mp3":
		meta := ReadMP3Metadata(data)
		comment = meta.Comment
	default:
		meta := ReadMetadata(data)
		comment = meta.Comment
	}

	if comment == "" {
		return nil, false
	}
	return ParseComment(comment)
}
