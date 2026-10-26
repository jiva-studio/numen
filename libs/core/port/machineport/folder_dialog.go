package machineport

import (
	"context"
	"errors"
)

// ErrChoosing is a picker asked for while one is already in front of the
// person. One is up at a time: a person answers one at a time.
var ErrChoosing = errors.New("a folder picker is already open")

// ErrNoFolderDialog is a machine that cannot put a folder picker up. A folder is
// named to this application some other way, and a vault is added by its path.
var ErrNoFolderDialog = errors.New("this machine cannot put a folder picker up")

// FolderDialog is the person picking a folder on this machine.
//
// Only an application with a window can satisfy it; where there is none,
// nothing asks.
type FolderDialog interface {
	// ChooseFolder puts the machine's own picker in front of the person. It answers
	// with the folder they chose, and with false where they chose none.
	//
	// The picker is titled title and opens at startingAt; either one empty
	// takes what this machine chooses.
	//
	// It answers ErrChoosing where a picker is already up, and
	// ErrNoFolderDialog where this machine cannot put one up at all.
	ChooseFolder(ctx context.Context, title, startingAt string) (string, bool, error)
}

// PathDialog is the person picking a file or a folder on this machine.
type PathDialog interface {
	FolderDialog
	// ChooseFile puts the machine's own file picker in front of the person, showing the files whose
	// extension is one of accept, or every file where accept is empty. It answers as ChooseFolder does.
	ChooseFile(ctx context.Context, title, startingAt string, accept []string) (string, bool, error)
}
