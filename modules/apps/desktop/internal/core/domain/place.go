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
}
