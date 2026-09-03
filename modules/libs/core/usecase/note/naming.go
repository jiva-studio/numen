package note

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Naming is which of the two a note is shown by, and so which one naming it
// writes.
type Naming string

const (
	ByFrontmatter Naming = "frontmatter"
	ByFilename    Naming = "filename"
)

// SyncTitleAndFilename is whether a note's title and its filename are kept as one name.
type SyncTitleAndFilename bool

// Renaming is what one rename brings into line: whether a new title moves the
// file, and whether a new filename is written into the note.
//
// A note its filename names carries its name nowhere else, so its file moves
// whatever this is set to and nothing is written into it.
func (s SyncTitleAndFilename) Renaming(by Naming) (moves, writes bool) {
	if by == ByFilename {
		return true, false
	}
	return bool(s), bool(s)
}

// Syncing is asked as each rename is made, so a person who turns the setting is
// answered by the next rename.
//
// Nothing asked keeps the two one name, which is what an installation nobody
// has configured does.
type Syncing func() SyncTitleAndFilename

// Kept is what a rename reads.
func (ask Syncing) Kept() SyncTitleAndFilename {
	if ask == nil {
		return true
	}
	return ask()
}

// ErrUnnameable is a title no note can be given. Nothing is written.
var ErrUnnameable = errors.New("a note cannot be given this title")

// nameOf is the filename a title is filed under, and whether the filename is
// the whole of the title.
//
// Two titles are refused whatever the note: one that leaves no filename, and
// one carrying a line break. The title comes in trimmed.
func nameOf(title string) (name string, exact bool, err error) {
	switch name, exact = domain.Filename(title); {
	case name == "":
		return "", false, fmt.Errorf("%w: %q leaves nothing a file can be named after", ErrUnnameable, title)
	case strings.ContainsAny(title, "\n\r"):
		return "", false, fmt.Errorf("%w: %q is more than one line", ErrUnnameable, title)
	}
	return name, exact, nil
}
