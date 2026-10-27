package domain

// A Rect is a place on a page, in fractions of it.
type Rect struct {
	MinX, MinY, MaxX, MaxY float32
}

// A HighlightBox is one run of prose and where it was read: the page it is on,
// the run of bytes in the text, and the rectangle it covers.
type HighlightBox struct {
	Page int
	ByteRange
	Rect
}
