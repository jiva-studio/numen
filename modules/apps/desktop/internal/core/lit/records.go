package lit

import (
	"encoding/binary"
	"math"
)

// A record is one box, and every record is this wide. They are written in page
// order and prose grows within a page, so they come out sorted by Start and
// need no index of their own.
const record = 28

// Pack is boxes as bytes: fixed-width records, so a reader wanting one page
// seeks to it and reads no more.
func Pack(boxes []Box) []byte {
	raw := make([]byte, 0, len(boxes)*record)
	var one [record]byte
	for _, box := range boxes {
		binary.LittleEndian.PutUint32(one[0:], uint32(int32(box.Page)))
		binary.LittleEndian.PutUint32(one[4:], uint32(int32(box.Start)))
		binary.LittleEndian.PutUint32(one[8:], uint32(int32(box.Length)))
		binary.LittleEndian.PutUint32(one[12:], math.Float32bits(box.MinX))
		binary.LittleEndian.PutUint32(one[16:], math.Float32bits(box.MinY))
		binary.LittleEndian.PutUint32(one[20:], math.Float32bits(box.MaxX))
		binary.LittleEndian.PutUint32(one[24:], math.Float32bits(box.MaxY))
		raw = append(raw, one[:]...)
	}
	return raw
}

// Unpack is the boxes bytes hold. Bytes that are not a whole number of records
// give back the records they do hold.
func Unpack(raw []byte) []Box {
	if len(raw) < record {
		return nil
	}
	boxes := make([]Box, 0, len(raw)/record)
	for at := 0; at+record <= len(raw); at += record {
		one := raw[at : at+record]
		boxes = append(boxes, Box{
			Page:   int(int32(binary.LittleEndian.Uint32(one[0:]))),
			Start:  int(int32(binary.LittleEndian.Uint32(one[4:]))),
			Length: int(int32(binary.LittleEndian.Uint32(one[8:]))),
			MinX:   math.Float32frombits(binary.LittleEndian.Uint32(one[12:])),
			MinY:   math.Float32frombits(binary.LittleEndian.Uint32(one[16:])),
			MaxX:   math.Float32frombits(binary.LittleEndian.Uint32(one[20:])),
			MaxY:   math.Float32frombits(binary.LittleEndian.Uint32(one[24:])),
		})
	}
	return boxes
}
