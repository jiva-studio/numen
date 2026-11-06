package host

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// RenamedNote is a note after a rename: its path and title, whether its filename carries the title,
// and the move of its file, nil when the file kept its path.
type RenamedNote struct {
	Path        string
	Title       string
	IsFilenamed bool
	Moved       *domain.MoveResult
}

// NoteMove is the outcome of moving one note. Moved.From is always set; Moved.To and Moved.IsMoved
// only when the note moved. Err is why it did not: ErrNotExist, ErrNotANote, ErrAlreadyExists,
// ErrStale, domain.ErrInvalidName or another error.
type NoteMove struct {
	Moved domain.MoveResult
	Err   error
}

// NoteRemoval is the outcome of removing one note: where its file is in the trash and the notes whose
// links now resolve to nothing. Err is why it was not removed: ErrNotExist, ErrNotANote for a folder
// or a file that is not a note, or another error.
type NoteRemoval struct {
	Path     string
	Trashed  string
	Dangling []string
	Err      error
}

// VaultProblems is what the checks of a vault found, and the names of the checks that ran.
type VaultProblems struct {
	Ran   []string
	Found []domain.VaultProblem
}

// NoteArranger renames, moves and removes the notes of a vault, with the links that stop resolving
// repaired by the host, and runs the checks of the vault. The changes need vault:write and the checks
// vault:read.
type NoteArranger interface {
	// RenameNote gives the note another title and moves its file to the filename of the title.
	RenameNote(ctx context.Context, path, title string) (RenamedNote, error)
	// MoveNotes moves each note to the folder, keeping its filename. An empty folder is the vault root.
	// The answer has one entry per path, in order; a note that fails does not stop the others.
	MoveNotes(ctx context.Context, paths []string, folder string) ([]NoteMove, error)
	// RemoveNotes moves each note to the vault's trash. The answer has one entry per path, in order.
	RemoveNotes(ctx context.Context, paths []string) ([]NoteRemoval, error)
	// GetProblems runs the checks named (parse, frontmatter, ambiguous, dangling), or every check that
	// is not opt-in when none is named. dangling is opt-in.
	GetProblems(ctx context.Context, checks []string) (VaultProblems, error)
}
