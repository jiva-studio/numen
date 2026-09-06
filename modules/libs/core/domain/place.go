package domain

// Place is somewhere in the vault: a source, by the path the vault files it
// under, and the stretch of that source's text meant.
//
// A note is a place with no stretch, and so is a document asked for at its
// beginning.
type Place struct {
	Path string
	// Start and Length are the stretch, counted in bytes over the text the
	// source is read as. A length of zero names the source and no place inside
	// it.
	Start  int
	Length int
	// Stretches are the other stretches of the same source worth seeing. The
	// person is taken to the first stretch, and these are shown where they fall.
	Stretches []Stretch
}

// A Stretch is a run of a source's text, counted in bytes over the text the
// source is read as.
type Stretch struct {
	Start  int
	Length int
}

// MostHighlights is how many places of one source are highlighted at once, the
// place the person was taken to among them.
//
// A page with everything on it highlighted says nothing about where to look,
// and whoever is choosing the places chooses which of them matter.
const MostHighlights = 8
