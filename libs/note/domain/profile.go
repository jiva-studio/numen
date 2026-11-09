package domain

import "regexp"

// NoteProfile defines how the index stores the notes of a type provided by a plugin. A type no plugin
// profiles is stored like an ordinary note: its prose is cut into chunks and every heading is kept.
type NoteProfile struct {
	HasChunks       bool
	HeadingLevels   []int
	IDHeadingLevels []int
	IDPattern       *regexp.Regexp
}

// DefaultProfile is how an ordinary note is kept.
func DefaultProfile() NoteProfile { return NoteProfile{HasChunks: true} }
