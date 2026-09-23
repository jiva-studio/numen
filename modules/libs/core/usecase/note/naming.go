package note

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// NameSource is which of the two a note is shown by, and so which one naming it
// writes.
type NameSource string

const (
	ByFrontmatter NameSource = "frontmatter"
	ByFilename    NameSource = "filename"
)

// SyncTitleAndFilename is whether a note's title and its filename are kept as one name.
type SyncTitleAndFilename bool

// GetRenameEffects is what one rename brings into line: whether a new title
// moves the file, and whether a new filename is written into the note.
//
// A note its filename names carries its name nowhere else, so its file moves
// whatever this is set to and nothing is written into it.
func (s SyncTitleAndFilename) GetRenameEffects(by NameSource) (moves, writes bool) {
	if by == ByFilename {
		return true, false
	}
	return bool(s), bool(s)
}

// SyncSetting is asked as each rename is made, so a person who turns the
// setting is answered by the next rename.
//
// Nothing asked keeps the two one name, which is what an installation nobody
// has configured does.
type SyncSetting func() SyncTitleAndFilename

// GetSetting is what a rename reads.
func (ask SyncSetting) GetSetting() SyncTitleAndFilename {
	if ask == nil {
		return true
	}
	return ask()
}

// ErrUnnameable is a title no note can be given. Nothing is written.
var ErrUnnameable = domain.ErrUnnameable
