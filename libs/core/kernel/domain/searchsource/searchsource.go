// Package searchsource holds what the core knows of a search source: the recipe and the width of the
// vectors it keeps in the index, the size of the chunks it reads, and what it is handed and answers.
package searchsource

import (
	"fmt"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Description is what a search source tells the core about the data it keeps. SourceRecipe is opaque to the
// core: two vectors made under different recipes cannot be compared. ChunkTokens and MinTextLength are not
// part of the identity: a vector made for a text is the same whatever the chunk size, and the shortest text
// handed decides only which chunks are handed.
type Description struct {
	Name          string
	SourceRecipe  string
	Width         int
	ChunkTokens   int
	MinTextLength int
}

// String is the name and the width as one value, for a column that holds it.
func (d Description) String() string {
	return d.Name + "@" + strconv.Itoa(d.Width)
}

// Recipe is the key every vector of the source is kept under, and every floor of a vault: the recipe of
// the source and the way the index stores a vector. A changed scale gives every stored byte a new meaning.
func (d Description) Recipe() string {
	return fmt.Sprintf("%s|%s@%g", d.SourceRecipe, vector.QuantizedInt8, vector.Int8Scale)
}

// Place is where the vectors of a source are kept: the vault and the recipe they were made under.
type Place struct {
	VaultID domain.VaultID
	Recipe  string
}

// Text is one text the core hands a source to index: the chunk it was cut as, the path of its source and the
// text version it was cut from, empty for a note.
type Text struct {
	ChunkID   domain.ChunkID
	Path      string
	VersionID string
	Text      string
}

// Query is what a search asks a source. A Floor of zero applies the floor of the vault.
type Query struct {
	Text       string
	Kinds      []domain.SourceKind
	PathPrefix string
	Limit      int
	Floor      float64
}

// Match is a chunk that answers a query and how well, the higher the better.
type Match struct {
	ChunkID domain.ChunkID
	Score   float64
}

// Counts is how many chunks of a vault can be handed to a source, and how many have a vector in its place.
type Counts struct {
	Chunks  int64
	Indexed int64
}
