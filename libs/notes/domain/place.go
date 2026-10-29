package domain

// Place is a location in the vault: a source, by the path the vault files it
// under, and the spans of its text that are meant.
//
// The first span is the navigation target and the rest are shown where they
// fall. A note is a place with no spans, and so is a document requested at its
// beginning.
type Place struct {
	Path string
	// Ranges are the runs, over the text the source is read as.
	Ranges []ByteRange
}

// A ByteRange is a run of text, by where it begins and where it ends, counted in
// bytes of the text as the vault stores it. It is what the core measures with.
type ByteRange struct {
	Start int
	End   int
}

// Empty is a span naming no run at all.
func (s ByteRange) Empty() bool { return s.End <= s.Start }

// Len is how many bytes the span covers.
func (s ByteRange) Len() int { return s.End - s.Start }

// A UnitRange is a run of text counted the way a client counts text: in UTF-16
// way across.
type UnitRange struct {
	Start int
	End   int
}

// Empty is a span naming no run at all.
func (s UnitRange) Empty() bool { return s.End <= s.Start }

// Len is how many code units the span covers.
func (s UnitRange) Len() int { return s.End - s.Start }

// MaxHighlights is how many spans of one source are highlighted at once.
//
// Highlighting every span of a page conveys nothing about where to look, and the
// code choosing the places decides which of them matter.
const MaxHighlights = 8
