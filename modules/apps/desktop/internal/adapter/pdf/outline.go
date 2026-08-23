package pdf

import (
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
)

// What one outline may spend. An outline is written by whoever made the file,
// so it is as deep and as long as they liked, and a cycle in it is a program
// that does not stop.
const (
	mostEntries = 10_000
	mostDepth   = 32
)

// outline is what the document's own outline names, as places in its text.
//
// An entry points at a page, so a place begins where that page begins. That is
// coarser than an EPUB's anchor, which points inside a document, and it is what
// the format offers: a destination carries a position on the page, but the text
// layer's order is not the page's geometry, so a position cannot be turned into
// an offset without guessing.
//
// starts is where each page's text begins, so an entry pointing past the pages
// that were read names nothing and is dropped.
func (d *document) outline(starts []int) []Place {
	var places []Place
	seen := map[references.FPDF_BOOKMARK]bool{}

	var walk func(bookmark *references.FPDF_BOOKMARK, level int)
	walk = func(bookmark *references.FPDF_BOOKMARK, level int) {
		if level > mostDepth {
			return
		}
		child, err := d.worker.FPDFBookmark_GetFirstChild(&requests.FPDFBookmark_GetFirstChild{
			Document: d.ref, Bookmark: bookmark,
		})
		if err != nil || child.Bookmark == nil {
			return
		}
		for at := child.Bookmark; at != nil; {
			if len(places) >= mostEntries || seen[*at] {
				// An outline that points back at itself is read once.
				return
			}
			seen[*at] = true

			if place, ok := d.place(*at, level, starts); ok {
				places = append(places, place)
			}
			walk(at, level+1)

			next, err := d.worker.FPDFBookmark_GetNextSibling(&requests.FPDFBookmark_GetNextSibling{
				Document: d.ref, Bookmark: *at,
			})
			if err != nil {
				return
			}
			at = next.Bookmark
		}
	}
	walk(nil, 1)
	return places
}

// place is one outline entry, as a name and the offset of the page it leads to.
// An entry with no name, or one that leads nowhere, is not a place.
func (d *document) place(bookmark references.FPDF_BOOKMARK, level int, starts []int) (Place, bool) {
	named, err := d.worker.FPDFBookmark_GetTitle(&requests.FPDFBookmark_GetTitle{Bookmark: bookmark})
	if err != nil {
		return Place{}, false
	}
	title := tidy(named.Title)
	if title == "" {
		return Place{}, false
	}
	page, ok := d.destination(bookmark)
	if !ok || page < 0 || page >= len(starts) {
		return Place{}, false
	}
	return Place{Title: title, Offset: starts[page], Level: level}, true
}

// destination is the page an outline entry leads to.
//
// An entry carries its destination directly or carries an action that holds
// one. Both are asked, because which of the two a file uses is the choice of
// whatever wrote it.
func (d *document) destination(bookmark references.FPDF_BOOKMARK) (int, bool) {
	dest, err := d.worker.FPDFBookmark_GetDest(&requests.FPDFBookmark_GetDest{
		Document: d.ref, Bookmark: bookmark,
	})
	if err == nil && dest.Dest != nil {
		return d.pageOf(*dest.Dest)
	}

	action, err := d.worker.FPDFBookmark_GetAction(&requests.FPDFBookmark_GetAction{Bookmark: bookmark})
	if err != nil || action.Action == nil {
		return 0, false
	}
	kind, err := d.worker.FPDFAction_GetType(&requests.FPDFAction_GetType{Action: *action.Action})
	if err != nil || kind.Type != enums.FPDF_ACTION_ACTION_GOTO {
		// An action that opens a file or a URL leads out of this document, and
		// there is nowhere in this text to put it.
		return 0, false
	}
	within, err := d.worker.FPDFAction_GetDest(&requests.FPDFAction_GetDest{
		Document: d.ref, Action: *action.Action,
	})
	if err != nil || within.Dest == nil {
		return 0, false
	}
	return d.pageOf(*within.Dest)
}

func (d *document) pageOf(dest references.FPDF_DEST) (int, bool) {
	at, err := d.worker.FPDFDest_GetDestPageIndex(&requests.FPDFDest_GetDestPageIndex{
		Document: d.ref, Dest: dest,
	})
	if err != nil {
		return 0, false
	}
	return at.Index, true
}
