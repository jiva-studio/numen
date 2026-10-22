package recognition

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// getProofreadPages is the printed lines of a recognition, gathered by the page
// they were read from.
//
// A line is known by the index of its box in the recognition, so the number a
// correction is keyed by counts through the whole book. Boxes come in reading
// order, so pages and their lines come back in it.
//
// A box reaching past the prose was written for other bytes, and the
// recognition is refused whole. So is a box on a page the recognition has
// pages under one number would write one page's corrections onto the other's
// lines.
func getProofreadPages(prose string, boxes []domain.HighlightBox) []domain.ProofreadPage {
	var out []domain.ProofreadPage
	for at, box := range boxes {
		if box.Empty() {
			continue
		}
		if box.From < 0 || box.To > len(prose) {
			return nil
		}
		line := domain.ProofreadLine{Number: at, Text: prose[box.From:box.To]}
		if n := len(out); n > 0 {
			switch {
			case box.Page == out[n-1].Number:
				out[n-1].Lines = append(out[n-1].Lines, line)
				continue
			case box.Page < out[n-1].Number:
				return nil
			}
		}
		out = append(out, domain.ProofreadPage{Number: box.Page, Lines: []domain.ProofreadLine{line}})
	}
	return out
}
