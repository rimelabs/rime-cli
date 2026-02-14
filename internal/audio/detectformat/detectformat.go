package detectformat

import "bytes"

func DetectFormat(data []byte) string {
	if len(data) >= 4 && bytes.Equal(data[0:4], []byte("RIFF")) {
		return "audio/wav"
	}
	if len(data) >= 3 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mp3"
	}
	if len(data) >= 3 && bytes.Equal(data[0:3], []byte("ID3")) {
		return "audio/mp3"
	}
	return ""
}
