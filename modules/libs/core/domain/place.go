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
	// Also are the other stretches of the same source worth seeing. The person
	// is taken to the first stretch, and these are shown where they fall.
	Also []Stretch
}

// A Stretch is a run of a source's text, counted in bytes over the text the
// source is read as.
type Stretch struct {
	Start  int
	Length int
}

// MostLit is how many places of one source are lit at once, the place the
// person was taken to among them.
//
// A page with everything on it marked says nothing about where to look, and
// whoever is choosing the places chooses which of them matter.
const MostLit = 8
