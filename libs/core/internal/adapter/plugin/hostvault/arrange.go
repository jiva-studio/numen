package hostvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

// NoteChanges is the use cases that rename, move and remove the notes of a vault, repairing the
// links that stop resolving.
type NoteChanges struct {
	Rename note.Rename
	Move   note.Move
	Remove note.RemoveNote
}

// Arranger is the use cases that change the notes of a vault and that run the checks of a vault.
type Arranger struct {
	Notes NoteChanges
	// Window is the window's own NoteChanges, used for a call on the vault WindowVault names, so the
	// window follows a note a plugin renames or moves. Without WindowVault every call uses Notes.
	Window      NoteChanges
	WindowVault func() (domain.Vault, bool)
	// Checks runs the checks of a vault. Without it every call for the checks fails with
	// completionport.ErrUnavailable.
	Checks *vaults.GetProblems
}

// getChanges is the NoteChanges for a call on the vault.
func (a Arranger) getChanges(v domain.Vault) NoteChanges {
	if a.WindowVault == nil {
		return a.Notes
	}
	if current, isShowing := a.WindowVault(); !isShowing || current.ID != v.ID {
		return a.Notes
	}
	changes := a.Window
	changes.Move.Drawing = a.drawWhileCurrent(v, changes.Move.Drawing)
	changes.Rename.Move.Drawing = a.drawWhileCurrent(v, changes.Rename.Move.Drawing)
	return changes
}

// drawWhileCurrent tells the window a move only while the window still shows the vault the move is
// in: the window may change vault while a rename or a move of a plugin runs.
func (a Arranger) drawWhileCurrent(v domain.Vault, draw note.TellMove) note.TellMove {
	if draw == nil {
		return nil
	}
	return func(ctx context.Context, move domain.Move) error {
		if current, isShowing := a.WindowVault(); !isShowing || current.ID != v.ID {
			return nil
		}
		return draw(ctx, move)
	}
}

func (a Vault) RenameNote(ctx context.Context, path, title string) (string, bool, *domain.MoveResult, error) {
	v, err := a.open()
	if err != nil {
		return "", false, nil, err
	}
	rename, err := a.Arranger.getChanges(v).Rename.Execute(ctx, v, path, title)
	if rename.Moved != nil || err == nil {
		a.notifyFileWritten()
	}
	return rename.Path, rename.By == note.ByFilename, rename.Moved, err
}

func (a Vault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	v, err := a.open()
	if err != nil {
		return domain.MoveResult{From: from, To: to}, err
	}
	move, err := a.Arranger.getChanges(v).Move.Execute(ctx, v, from, to)
	if move.IsMoved {
		a.notifyFileWritten()
	}
	return move, err
}

func (a Vault) RemoveNote(ctx context.Context, path string) (string, []string, error) {
	v, err := a.open()
	if err != nil {
		return "", nil, err
	}
	removal, err := a.Arranger.getChanges(v).Remove.Execute(ctx, v, path, false)
	if errors.Is(err, note.ErrFolder) {
		return "", nil, fmt.Errorf("remove %s: %w", path, vaultport.ErrNotANote)
	}
	if removal.Trashed != "" {
		a.notifyFileWritten()
	}
	return removal.Trashed, removal.Dangling, err
}

func (a Vault) GetProblems(ctx context.Context, checks []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	v, err := a.open()
	if err != nil {
		return nil, nil, err
	}
	if a.Arranger.Checks == nil {
		return nil, nil, fmt.Errorf("run the checks of %s: %w", v.ID, completionport.ErrUnavailable)
	}
	report, err := a.Arranger.Checks.Execute(ctx, v, checks...)
	if err != nil {
		return nil, nil, err
	}
	return report.Ran, report.Found, nil
}
