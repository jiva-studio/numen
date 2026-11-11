package document

import "github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"

// Place is the rectangles of a run of text on one page.
type Place struct {
	Page  int
	Rects []pdf.Rect
}

// GetPlaces merges the boxes of a run into one place for each page. Boxes of one page that follow
// each other share a place; a page the boxes return to starts a new one.
func GetPlaces(boxes []pdf.HighlightBox) []Place {
	places := []Place{}
	for _, box := range boxes {
		if n := len(places); n > 0 && places[n-1].Page == box.Page {
			places[n-1].Rects = append(places[n-1].Rects, box.Rect)
			continue
		}
		places = append(places, Place{Page: box.Page, Rects: []pdf.Rect{box.Rect}})
	}
	return places
}
