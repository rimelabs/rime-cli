package metadata

import (
	"bytes"
	"encoding/binary"
)

// WAV file structure (relevant parts):
//
//	Offset  Size  Description
//	------  ----  -----------
//	0       4     "RIFF" magic bytes
//	4       4     File size minus 8 (little-endian uint32)
//	8       4     "WAVE" format identifier
//	12      4     Chunk ID (e.g., "fmt ", "LIST", "data")
//	16      4     Chunk size (little-endian uint32)
//	20      ...   Chunk data
//	...     ...   More chunks follow until "data" chunk
//
// The "data" chunk contains the actual audio samples.
// Its header is 8 bytes: "data" + 4-byte size.
//
// Streaming responses may have placeholder sizes (0 or 0xFFFFFFFF)
// because the total size isn't known until streaming completes.
// FixWavHeader fixes those sizes based on actual file length.
//
// LIST chunk structure:
//
//	Offset  Size  Description
//	------  ----  -----------
//	0       4     "LIST" chunk ID
//	4       4     Chunk size (little-endian uint32)
//	8       4     List type (e.g., "INFO")
//	12      ...   Sub-chunks (variable length)
//
// INFO sub-chunk structure:
//
//	Offset  Size  Description
//	------  ----  -----------
//	0       4     Sub-chunk ID (e.g., "IART", "INAM", "ICMT")
//	4       4     Sub-chunk size (little-endian uint32)
//	8       ...   Text data (null-padded to even length)
//
// Supported INFO sub-chunks:
//	- IART: Artist name
//	- INAM: Track/name title
//	- ICMT: Comment/description
//
// The LIST chunk must be placed before the "data" chunk in the WAV file.
// Sub-chunks are padded to even byte boundaries.

type WavMetadata struct {
	Artist  string // IART: "Rime AI TTS"
	Name    string // INAM: "celeste (arcana) eng"
	Comment string // ICMT: "[celeste-arcana-eng]: The quick brown fox..."
}

func EmbedMetadata(data []byte, meta WavMetadata) []byte {
	// Minimum WAV file size is 44 bytes (RIFF header + fmt chunk + data chunk header)
	if len(data) < 44 {
		return data
	}
	// Verify RIFF magic bytes at offset 0 and WAVE identifier at offset 8
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return data
	}

	var listChunk bytes.Buffer

	writeInfoChunk := func(id string, value string) {
		if value == "" {
			return
		}
		listChunk.WriteString(id) // 4 bytes: sub-chunk ID (e.g., "IART", "INAM", "ICMT")
		padded := value
		// WAV chunks must be aligned to even byte boundaries
		if len(padded)%2 != 0 {
			padded += "\x00" // Add null padding byte if odd length
		}
		binary.Write(&listChunk, binary.LittleEndian, uint32(len(value))) // 4 bytes: size (actual value length, not padded)
		listChunk.WriteString(padded)                                     // Variable: text data (padded to even length)
	}

	writeInfoChunk("IART", meta.Artist)
	writeInfoChunk("INAM", meta.Name)
	writeInfoChunk("ICMT", meta.Comment)

	if listChunk.Len() == 0 {
		return data
	}

	infoData := listChunk.Bytes()
	listSize := uint32(4 + len(infoData)) // Size includes "INFO" (4 bytes) + sub-chunks

	var list bytes.Buffer
	list.WriteString("LIST")                           // 4 bytes: chunk ID
	binary.Write(&list, binary.LittleEndian, listSize) // 4 bytes: chunk size (little-endian)
	list.WriteString("INFO")                           // 4 bytes: list type
	list.Write(infoData)                               // Variable: INFO sub-chunks

	dataPos := findDataChunkPos(data)
	if dataPos < 0 {
		return data
	}

	// Insert LIST chunk before data chunk
	result := make([]byte, 0, len(data)+list.Len())
	result = append(result, data[:dataPos]...) // Everything before data chunk
	result = append(result, list.Bytes()...)   // Insert LIST/INFO chunk
	result = append(result, data[dataPos:]...) // Data chunk and everything after

	// Update RIFF size in header (bytes 4-7): file size minus 8 bytes
	newRiffSize := uint32(len(result) - 8)
	binary.LittleEndian.PutUint32(result[4:8], newRiffSize)

	return result
}

// ReadMetadata reads metadata from a WAV file's LIST/INFO chunk.
// Returns empty metadata if the file doesn't contain a LIST/INFO chunk
// or if it's not a valid WAV file.
func ReadMetadata(data []byte) WavMetadata {
	var meta WavMetadata
	if len(data) < 44 {
		return meta
	}
	// Verify RIFF magic bytes at offset 0 and WAVE identifier at offset 8
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return meta
	}

	// Start parsing chunks after RIFF header (offset 12)
	pos := 12
	for pos+8 <= len(data) {
		// Read chunk ID (4 bytes at offset pos)
		chunkID := string(data[pos : pos+4])
		// Read chunk size (4 bytes, little-endian, at offset pos+4)
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		// Check for LIST chunk with INFO type
		if chunkID == "LIST" && pos+12 <= len(data) {
			// Read list type (4 bytes at offset pos+8)
			listType := string(data[pos+8 : pos+12])
			if listType == "INFO" {
				// Extract INFO sub-chunks (starts at pos+12, length is chunkSize-4)
				meta = parseInfoChunk(data[pos+12 : pos+8+chunkSize])
			}
		}

		// Advance to next chunk: skip header (8 bytes) + chunk data
		pos += 8 + chunkSize
		// WAV chunks are aligned to even byte boundaries
		if chunkSize%2 != 0 {
			pos++ // Skip padding byte if chunk size was odd
		}
	}

	return meta
}

func findDataChunkPos(data []byte) int {
	// Start searching after RIFF header (offset 12)
	pos := 12
	for pos+8 <= len(data) {
		// Read chunk ID (4 bytes at offset pos)
		chunkID := string(data[pos : pos+4])
		// Read chunk size (4 bytes, little-endian, at offset pos+4)
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		if chunkID == "data" {
			return pos // Found data chunk, return its position
		}

		// Advance to next chunk: skip header (8 bytes) + chunk data
		pos += 8 + chunkSize
		// WAV chunks are aligned to even byte boundaries
		if chunkSize%2 != 0 {
			pos++ // Skip padding byte if chunk size was odd
		}
	}
	return -1
}

func parseInfoChunk(data []byte) WavMetadata {
	var meta WavMetadata
	pos := 0

	for pos+8 <= len(data) {
		// Read sub-chunk ID (4 bytes at offset pos)
		chunkID := string(data[pos : pos+4])
		// Read sub-chunk size (4 bytes, little-endian, at offset pos+4)
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		if pos+8+chunkSize > len(data) {
			break // Sub-chunk extends beyond available data
		}

		// Extract text value (starts at pos+8, length is chunkSize)
		// Trim null bytes from end (padding for even alignment)
		value := string(bytes.TrimRight(data[pos+8:pos+8+chunkSize], "\x00"))

		switch chunkID {
		case "IART":
			meta.Artist = value
		case "INAM":
			meta.Name = value
		case "ICMT":
			meta.Comment = value
		}

		// Advance to next sub-chunk: skip header (8 bytes) + chunk data
		pos += 8 + chunkSize
		// Sub-chunks are aligned to even byte boundaries
		if chunkSize%2 != 0 {
			pos++ // Skip padding byte if chunk size was odd
		}
	}

	return meta
}

// FixWavHeader corrects WAV header sizes for streaming responses.
// Returns the original data unchanged if it's not a valid WAV file.
func FixWavHeader(data []byte) []byte {
	// Minimum WAV file size is 44 bytes (RIFF header + fmt chunk + data chunk header)
	if len(data) < 44 {
		return data
	}

	// Verify RIFF magic bytes at offset 0 and WAVE identifier at offset 8
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	fileSize := uint32(len(result))
	correctRiffSize := fileSize - 8 // RIFF size is file size minus 8 bytes

	// Start parsing chunks after RIFF header (offset 12)
	pos := 12
	for pos+8 <= len(result) {
		// Read chunk ID (4 bytes at offset pos)
		chunkID := string(result[pos : pos+4])
		// Read chunk size (4 bytes, little-endian, at offset pos+4)
		chunkSize := binary.LittleEndian.Uint32(result[pos+4 : pos+8])

		if chunkID == "data" {
			// Calculate correct data chunk size: file size minus data chunk header position
			correctDataSize := fileSize - uint32(pos+8)

			if chunkSize != correctDataSize {
				// Update data chunk size (bytes pos+4 to pos+7)
				binary.LittleEndian.PutUint32(result[pos+4:pos+8], correctDataSize)
				// Update RIFF size in header (bytes 4-7)
				binary.LittleEndian.PutUint32(result[4:8], correctRiffSize)
			}
			return result
		}

		// Advance to next chunk: skip header (8 bytes) + chunk data
		pos += 8 + int(chunkSize)
	}

	return result
}
