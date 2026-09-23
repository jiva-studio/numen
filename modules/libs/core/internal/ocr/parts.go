package ocr

import "encoding/binary"

// A Part is one of the parts a document divides into: where its heading begins
// in the prose, how far the heading runs, and how deep the part sits.
//
// A part carries a run of the prose, so a heading the scan read badly still
// says where the part begins.
type Part struct {
	Start  int
	Length int
	// Depth is how far in the part sits, counted from zero. A document title
	// stands above the section titles within it.
	Depth int
}

// A part record is one part, and every record is this wide. They are written in
// the order the prose is read, so they come out sorted by Start and need no
// index of their own.
const partRecord = 12

// Pack is parts as bytes: fixed-width records, so a reader wanting one part
// seeks to it and reads no more.
func Pack(parts []Part) []byte {
	raw := make([]byte, 0, len(parts)*partRecord)
	var one [partRecord]byte
	for _, part := range parts {
		binary.LittleEndian.PutUint32(one[0:], uint32(int32(part.Start)))
		binary.LittleEndian.PutUint32(one[4:], uint32(int32(part.Length)))
		binary.LittleEndian.PutUint32(one[8:], uint32(int32(part.Depth)))
		raw = append(raw, one[:]...)
	}
	return raw
}

// Unpack is the parts bytes hold. Bytes that are not a whole number of records
// give back the records they do hold.
func Unpack(raw []byte) []Part {
	if len(raw) < partRecord {
		return nil
	}
	parts := make([]Part, 0, len(raw)/partRecord)
	for at := 0; at+partRecord <= len(raw); at += partRecord {
		one := raw[at : at+partRecord]
		parts = append(parts, Part{
			Start:  int(int32(binary.LittleEndian.Uint32(one[0:]))),
			Length: int(int32(binary.LittleEndian.Uint32(one[4:]))),
			Depth:  int(int32(binary.LittleEndian.Uint32(one[8:]))),
		})
	}
	return parts
}
