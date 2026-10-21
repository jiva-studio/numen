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
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mostEntries is the upper bound on outline entries to prevent runaway memory"
	mostEntries = 10_000
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mostDepth is the recursion limit to prevent stack exhaustion on cyclic outlines"
	mostDepth = 32
)

// outline is what the document's own outline names, as parts of its text.
//
// An entry points at a page, so a part begins where that page begins. That is
// coarser than an EPUB's anchor, which points inside a document, and it is what
// the format offers: a destination carries a position on the page, but the text
// layer's order is not the page's geometry, so a position cannot be turned into
// an offset without guessing.
//
// starts is where each page's text begins, so an entry pointing past the pages
// that were read is empty and is dropped.
func (d *document) outline(starts []int) []Part {
	w := &outlineWalk{doc: d, starts: starts, seen: map[references.FPDF_BOOKMARK]bool{}}
	w.walk(nil, 1)
	return w.parts
}

// outlineWalk is the reading of an outline: the parts found, and the entries
// already read.
type outlineWalk struct {
	doc    *document
	starts []int
	parts  []Part
	seen   map[references.FPDF_BOOKMARK]bool
}

// walk reads the children of a bookmark, each with what lies under it, down to
// the deepest level the outline is read to.
func (w *outlineWalk) walk(bookmark *references.FPDF_BOOKMARK, level int) {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="recursion stops when level exceeds mostDepth"
	if level > mostDepth {
		return
	}
	d := w.doc
	child, err := d.worker.FPDFBookmark_GetFirstChild(&requests.FPDFBookmark_GetFirstChild{
		Document: d.ref, Bookmark: bookmark,
	})
	// gomutants:disable-next-line INVERT_LOGICAL reason="walk stops when there are no child bookmarks or an error occurs"
	if err != nil || child.Bookmark == nil {
		return
	}
	for at := child.Bookmark; at != nil; {
		// gomutants:disable-next-line BRANCH_IF reason="walk stops when visit limit is reached or entry is already seen"
		if !w.visit(at, level) {
			return
		}
		next, err := d.worker.FPDFBookmark_GetNextSibling(&requests.FPDFBookmark_GetNextSibling{
			Document: d.ref, Bookmark: *at,
		})
		// gomutants:disable-next-line BRANCH_IF reason="walk stops when next sibling cannot be retrieved"
		if err != nil {
			return
		}
		at = next.Bookmark
	}
}

// visit reads one entry and what lies under it, and says whether the walk
// goes on. An outline that points back at itself is read once.
func (w *outlineWalk) visit(at *references.FPDF_BOOKMARK, level int) bool {
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="visit drops seen bookmarks to prevent cycles and respects mostEntries limit"
	if len(w.parts) >= mostEntries || w.seen[*at] {
		return false
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="marks bookmark as seen to prevent cycles"
	w.seen[*at] = true

	if part, ok := w.doc.readPart(*at, level, w.starts); ok {
		w.parts = append(w.parts, part)
	}
	w.walk(at, level+1)
	return true
}

// readPart is one outline entry, as a name and the offset of the page it leads to.
// An entry with no name, or one that leads nowhere, is not a readPart.
func (d *document) readPart(bookmark references.FPDF_BOOKMARK, level int, starts []int) (Part, bool) {
	named, err := d.worker.FPDFBookmark_GetTitle(&requests.FPDFBookmark_GetTitle{Bookmark: bookmark})
	// gomutants:disable-next-line BRANCH_IF reason="unreadable outline bookmark title drops the entry"
	if err != nil {
		return Part{}, false
	}
	title := normalizeSpaces(named.Title)
	// gomutants:disable-next-line BRANCH_IF reason="empty bookmark title drops the entry"
	if title == "" {
		return Part{}, false
	}
	page, ok := d.getDestinationPage(bookmark)
	// gomutants:disable-next-line INVERT_LOGICAL reason="missing or out-of-range destinations drop the entry"
	if !ok || page < 0 || page >= len(starts) {
		return Part{}, false
	}
	return Part{Title: title, Offset: starts[page], Level: level}, true
}

// getDestinationPage is the page an outline entry leads to.
//
// An entry carries its getDestinationPage directly or carries an action that holds
// one. Both are asked, because which of the two a file uses is the choice of
// whatever wrote it.
func (d *document) getDestinationPage(bookmark references.FPDF_BOOKMARK) (int, bool) {
	dest, err := d.worker.FPDFBookmark_GetDest(&requests.FPDFBookmark_GetDest{
		Document: d.ref, Bookmark: bookmark,
	})
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="destination lookup requires both successful call and present destination"
	if err == nil && dest.Dest != nil {
		return d.pageOf(*dest.Dest)
	}

	action, err := d.worker.FPDFBookmark_GetAction(&requests.FPDFBookmark_GetAction{Bookmark: bookmark})
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="action lookup requires both successful call and present action"
	if err != nil || action.Action == nil {
		return 0, false
	}
	kind, err := d.worker.FPDFAction_GetType(&requests.FPDFAction_GetType{Action: *action.Action})
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="only GoTo actions are resolved to document pages"
	if err != nil || kind.Type != enums.FPDF_ACTION_ACTION_GOTO {
		// An action that opens a file or a URL leads out of this document, and
		// there is nowhere in this text to put it.
		return 0, false
	}
	within, err := d.worker.FPDFAction_GetDest(&requests.FPDFAction_GetDest{
		Document: d.ref, Action: *action.Action,
	})
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="action destination lookup requires both successful call and present destination"
	if err != nil || within.Dest == nil {
		return 0, false
	}
	return d.pageOf(*within.Dest)
}

func (d *document) pageOf(dest references.FPDF_DEST) (int, bool) {
	at, err := d.worker.FPDFDest_GetDestPageIndex(&requests.FPDFDest_GetDestPageIndex{
		Document: d.ref, Dest: dest,
	})
	// gomutants:disable-next-line BRANCH_IF reason="invalid destination drops destination page lookup"
	if err != nil {
		return 0, false
	}
	return at.Index, true
}
