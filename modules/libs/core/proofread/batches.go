package proofread

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
)

// Scanned is the printed lines of a reading, gathered by the page they were
// read from.
//
// A line is known by the index of its box in the reading, so the number a
// correction is keyed by counts through the whole book. Boxes come in reading
// order, so batches and their lines come back in it.
//
// A box reaching past the prose was written for other bytes, and the reading is
// refused whole. So is a box on a page the reading has already left behind: a
// batch is asked about and answered for by its page, and two batches under one
// page would take one reply between them, writing one page's corrections onto
// the other's lines.
func Scanned(prose string, boxes []highlight.Box) []Batch {
	var out []Batch
	for at, box := range boxes {
		if box.Empty() {
			continue
		}
		if box.From < 0 || box.To > len(prose) {
			return nil
		}
		line := Line{Number: at, Text: prose[box.From:box.To]}
		if n := len(out); n > 0 {
			switch {
			case box.Page == out[n-1].Number:
				out[n-1].Lines = append(out[n-1].Lines, line)
				continue
			case box.Page < out[n-1].Number:
				return nil
			}
		}
		out = append(out, Batch{Number: box.Page, Lines: []Line{line}})
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
//
// A run of lines is answered for inside one batch, so a sentence broken over a
// cut is put back together here where the overlap carries it whole.
func Spoken(cues []transcript.Cue, size, overlap int) []Batch {
	if size <= 0 {
		return nil
	}
	lines := getSpokenLines(cues)
	step := size - getSharedLines(size, overlap)
	var out []Batch
	for start := 0; start < len(lines); start += step {
		end := min(start+size, len(lines))
		out = append(out, Batch{Number: len(out), Lines: lines[start:end:end], Joinable: true})
		if end == len(lines) {
			break
		}
	}
	return out
}

// Seams is a batch for each of the cuts, holding the size lines around that
// cut, half of them before it and half after, and numbered on from the batches
// Spoken gives back. A cut is named by the batch it comes after, and the cuts
// come in the order they stand in the transcript.
//
// A run of lines is answered for inside one batch, and a sentence reaching
// across a cut further than the overlap carries is in no batch Spoken makes.
// The seam batch for that cut holds it whole.
//
// A transcript of one batch has no cut, and a batch of one line has no room
// for a line on either side of one. The last batch is followed by no cut.
func Seams(cues []transcript.Cue, size, overlap int, cuts []int) []Batch {
	batches := Spoken(cues, size, overlap)
	if size < 2 || len(batches) < 2 {
		return nil
	}

	lines := getSpokenLines(cues)
	step := size - getSharedLines(size, overlap)
	var out []Batch
	reach := -1
	for _, cut := range cuts {
		if cut < 0 || cut >= len(batches)-1 {
			continue
		}
		// One window covers two cuts near the end of the transcript, and each
		// seam batch reaches further than the one before it.
		end := min(max((cut+1)*step-size/2, 0)+size, len(lines))
		if end <= reach {
			continue
		}
		reach = end
		start := max(end-size, 0)
		out = append(out, Batch{
			Number:   len(batches) + len(out),
			Lines:    lines[start:end:end],
			Joinable: true,
		})
	}
	return out
}

// getSpokenLines is every cue that says something, as a line known by the index
// of its cue in the transcript.
func getSpokenLines(cues []transcript.Cue) []Line {
	var out []Line
	for at, cue := range cues {
		if cue.Text == "" {
			continue
		}
		out = append(out, Line{Number: at, Text: cue.Text})
	}
	return out
}

// getSharedLines is how many lines a batch keeps from the one before it: never
// as many as size, so each batch reaches further into the transcript than its
// neighbour.
func getSharedLines(size, overlap int) int {
	if overlap < 0 {
		return 0
	}
	return min(overlap, size-1)
}
