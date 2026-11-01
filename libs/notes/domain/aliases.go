package domain

import (
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain/name"
)

type (
	// From name subpackage
	SourceKind = name.SourceKind
	FileStat   = name.FileStat
)

const (
	// From name subpackage
	KindNote      = name.KindNote
	KindFile      = name.KindFile
	NoteExtension = name.NoteExtension
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = errors.New("unavailable")

	// ErrInvalidName is returned for a title no note can be given.
	ErrInvalidName = name.ErrInvalidName
)

// Filename returns the name a title is filed under and whether it survived the
// conversion unchanged, or an error for a title no note can be given.
func Filename(title string) (filename string, exact bool, err error) {
	return name.Filename(title)
}

// GetReducedFilename returns the filename a title reduces to, rejecting nothing: a
// title that reduces to no name at all returns empty.
func GetReducedFilename(title string) (filename string, exact bool) {
	return name.GetReducedFilename(title)
}

// ChooseMatch applies the priority a name resolves by: an exact path from the root,
// then a path relative to the note the link is written in, then a single match
// by name.
func ChooseMatch(from, target string, candidates []string) (bestMatch string, ambiguous bool) {
	return name.ChooseMatch(from, target, candidates)
}

// FoldName is the key under which two names are one name.
func FoldName(text string) string { return name.FoldName(text) }

// Basename is the name a note is filed under: the last segment of its path,
// without its extension.
func Basename(path string) string { return name.Basename(path) }

// LinkName is the name a link is written by: the last segment of the text
// between the brackets, without a note's extension.
func LinkName(target string) string { return name.LinkName(target) }
