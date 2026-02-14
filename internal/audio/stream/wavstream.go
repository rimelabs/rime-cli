package stream

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/gopxl/beep/v2"
)

type wavHeader struct {
	SampleRate    uint32
	NumChannels   uint16
	BitsPerSample uint16
	ByteRate      uint32
	BlockAlign    uint16
}

type StreamingDecoder struct {
	r              io.Reader
	format         beep.Format
	bytesPerSample int
	buf            []byte
	err            error
}

func DecodeStreaming(r io.Reader) (*StreamingDecoder, beep.Format, error) {
	header, err := readWavHeader(r)
	if err != nil {
		return nil, beep.Format{}, err
	}

	precision := int(header.BitsPerSample / 8)
	format := beep.Format{
		SampleRate:  beep.SampleRate(header.SampleRate),
		NumChannels: int(header.NumChannels),
		Precision:   precision,
	}

	bytesPerSample := int(header.NumChannels) * precision

	return &StreamingDecoder{
		r:              r,
		format:         format,
		bytesPerSample: bytesPerSample,
		buf:            make([]byte, bytesPerSample*512),
	}, format, nil
}

func (d *StreamingDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}

	bytesNeeded := len(samples) * d.bytesPerSample
	if len(d.buf) < bytesNeeded {
		d.buf = make([]byte, bytesNeeded)
	}

	numRead, err := io.ReadFull(d.r, d.buf[:bytesNeeded])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		d.err = err
		return 0, false
	}

	if numRead == 0 {
		return 0, false
	}

	numSamples := numRead / d.bytesPerSample

	switch d.format.Precision {
	case 1:
		for i := 0; i < numSamples; i++ {
			offset := i * d.bytesPerSample
			for ch := 0; ch < d.format.NumChannels && ch < 2; ch++ {
				val := float64(d.buf[offset+ch])/128.0 - 1.0
				samples[i][ch] = val
			}
			if d.format.NumChannels == 1 {
				samples[i][1] = samples[i][0]
			}
		}
	case 2:
		for i := 0; i < numSamples; i++ {
			offset := i * d.bytesPerSample
			for ch := 0; ch < d.format.NumChannels && ch < 2; ch++ {
				val := int16(binary.LittleEndian.Uint16(d.buf[offset+ch*2:]))
				samples[i][ch] = float64(val) / 32768.0
			}
			if d.format.NumChannels == 1 {
				samples[i][1] = samples[i][0]
			}
		}
	case 3:
		for i := 0; i < numSamples; i++ {
			offset := i * d.bytesPerSample
			for ch := 0; ch < d.format.NumChannels && ch < 2; ch++ {
				b := d.buf[offset+ch*3:]
				val := int32(b[0]) | int32(b[1])<<8 | int32(int8(b[2]))<<16
				samples[i][ch] = float64(val) / 8388608.0
			}
			if d.format.NumChannels == 1 {
				samples[i][1] = samples[i][0]
			}
		}
	}

	return numSamples, true
}

func (d *StreamingDecoder) Err() error {
	return d.err
}

func readWavHeader(r io.Reader) (*wavHeader, error) {
	var buf [12]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("failed to read RIFF header: %w", err)
	}

	if string(buf[0:4]) != "RIFF" {
		return nil, fmt.Errorf("not a RIFF file")
	}
	if string(buf[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAVE file")
	}

	var header wavHeader

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			return nil, fmt.Errorf("failed to read chunk header: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, fmt.Errorf("fmt chunk too small")
			}
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("failed to read fmt chunk: %w", err)
			}

			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			if audioFormat != 1 {
				return nil, fmt.Errorf("unsupported audio format: %d (only PCM supported)", audioFormat)
			}

			header.NumChannels = binary.LittleEndian.Uint16(fmtData[2:4])
			header.SampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			header.ByteRate = binary.LittleEndian.Uint32(fmtData[8:12])
			header.BlockAlign = binary.LittleEndian.Uint16(fmtData[12:14])
			header.BitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])

		case "data":
			return &header, nil

		default:
			if _, err := io.CopyN(io.Discard, r, int64(chunkSize)); err != nil {
				return nil, fmt.Errorf("failed to skip %s chunk: %w", chunkID, err)
			}
		}

		if chunkSize%2 != 0 {
			io.CopyN(io.Discard, r, 1)
		}
	}
}
