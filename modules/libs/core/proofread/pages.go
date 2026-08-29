package proofread

import "github.com/jiva-studio/numen/modules/libs/core/lit"

// Pages are the printed lines of a reading, gathered by the page they were read
// from.
//
// A line is known by the index of its box in the reading, so the number a
// correction is keyed by counts through the whole book. Boxes come in reading
// order, so pages and their lines come back in it.
//
// A box reaching past the prose was written for other bytes, and the reading is
// refused whole.
func Pages(prose string, boxes []lit.Box) []Page {
	var out []Page
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
		out = append(out, Page{At: box.Page, Lines: []Line{line}})
	}
	return out
}
