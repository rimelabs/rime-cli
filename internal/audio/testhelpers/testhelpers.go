package testhelpers

import (
	"bytes"
	"encoding/binary"
)

func MakeValidWAV(sampleCount int) []byte {
	var buf bytes.Buffer

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+sampleCount*2))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint32(24000))
	binary.Write(&buf, binary.LittleEndian, uint32(48000))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint16(16))

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(sampleCount*2))

	for i := 0; i < sampleCount; i++ {
		binary.Write(&buf, binary.LittleEndian, int16(0))
	}

	return buf.Bytes()
}

func MakeMinimalWAV() []byte {
	var buf bytes.Buffer

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint32(44100))
	binary.Write(&buf, binary.LittleEndian, uint32(88200))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint16(16))

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	return buf.Bytes()
}

func MakeMinimalMP3() []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFB, 0x90, 0x00})
	buf.Write(make([]byte, 100))
	return buf.Bytes()
}
