// Package arrange renames, moves and removes notes, and lists the problems the vault checks find.
package arrange

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// UseCase arranges the notes of one vault.
type UseCase struct {
	arranger ports.NoteArranger
}

// New creates a UseCase over the arranger of one vault.
func New(arranger ports.NoteArranger) *UseCase {
	return &UseCase{arranger: arranger}
}

// Rename gives a note a new title and moves its file to the file name of the title.
func (u *UseCase) Rename(ctx context.Context, path, title string) (host.NoteRename, error) {
	if strings.TrimSpace(path) == "" {
		return host.NoteRename{}, errors.New("path is required")
	}
	return u.arranger.RenameNote(ctx, path, title)
}

// Move moves notes to a folder; a note that fails does not stop the others.
func (u *UseCase) Move(ctx context.Context, paths []string, folder string) ([]host.NoteMove, error) {
	if err := checkPaths(paths); err != nil {
		return nil, err
	}
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return nil, errors.New(`folder is empty; pass "/" for the vault root`)
	}
	return u.arranger.MoveNotes(ctx, paths, strings.Trim(folder, "/"))
}

// Remove moves notes to the trash of the vault; a note that fails does not stop the others.
func (u *UseCase) Remove(ctx context.Context, paths []string) ([]host.NoteRemoval, error) {
	if err := checkPaths(paths); err != nil {
		return nil, err
	}
	return u.arranger.RemoveNotes(ctx, paths)
}

// ListProblems runs the checks named, or the default checks, and lists what they find.
func (u *UseCase) ListProblems(ctx context.Context, checks []string) (host.VaultProblems, error) {
	return u.arranger.GetProblems(ctx, checks)
}

func checkPaths(paths []string) error {
	if len(paths) == 0 {
		return errors.New("paths is required")
	}
	return nil
}
