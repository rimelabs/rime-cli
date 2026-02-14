package metadata

import (
	"bytes"
	"encoding/binary"
)

// MP3 file structure with ID3v2.3 tags (relevant parts):
//
//	Offset  Size  Description
//	------  ----  -----------
//	0       3     "ID3" magic bytes
//	3       1     Version (0x03 for ID3v2.3)
//	4       1     Revision (0x00)
//	5       1     Flags (0x00)
//	6       4     Tag size (synchsafe uint32, big-endian)
//	10      ...   Frames (variable length)
//	...     ...   MP3 audio data starts after tag
//
// Frame structure:
//
//	Offset  Size  Description
//	------  ----  -----------
//	0       4     Frame ID (e.g., "TPE1", "TIT2", "COMM")
//	4       4     Frame size (uint32, big-endian)
//	8       2     Flags (uint16, big-endian)
//	10      1     Text encoding (0x00 = ISO-8859-1, 0x01 = UTF-16)
//	11      ...   Frame data (variable length)
//
// Text frames (TPE1, TIT2):
//	- Encoding byte (0x00)
//	- Text string (null-terminated)
//
// Comment frame (COMM):
//	- Encoding byte (0x00)
//	- Language (3 bytes, e.g., "eng")
//	- Description (null-terminated)
//	- Comment text (null-terminated)
//
// Synchsafe integers encode size values where the MSB of each byte is 0,
// allowing MP3 sync patterns (0xFF) to be detected in the tag area.

type MP3Metadata struct {
	Artist  string
	Title   string
	Comment string
}

func synchsafeEncode(size uint32) [4]byte {
	var result [4]byte
	// Extract 7 bits per byte, ensuring MSB is always 0
	// This prevents 0xFF sync patterns from appearing in tag data
	result[0] = byte((size >> 21) & 0x7F) // Bits 28-21 -> byte 0, bits 6-0
	result[1] = byte((size >> 14) & 0x7F) // Bits 20-14 -> byte 1, bits 6-0
	result[2] = byte((size >> 7) & 0x7F)  // Bits 13-7 -> byte 2, bits 6-0
	result[3] = byte(size & 0x7F)         // Bits 6-0 -> byte 3, bits 6-0
	return result
}

func synchsafeDecode(data [4]byte) uint32 {
	// Reconstruct uint32 from 4 synchsafe bytes
	// Each byte contains 7 bits, shifted back into position
	return uint32(data[0])<<21 | uint32(data[1])<<14 | uint32(data[2])<<7 | uint32(data[3])
}

// EmbedMP3Metadata embeds metadata into an MP3 file using ID3v2.3 tags.
// If the file already has an ID3 tag, it replaces it with the new metadata.
// If the file has no ID3 tag, it prepends one before the audio data.
// Returns the original data unchanged if embedding fails or metadata is empty.
func EmbedMP3Metadata(data []byte, meta MP3Metadata) ([]byte, error) {
	var frames bytes.Buffer

	writeTextFrame := func(id string, text string) {
		if text == "" {
			return
		}
		textBytes := []byte(text)
		frameSize := uint32(1 + len(textBytes))            // 1 byte encoding + text length
		frames.WriteString(id)                             // 4 bytes: frame ID (e.g., "TPE1", "TIT2")
		binary.Write(&frames, binary.BigEndian, frameSize) // 4 bytes: frame size
		binary.Write(&frames, binary.BigEndian, uint16(0)) // 2 bytes: flags (unused)
		frames.WriteByte(0)                                // 1 byte: encoding (0x00 = ISO-8859-1)
		frames.Write(textBytes)                            // Variable: text data
	}

	writeCommentFrame := func(text string) {
		if text == "" {
			return
		}
		textBytes := []byte(text)
		frameSize := uint32(1 + 3 + 1 + len(textBytes))    // 1 byte encoding + 3 byte lang + 1 byte desc terminator + text
		frames.WriteString("COMM")                         // 4 bytes: frame ID
		binary.Write(&frames, binary.BigEndian, frameSize) // 4 bytes: frame size
		binary.Write(&frames, binary.BigEndian, uint16(0)) // 2 bytes: flags (unused)
		frames.WriteByte(0)                                // 1 byte: encoding (0x00 = ISO-8859-1)
		frames.WriteString("eng")                          // 3 bytes: language code
		frames.WriteByte(0)                                // 1 byte: description terminator (empty description)
		frames.Write(textBytes)                            // Variable: comment text
	}

	writeTextFrame("TPE1", meta.Artist)
	writeTextFrame("TIT2", meta.Title)
	writeCommentFrame(meta.Comment)

	if frames.Len() == 0 {
		return data, nil
	}

	tagSize := uint32(frames.Len())
	var header bytes.Buffer
	header.WriteString("ID3")             // Bytes 0-2: magic identifier
	header.WriteByte(0x03)                // Byte 3: version (ID3v2.3)
	header.WriteByte(0x00)                // Byte 4: revision
	header.WriteByte(0x00)                // Byte 5: flags
	sizeBytes := synchsafeEncode(tagSize) // Bytes 6-9: tag size (synchsafe uint32)
	header.Write(sizeBytes[:])

	audioStart := findMP3AudioStart(data)
	result := make([]byte, 0, len(data)+header.Len()+frames.Len())
	result = append(result, header.Bytes()...)
	result = append(result, frames.Bytes()...)
	result = append(result, data[audioStart:]...)

	return result, nil
}

// ReadMP3Metadata reads ID3v2.3 metadata from an MP3 file.
// Returns empty metadata if the file doesn't contain valid ID3v2.3 tags
// or if the required frames are not present.
func ReadMP3Metadata(data []byte) MP3Metadata {
	var meta MP3Metadata
	if len(data) < 10 {
		return meta
	}

	// Check for ID3v2 magic bytes at offset 0
	if !bytes.Equal(data[0:3], []byte("ID3")) {
		return meta
	}

	// Verify ID3v2.3 version byte at offset 3
	if data[3] != 0x03 {
		return meta
	}

	// Extract synchsafe tag size from bytes 6-9
	tagSizeBytes := [4]byte{data[6], data[7], data[8], data[9]}
	tagSize := synchsafeDecode(tagSizeBytes)
	if len(data) < 10+int(tagSize) {
		return meta
	}

	// Frames start at offset 10 (after 10-byte header)
	pos := 10
	endPos := 10 + int(tagSize)

	for pos+10 <= endPos {
		// Read frame ID (4 bytes at offset pos)
		frameID := string(data[pos : pos+4])
		if frameID[0] == 0 {
			break // Null byte indicates end of frames
		}

		// Read frame size (4 bytes, big-endian, at offset pos+4)
		frameSize := binary.BigEndian.Uint32(data[pos+4 : pos+8])
		// Read flags (2 bytes, big-endian, at offset pos+8)
		flags := binary.BigEndian.Uint16(data[pos+8 : pos+10])

		if pos+10+int(frameSize) > endPos {
			break // Frame extends beyond tag boundary
		}

		// Read encoding byte (1 byte at offset pos+10)
		encoding := data[pos+10]
		// Extract frame data (starts at pos+11, length is frameSize)
		frameData := data[pos+11 : pos+10+int(frameSize)]

		switch frameID {
		case "TPE1":
			// TPE1: Text frame with artist name
			// Frame data is encoding byte (already read) + null-terminated text
			if encoding == 0 && len(frameData) > 0 {
				meta.Artist = string(bytes.TrimRight(frameData, "\x00"))
			}
		case "TIT2":
			// TIT2: Text frame with title
			// Frame data is encoding byte (already read) + null-terminated text
			if encoding == 0 && len(frameData) > 0 {
				meta.Title = string(bytes.TrimRight(frameData, "\x00"))
			}
		case "COMM":
			// COMM: Comment frame structure:
			// - Encoding byte (already read at pos+10)
			// - Language (3 bytes at frameData[0:3])
			// - Description (null-terminated, starts at frameData[3])
			// - Comment text (null-terminated, starts after description)
			if encoding == 0 && len(frameData) > 4 {
				lang := string(frameData[0:3]) // Extract 3-byte language code
				_ = lang
				descEnd := bytes.IndexByte(frameData[3:], 0) // Find description terminator
				if descEnd >= 0 {
					textStart := 4 + descEnd // Comment text starts after lang(3) + desc terminator(1)
					if textStart < len(frameData) {
						meta.Comment = string(bytes.TrimRight(frameData[textStart:], "\x00"))
					}
				} else if len(frameData) > 3 {
					// No description terminator, comment starts immediately after language
					meta.Comment = string(bytes.TrimRight(frameData[3:], "\x00"))
				}
			}
		}

		// Check if frame has unsynchronization flag (bit 6 of flags)
		if flags&0x40 != 0 {
			break
		}

		// Advance to next frame: skip header (10 bytes) + frame data
		pos += 10 + int(frameSize)
	}

	return meta
}

func findMP3AudioStart(data []byte) int {
	if len(data) < 10 {
		return 0
	}

	// Check for ID3v2 tag
	if !bytes.Equal(data[0:3], []byte("ID3")) {
		return 0
	}

	// Read tag size from bytes 6-9
	tagSizeBytes := [4]byte{data[6], data[7], data[8], data[9]}
	tagSize := synchsafeDecode(tagSizeBytes)
	audioStart := 10 + int(tagSize) // Audio starts after 10-byte header + tag size

	// Verify MP3 sync pattern: 0xFF followed by 0xE0-0xFF (bits 111xxxxx)
	// This confirms we've found the actual audio data
	if len(data) >= audioStart && data[audioStart] == 0xFF && (data[audioStart+1]&0xE0) == 0xE0 {
		return audioStart
	}

	return 0
}
