package host

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// NoteRename is the outcome of renaming a note: its path and title, whether the title is stored in
// its filename, and the move of its file, nil when the file kept its path.
type NoteRename struct {
	Path        string
	Title       string
	IsFilenamed bool
	Move        *domain.MoveResult
}

// NoteMove is the outcome of moving one note. Move.From is always set; Move.To and Move.IsMoved
// only when the note moved. Err is the failure: ErrNotExist, ErrNotANote, ErrAlreadyExists,
// ErrStale, domain.ErrInvalidName or another error.
type NoteMove struct {
	Move domain.MoveResult
	Err  error
}

// NoteRemoval is the outcome of removing one note: TrashPath is the path in the trash, and
// BrokenLinks the notes whose links are now broken. Err is the failure: ErrNotExist, ErrNotANote
// for a folder or a file that is not a note, or another error.
type NoteRemoval struct {
	Path        string
	TrashPath   string
	BrokenLinks []string
	Err         error
}

// VaultProblems is the problems the checks of a vault found, and the names of the checks the result covers.
type VaultProblems struct {
	Checks []string
	Found  []domain.VaultProblem
}

// LinkAddition is one link to write into the note at From, which the caller read as Baseline.
type LinkAddition struct {
	From     string
	Baseline domain.FileStat
	Link     domain.Link
}

// LinkAdded is the outcome of one addition: the note after the write, or the error of a link not
// written. A stale baseline returns ErrStale.
type LinkAdded struct {
	Stat domain.FileStat
	Err  error
}
