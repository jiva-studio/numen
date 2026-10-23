package artifacts

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

// An OCRPageStart is a page of the artifact, at the offset its prose begins.
type OCRPageStart struct {
	Offset int
}

// An OCRPart is one of the parts a document divides into: where its heading begins
// in the prose, how far the heading runs, and how deep the part sits.
//
// A part carries a run of the prose, so a heading the scan read badly still
// says where the part begins.
type OCRPart struct {
	Start  int
	Length int
	// Depth is how far in the part sits, counted from zero. A document title
	// stands above the section titles within it.
	Depth int
}
