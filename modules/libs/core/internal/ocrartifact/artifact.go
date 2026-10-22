// Package ocrartifact is the stored form of a read document: plain text with
// the pages marked in it, the boxes its prose was read from, and the parts it
// divides into.
package ocrartifact

import (
	"image"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A Page is one page of a document, read.
//
// It carries no name of its own. What a page is called is at its current position in
// the document, and that is what a person is shown and what a viewer opens at.
type Page struct {
	// Index is which page of the document this is, counted from zero.
	Index int
	// Size is the page as it was rendered, which the rectangles are addressed
	// from. A zero size is a page nothing was measured on.
	Size image.Point
	// Blocks are what it says, in the order it is read.
	Blocks []domain.OCRBlock
}

// The artifact is plain text with the pages marked in it:
//
//	\x0c\x0c
//	<the page's prose, one blank line between regions>
//	\x0c\x0c
//	…
//
// A form feed is what a page break has meant in plain text since long before
// any of this, no escaping is needed because a recogniser has no character for
// one, and a person opening the file sees the book.
//
// The mark says only that a page begins here. Where a page stands among the
// marks is what it is called, and a second name for one page is a second thing
// to be wrong about.
const (
	pageMark  = '\x0c'
	blockGap  = "\n\n"
	pageStart = "\x0c"
)

// Write is the artifact for a document that has been read, the boxes its prose
// was read from, and the parts it divides into.
//
// A page with nothing on it is still written: its mark is what makes the page
// after it findable, and a blank page is a fact about the document.
//
// A box and a part are both placed in the prose, which is what Read gives back.
// The mark and the newline closing it are bookkeeping and are counted in none of
// them.
func Write(pages []Page) ([]byte, []domain.HighlightBox, []domain.OCRPart) {
	var out strings.Builder
	var boxes []domain.HighlightBox
	var parts []domain.OCRPart
	prose := 0
	for _, page := range pages {
		out.WriteString(pageStart)
		out.WriteString(pageStart)
		out.WriteString("\n")
		for i, block := range page.Blocks {
			if i > 0 {
				out.WriteString(blockGap)
				prose += len(blockGap)
			}
			boxes = append(boxes, getBoxesWithin(page, block, prose)...)
			if block.IsHeading && block.Text != "" {
				parts = append(parts, domain.OCRPart{Start: prose, Length: len(block.Text), Depth: block.Depth})
			}
			out.WriteString(block.Text)
			prose += len(block.Text)
		}
		out.WriteString("\n")
		prose++
	}
	return []byte(out.String()), boxes, parts
}

// getBoxesWithin is where each box of a block sits: at its offset from base in
// the prose, and over the fraction of the page its rectangle covers. A page
// nothing was measured on gives no boxes, having no size to take a fraction of.
func getBoxesWithin(page Page, block domain.OCRBlock, base int) []domain.HighlightBox {
	if page.Size.X <= 0 || page.Size.Y <= 0 {
		return nil
	}
	wide, high := float32(page.Size.X), float32(page.Size.Y)
	boxes := make([]domain.HighlightBox, 0, len(block.Boxes))
	for _, one := range block.Boxes {
		boxes = append(boxes, domain.HighlightBox{
			Page:     page.Index,
			ByteSpan: domain.ByteSpan{From: base + one.Span.From, To: base + one.Span.To},
			Rect: domain.Rect{
				MinX: float32(one.Rect.Min.X) / wide,
				MinY: float32(one.Rect.Min.Y) / high,
				MaxX: float32(one.Rect.Max.X) / wide,
				MaxY: float32(one.Rect.Max.Y) / high,
			},
		})
	}
	return boxes
}

// Note is a line the artifact carries for whatever wrote it and nobody else: a
// run stopped part way says here how far it got. It begins with a byte no
// recogniser can write, so a line of the book is never mistaken for one.
const Note = "\x00"

// Read is an artifact, as the text a chunk is a place in and the pages that
// text names.
//
// The marks are taken out of the text: an offset in what comes back is an offset
// in the prose, so a chunk cut from it holds what the page says and not the
// bookkeeping around it.
//
// Anything before the first mark is prose belonging to no page, which is what a
// file written by something else looks like. It is kept, because dropping text
// silently is worse than naming its page wrongly.
func Read(raw []byte) (string, []domain.OCRPageStart) {
	text := withoutNotes(string(raw))
	if !strings.ContainsRune(text, pageMark) {
		return text, nil
	}

	var out strings.Builder
	var marks []domain.OCRPageStart
	rest := text
	for {
		before, after, found := strings.Cut(rest, pageStart)
		out.WriteString(before)
		if !found {
			break
		}
		_, prose, closed := strings.Cut(after, pageStart)
		if !closed {
			// A mark that never closes is not a mark. What follows is prose.
			out.WriteString(after)
			break
		}
		prose = strings.TrimPrefix(prose, "\n")
		marks = append(marks, domain.OCRPageStart{Offset: out.Len()})
		rest = prose
	}
	return out.String(), marks
}

// withoutNotes drops the lines an artifact carries for whatever wrote it. They
// are not what the page says, and an offset into the prose must not count them.
func withoutNotes(text string) string {
	if !strings.Contains(text, Note) {
		return text
	}
	var out strings.Builder
	for rest := text; rest != ""; {
		line, more, found := strings.Cut(rest, "\n")
		if !strings.HasPrefix(line, Note) {
			out.WriteString(line)
			if found {
				out.WriteString("\n")
			}
		}
		if !found {
			break
		}
		rest = more
	}
	return out.String()
}
