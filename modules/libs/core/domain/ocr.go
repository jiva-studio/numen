package domain

import "image"

// An OCRBox is one box of a region, in what was written from it: the rectangle
// it covers on the page, and the run of bytes it produced.
type OCRBox struct {
	Rect image.Rectangle
	Span ByteSpan
}

// An OCRBlock is one region of a page, written out.
type OCRBlock struct {
	Label string
	Text  string
	// IsHeading says the region is a heading, and Depth is how far in the part it
	// opens sits. A document title stands above the section titles within it.
	IsHeading bool
	Depth     int
	// Boxes are where on the page each run of Text was read. A recogniser that
	// reports no rectangles leaves them empty.
	Boxes []OCRBox
}
