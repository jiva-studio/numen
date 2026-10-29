package domain

import "regexp"

// NoteProfile defines how the index stores the notes of a type provided by a plugin. A type no plugin
// profiles is stored like an ordinary note: its prose is cut into chunks and every heading is kept.
type NoteProfile struct {
	// HasChunks reports whether the prose of the note is cut into chunks and searched by meaning. A
	// note without chunks is found by its title and its headings.
	HasChunks bool
	// HeadingLevels are the levels of heading the index keeps, counted from one. Nil keeps every
	// heading, and an empty list keeps none.
	HeadingLevels []int
	// IDHeadingLevels are the levels whose headings end in a block id (a space, a caret and a word),
	// which is not part of the heading text and is excluded from the text the index holds.
	IDHeadingLevels []int
	// IDPattern matches a block id whole. Nil takes any word of letters, digits and hyphens.
	IDPattern *regexp.Regexp
}

// DefaultProfile is how an ordinary note is kept.
func DefaultProfile() NoteProfile { return NoteProfile{HasChunks: true} }
