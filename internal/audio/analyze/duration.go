package analyze

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/rimelabs/rime-cli/internal/audio/detectformat"
	"github.com/rimelabs/rime-cli/internal/audio/stream"
)

func CalculateDuration(data []byte, sampleRate int, numChannels int, bitsPerSample int) time.Duration {
	if sampleRate <= 0 || numChannels <= 0 || bitsPerSample <= 0 {
		return 0
	}
	if len(data) < 44 {
		return 0
	}

	contentType := detectformat.DetectFormat(data)
	if contentType == "audio/wav" || contentType == "" {
		duration := CalculateWavDurationWithParams(data, sampleRate, numChannels, bitsPerSample)
		if duration > 0 {
			return duration
		}
	}

	dataSize := len(data) - 44
	bytesPerFrame := numChannels * (bitsPerSample / 8)
	if bytesPerFrame <= 0 {
		return 0
	}
	frames := dataSize / bytesPerFrame
	return time.Duration(frames) * time.Second / time.Duration(sampleRate)
}

func CalculateWavDurationWithParams(data []byte, sampleRate int, numChannels int, bitsPerSample int) time.Duration {
	if len(data) < 12 {
		return 0
	}
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return 0
	}

	var dataChunkSize int
	foundData := false

	pos := 12
	for pos+8 <= len(data) && !foundData {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		switch chunkID {
		case "data":
			remainingSize := len(data) - (pos + 8)
			if chunkSize > remainingSize {
				dataChunkSize = remainingSize
			} else {
				dataChunkSize = chunkSize
			}
			foundData = true
		}

		pos += 8 + chunkSize
		if chunkSize%2 != 0 {
			pos++
		}
	}

	if !foundData || dataChunkSize <= 0 {
		return 0
	}

	bytesPerFrame := numChannels * (bitsPerSample / 8)
	if bytesPerFrame <= 0 {
		return 0
	}
	frames := dataChunkSize / bytesPerFrame
	return time.Duration(frames) * time.Second / time.Duration(sampleRate)
}

func CalculateWavDuration(data []byte) time.Duration {
	if len(data) < 12 {
		return 0
	}
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return 0
	}

	var fmtSampleRate uint32
	var fmtNumChannels uint16
	var fmtBitsPerSample uint16
	var dataChunkSize int
	foundFmt := false
	foundData := false

	pos := 12
	for pos+8 <= len(data) && (!foundFmt || !foundData) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 && pos+8+chunkSize <= len(data) {
				fmtData := data[pos+8 : pos+8+chunkSize]
				if len(fmtData) >= 16 {
					audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
					if audioFormat == 1 {
						fmtNumChannels = binary.LittleEndian.Uint16(fmtData[2:4])
						fmtSampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
						fmtBitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])
						foundFmt = true
					}
				}
			}
		case "data":
			remainingSize := len(data) - (pos + 8)
			if chunkSize > remainingSize {
				dataChunkSize = remainingSize
			} else {
				dataChunkSize = chunkSize
			}
			foundData = true
		}

		pos += 8 + chunkSize
		if chunkSize%2 != 0 {
			pos++
		}
	}

	if !foundFmt || !foundData {
		return 0
	}

	if fmtSampleRate <= 0 || fmtNumChannels <= 0 || fmtBitsPerSample <= 0 || dataChunkSize <= 0 {
		return 0
	}

	bytesPerFrame := int(fmtNumChannels) * (int(fmtBitsPerSample) / 8)
	if bytesPerFrame <= 0 {
		return 0
	}
	frames := dataChunkSize / bytesPerFrame
	return time.Duration(frames) * time.Second / time.Duration(fmtSampleRate)
}

func CalculateMP3DurationFromData(data []byte) time.Duration {
	if len(data) == 0 {
		return 0
	}

	reader := bytes.NewReader(data)
	decoder, format, err := stream.DecodeMP3Streaming(io.NopCloser(reader))
	if err != nil {
		return 0
	}
	defer decoder.Close()

	length := decoder.Len()
	if length > 0 {
		return format.SampleRate.D(length)
	}

	buf := make([][2]float64, 1024)
	totalSamples := 0
	for {
		n, ok := decoder.Stream(buf)
		if n == 0 {
			break
		}
		totalSamples += n
		if !ok {
			break
		}
	}

	if totalSamples > 0 {
		return format.SampleRate.D(totalSamples)
	}

	return 0
}

func CalculateMP3Duration(streamer interface {
	Len() int
	Position() int
	Seek(p int) error
}, sampleRate beep.SampleRate) time.Duration {
	if streamer == nil || sampleRate <= 0 {
		return 0
	}

	length := streamer.Len()
	if length > 0 {
		return sampleRate.D(length)
	}

	return 0
}
