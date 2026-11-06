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

// Arranger is the use cases that rename, move and remove the notes of a vault, repairing the links
// that stop resolving, and that run the checks of a vault.
type Arranger struct {
	Rename note.Rename
	Move   note.Move
	Remove note.RemoveNote
	// Checks runs the checks of a vault. Without it every call for the checks fails with
	// completionport.ErrUnavailable.
	Checks *vaults.GetProblems
}

func (a Vault) RenameNote(ctx context.Context, path, title string) (string, bool, *domain.MoveResult, error) {
	v, err := a.open()
	if err != nil {
		return "", false, nil, err
	}
	renamed, err := a.Arranger.Rename.Execute(ctx, v, path, title)
	if renamed.Moved != nil || err == nil {
		a.notifyFileWritten()
	}
	return renamed.Path, renamed.By == note.ByFilename, renamed.Moved, err
}

func (a Vault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	v, err := a.open()
	if err != nil {
		return domain.MoveResult{From: from, To: to}, err
	}
	moved, err := a.Arranger.Move.Execute(ctx, v, from, to)
	if moved.IsMoved {
		a.notifyFileWritten()
	}
	return moved, err
}

func (a Vault) RemoveNote(ctx context.Context, path string) (string, []string, error) {
	v, err := a.open()
	if err != nil {
		return "", nil, err
	}
	removed, err := a.Arranger.Remove.Execute(ctx, v, path, false)
	if errors.Is(err, note.ErrFolder) {
		return "", nil, fmt.Errorf("remove %s: %w", path, vaultport.ErrNotANote)
	}
	if removed.Trashed != "" {
		a.notifyFileWritten()
	}
	return removed.Trashed, removed.Dangling, err
}

func (a Vault) GetProblems(ctx context.Context, checks []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	v, err := a.open()
	if err != nil {
		return nil, nil, err
	}
	if a.Arranger.Checks == nil {
		return nil, nil, fmt.Errorf("run the checks of %s: %w", v.ID, completionport.ErrUnavailable)
	}
	problems, err := a.Arranger.Checks.Execute(ctx, v, checks...)
	if err != nil {
		return nil, nil, err
	}
	return problems.Ran, problems.Found, nil
}
