package proofread

import (
	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// Scanned is the printed lines of a reading, gathered by the page they were
// read from.
//
// A line is known by the index of its box in the reading, so the number a
// correction is keyed by counts through the whole book. Boxes come in reading
// order, so batches and their lines come back in it.
//
// A box reaching past the prose was written for other bytes, and the reading is
// refused whole.
func Scanned(prose string, boxes []lit.Box) []Batch {
	var out []Batch
	for at, box := range boxes {
		if box.Length <= 0 {
			continue
		}
		end := box.Start + box.Length
		if box.Start < 0 || end > len(prose) {
			return nil
		}
		line := Line{At: at, Text: prose[box.Start:end]}
		if n := len(out); n > 0 && out[n-1].At == box.Page {
			out[n-1].Lines = append(out[n-1].Lines, line)
			continue
		}
		out = append(out, Batch{At: box.Page, Lines: []Line{line}})
	}
	return out
}

// Spoken is the cues of a transcript, cut into batches of size lines, each
// batch opening on the last overlap lines of the one before it. Speech runs on
// past the cut, so the lines a batch shares with its neighbour are seen whole
// by one of the two.
//
// A line is known by the index of its cue in the transcript, and a batch by its
// place in the run. A cue saying nothing carries no line.
func Spoken(cues []transcript.Cue, size, overlap int) []Batch {
	if size <= 0 {
		return nil
	}
	var lines []Line
	for at, cue := range cues {
		if cue.Text == "" {
			continue
		}
		lines = append(lines, Line{At: at, Text: cue.Text})
	}

	step := size - shared(size, overlap)
	var out []Batch
	for start := 0; start < len(lines); start += step {
		end := min(start+size, len(lines))
		out = append(out, Batch{At: len(out), Lines: lines[start:end:end]})
		if end == len(lines) {
			break
		}
	}
	return out
}

// shared is how many lines a batch keeps from the one before it: never as many
// as size, so each batch reaches further into the transcript than its
// neighbour.
func shared(size, overlap int) int {
	if overlap < 0 {
		return 0
	}
	return min(overlap, size-1)
}
