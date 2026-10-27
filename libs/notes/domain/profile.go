package domain

import "regexp"

// NoteProfile says how the index keeps the notes of a type a plugin gave them. A type no plugin
// profiles is kept as an ordinary note is: its prose cut into chunks and every heading kept.
type NoteProfile struct {
	// HasChunks says whether the prose of the note is cut into chunks and searched by meaning. A
	// note without them is found by its title and its headings.
	HasChunks bool
	// HeadingLevels are the levels of heading the index keeps, counted from one. Nil keeps every
	// heading, and an empty list keeps none.
	HeadingLevels []int
	// IDHeadingLevels are the levels whose headings end in a block id — a space, a caret and a word —
	// that is no part of what the heading says and is kept out of the text the index holds.
	IDHeadingLevels []int
	// IDPattern matches a block id whole. Nil takes any word of letters, digits and hyphens.
	IDPattern *regexp.Regexp
}

// OrdinaryProfile is how an ordinary note is kept.
func OrdinaryProfile() NoteProfile { return NoteProfile{HasChunks: true} }
