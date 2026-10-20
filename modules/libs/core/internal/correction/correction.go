// Package correction is the byte form of the corrections a proofreading run
// stores beside a recognition.
//
// It is pure: no filesystem, no clock, no model.
package correction

import (
	"encoding/binary"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A record is one corrected line: its number, the bytes of its text, and the
// text. Records are variable width and carry no count, so a proofreading run
// puts down what came back for a batch of pages with an append.
const head = 8

// Pack is corrections as bytes.
func Pack(lines []domain.CorrectionLine) []byte {
	raw := make([]byte, 0, len(lines)*(head+32))
	var one [head]byte
	for _, line := range lines {
		binary.LittleEndian.PutUint32(one[0:], uint32(int32(line.Number)))
		binary.LittleEndian.PutUint32(one[4:], uint32(len(line.Text)))
		raw = append(raw, one[:]...)
		raw = append(raw, line.Text...)
	}
	return raw
}

// Unpack is the corrections bytes hold, in file order. Bytes that stop part way
// through a record give back the whole records before them.
func Unpack(raw []byte) []domain.CorrectionLine {
	var lines []domain.CorrectionLine
	for at := 0; at+head <= len(raw); {
		n := uint64(binary.LittleEndian.Uint32(raw[at+4:]))
		if uint64(at)+head+n > uint64(len(raw)) {
			break
		}
		text := raw[at+head : at+head+int(n)]
		lines = append(lines, domain.CorrectionLine{
			Number: int(int32(binary.LittleEndian.Uint32(raw[at:]))),
			Text:   string(text),
		})
		at += head + int(n)
	}
	return lines
}
