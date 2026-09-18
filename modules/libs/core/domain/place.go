package domain

// Place is somewhere in the vault: a source, by the path the vault files it
// under, and the runs of its text meant.
//
// The person is taken to the first span and the rest are shown where they fall.
// A note is a place with no spans, and so is a document asked for at its
// beginning.
type Place struct {
	Path string
	// Spans are the runs, over the text the source is read as.
	Spans []ByteSpan
}

// A ByteSpan is a run of text, by where it begins and where it ends, counted in
// bytes of the text as the vault stores it. It is what the core measures with.
type ByteSpan struct {
	From int
	To   int
}

// Empty is a span naming no run at all.
func (s ByteSpan) Empty() bool { return s.To <= s.From }

// Len is how many bytes the span covers.
func (s ByteSpan) Len() int { return s.To - s.From }

// A UnitSpan is a run of text counted the way a client counts text: in UTF-16
// code units. It is what a window is told, and markdown.CountUTF16 is the only
// way across.
type UnitSpan struct {
	From int
	To   int
}

// Empty is a span naming no run at all.
func (s UnitSpan) Empty() bool { return s.To <= s.From }

// Len is how many code units the span covers.
func (s UnitSpan) Len() int { return s.To - s.From }

// MostHighlights is how many spans of one source are highlighted at once.
//
// A page with everything on it highlighted says nothing about where to look,
// and whoever is choosing the places chooses which of them matter.
const MostHighlights = 8
