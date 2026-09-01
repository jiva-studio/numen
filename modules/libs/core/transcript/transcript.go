// Package transcript says when a run of a recording's text was spoken.
//
// It is pure: no filesystem, no clock, no model. A viewer asks where a run of
// the words sits and is told the moment to play from.
//
// The moment is milliseconds from the start, so a player is given a number it
// already understands and nothing is recomputed.
package transcript

import (
	"encoding/binary"
	"sort"
)

// A Moment is one stretch of speech and where it was heard: the run of bytes it
// produced, and the milliseconds it spans.
type Moment struct {
	Start  int
	Length int
	FromMs int
	ToMs   int
}

// A Run is a stretch of a source's text, in bytes.
type Run struct {
	Start  int
	Length int
}

// At is where a run of the words sits: the moments it falls in, in the order
// they were heard. A run crossing a silence is in both of them.
func At(moments []Moment, start, length int) []Moment {
	if length <= 0 || len(moments) == 0 {
		return nil
	}
	end := start + length

	// The first moment that reaches into the run. A moment before it ends
	// before the run begins.
	at := sort.Search(len(moments), func(i int) bool {
		return moments[i].Start+moments[i].Length > start
	})

	var out []Moment
	for ; at < len(moments) && moments[at].Start < end; at++ {
		out = append(out, moments[at])
	}
	return out
}

// Plays is the millisecond a run is played from, and whether any moment holds
// it. A run the recording never said is nowhere to play.
func Plays(moments []Moment, start, length int) (int, bool) {
	found := At(moments, start, length)
	if len(found) == 0 {
		return 0, false
	}
	return found[0].FromMs, true
}

// The record one moment is written as: four numbers, each of them small enough
// for a recording of any length a person keeps.
const record = 4 * 4

// Pack writes the moments down.
func Pack(moments []Moment) []byte {
	raw := make([]byte, 0, len(moments)*record)
	var one [record]byte
	for _, m := range moments {
		binary.LittleEndian.PutUint32(one[0:], uint32(m.Start))
		binary.LittleEndian.PutUint32(one[4:], uint32(m.Length))
		binary.LittleEndian.PutUint32(one[8:], uint32(m.FromMs))
		binary.LittleEndian.PutUint32(one[12:], uint32(m.ToMs))
		raw = append(raw, one[:]...)
	}
	return raw
}

// Unpack is the moments a file holds. Bytes past the last whole record are a
// write that did not land, and they are not a moment.
func Unpack(raw []byte) []Moment {
	out := make([]Moment, 0, len(raw)/record)
	for at := 0; at+record <= len(raw); at += record {
		out = append(out, Moment{
			Start:  int(binary.LittleEndian.Uint32(raw[at:])),
			Length: int(binary.LittleEndian.Uint32(raw[at+4:])),
			FromMs: int(binary.LittleEndian.Uint32(raw[at+8:])),
			ToMs:   int(binary.LittleEndian.Uint32(raw[at+12:])),
		})
	}
	return out
}
