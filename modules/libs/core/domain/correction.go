package domain

import "encoding/binary"

// A CorrectionLine is one printed line put right: the number the line is known by in the
// recognition, and what it should say.
type CorrectionLine struct {
	Number int
	Text   string
}

// A record is one corrected line: its number, the bytes of its text, and the
// text. Records are variable width and carry no count, so a proofreading run
// puts down what came back for a batch of pages with an append.
const correctionHead = 8

// PackCorrections is corrections as bytes.
func PackCorrections(lines []CorrectionLine) []byte {
	raw := make([]byte, 0, len(lines)*(correctionHead+32))
	var one [correctionHead]byte
	for _, line := range lines {
		binary.LittleEndian.PutUint32(one[0:], uint32(int32(line.Number)))
		binary.LittleEndian.PutUint32(one[4:], uint32(len(line.Text)))
		raw = append(raw, one[:]...)
		raw = append(raw, line.Text...)
	}
	return raw
}

// UnpackCorrections is the corrections bytes hold, in file order. Bytes that stop part way
// through a record give back the whole records before them.
func UnpackCorrections(raw []byte) []CorrectionLine {
	var lines []CorrectionLine
	for at := 0; at+correctionHead <= len(raw); {
		n := uint64(binary.LittleEndian.Uint32(raw[at+4:]))
		if uint64(at)+correctionHead+n > uint64(len(raw)) {
			break
		}
		text := raw[at+correctionHead : at+correctionHead+int(n)]
		lines = append(lines, CorrectionLine{
			Number: int(int32(binary.LittleEndian.Uint32(raw[at:]))),
			Text:   string(text),
		})
		at += correctionHead + int(n)
	}
	return lines
}
