package ocr

import (
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
)

// The artifact is plain text with the pages marked in it:
//
//	\x0c<page label>\x0c
//	<the page's prose, one blank line between regions>
//	\x0c<next page label>\x0c
//	…
//
// A form feed is what a page break has meant in plain text since long before
// any of this, no escaping is needed because a recogniser has no character for
// one, and a person opening the file sees the book.
//
// The mark carries the number the page prints on itself, which is what a person
// holding the book would say, and is empty where nothing names the page. Where a
// page stands among the marks is where it stands in the file.
const (
	pageMark  = '\x0c'
	blockGap  = "\n\n"
	pageStart = "\x0c"
)

// A Mark is a page of the artifact, at the offset its prose begins.
type Mark struct {
	// Label is the number the page prints on itself, empty where nothing names
	// the page.
	Label  string
	Offset int
}

// Write is the artifact for a document that has been read, and the boxes its
// prose was read from.
//
// A page with nothing on it is still written: its mark is what makes the page
// after it findable, and a blank page is a fact about the document.
//
// A box is placed in the prose, which is what Read gives back. The mark and the
// newline closing it are bookkeeping and are counted in neither.
func Write(pages []Page) ([]byte, []placed.Box) {
	var out strings.Builder
	var boxes []placed.Box
	prose := 0
	for _, page := range pages {
		out.WriteString(pageStart)
		out.WriteString(strings.ReplaceAll(called(page), pageStart, ""))
		out.WriteString(pageStart)
		out.WriteString("\n")
		for i, block := range page.Blocks {
			if i > 0 {
				out.WriteString(blockGap)
				prose += len(blockGap)
			}
			boxes = append(boxes, within(page, block, prose)...)
			out.WriteString(block.Text)
			prose += len(block.Text)
		}
		out.WriteString("\n")
		prose++
	}
	return []byte(out.String()), boxes
}

// called is what a page is called: the number it prints on itself, and what the
// document calls it where it prints none.
func called(page Page) string {
	if page.Number != "" {
		return page.Number
	}
	return page.Label
}

// within is where each span of a block sits: at its offset from base in the
// prose, and over the fraction of the page its rectangle covers. A page nothing
// was measured on gives no boxes, having no size to take a fraction of.
func within(page Page, block Block, base int) []placed.Box {
	if page.Size.X <= 0 || page.Size.Y <= 0 {
		return nil
	}
	wide, high := float32(page.Size.X), float32(page.Size.Y)
	boxes := make([]placed.Box, 0, len(block.Spans))
	for _, span := range block.Spans {
		boxes = append(boxes, placed.Box{
			Page:   page.At,
			Start:  base + span.Start,
			Length: span.Length,
			MinX:   float32(span.Box.Min.X) / wide,
			MinY:   float32(span.Box.Min.Y) / high,
			MaxX:   float32(span.Box.Max.X) / wide,
			MaxY:   float32(span.Box.Max.Y) / high,
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
// in the prose, so a window cut from it holds what the page says and not the
// bookkeeping around it.
//
// Anything before the first mark is prose belonging to no page, which is what a
// file written by something else looks like. It is kept, because dropping text
// silently is worse than naming its page wrongly.
func Read(raw []byte) (string, []Mark) {
	text := withoutNotes(string(raw))
	if !strings.ContainsRune(text, pageMark) {
		return text, nil
	}

	var out strings.Builder
	var marks []Mark
	rest := text
	for {
		before, after, found := strings.Cut(rest, pageStart)
		out.WriteString(before)
		if !found {
			break
		}
		label, prose, closed := strings.Cut(after, pageStart)
		if !closed {
			// A mark that never closes is not a mark. What follows is prose.
			out.WriteString(after)
			break
		}
		prose = strings.TrimPrefix(prose, "\n")
		marks = append(marks, Mark{Label: label, Offset: out.Len()})
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
