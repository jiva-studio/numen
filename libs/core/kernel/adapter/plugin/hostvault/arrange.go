package hostvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// NoteChanges has the use cases that rename, move and remove the notes of a vault, repairing the
// links that stop resolving.
type NoteChanges struct {
	Rename   note.Rename
	Move     note.Move
	Remove   note.RemoveNote
	Links    note.EditLinks
	AddLinks note.AddLinks
}

// Arranger has the use cases that change the notes of a vault and that run the checks of a vault.
type Arranger struct {
	Notes       NoteChanges
	Window      NoteChanges
	WindowVault func() (domain.Vault, bool)
	Checks      func(ctx context.Context, v domain.Vault, named ...domain.Check) (ran []domain.Check, found []domain.VaultProblem, err error)
	Problems    *vaults.GetParseProblems

	session sessionAccess
}

func (a Arranger) getChanges(v domain.Vault) NoteChanges {
	if a.WindowVault == nil {
		return a.Notes
	}
	if current, isShowing := a.WindowVault(); !isShowing || current.ID != v.ID {
		return a.Notes
	}
	return a.Window
}

func (a Arranger) RenameNote(ctx context.Context, path, title string) (string, bool, *domain.MoveResult, error) {
	v, err := a.session.open()
	if err != nil {
		return "", false, nil, err
	}
	rename, err := a.getChanges(v).Rename.Execute(ctx, v, path, title)
	if rename.Moved != nil || err == nil {
		a.session.notifyFileWritten()
	}
	return rename.Path, rename.By == naming.ByFilename, rename.Moved, err
}

func (a Arranger) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	v, err := a.session.open()
	if err != nil {
		return domain.MoveResult{From: from, To: to}, err
	}
	move, err := a.getChanges(v).Move.Execute(ctx, v, from, to)
	if move.IsMoved {
		a.session.notifyFileWritten()
	}
	return move, err
}

func (a Arranger) RemoveNote(ctx context.Context, path string) (string, []string, error) {
	v, err := a.session.open()
	if err != nil {
		return "", nil, err
	}
	removal, err := a.getChanges(v).Remove.Execute(ctx, v, path, false)
	if errors.Is(err, note.ErrFolder) {
		return "", nil, domain.NewDetailError(fmt.Sprintf("%s is a folder; pass the paths of the notes to remove", path), edit.ErrNotANote, err)
	}
	if removal.Trashed != "" {
		a.session.notifyFileWritten()
	}
	return removal.Trashed, removal.Dangling, err
}

func (a Arranger) GetProblems(ctx context.Context, checks []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	v, err := a.session.open()
	if err != nil {
		return nil, nil, err
	}
	if a.Checks == nil {
		return nil, nil, fmt.Errorf("the checks of %s cannot run: %w", v.Name, domain.ErrUnavailable)
	}
	return a.Checks(ctx, v, checks...)
}
func (a Arranger) AddLinks(ctx context.Context, additions []plugin.LinkAddition) ([]plugin.LinkAdded, error) {
	v, err := a.session.open()
	if err != nil {
		return nil, err
	}
	in := make([]note.LinkAddition, len(additions))
	for i, one := range additions {
		in[i] = note.LinkAddition{From: one.From, FileStat: one.Baseline, Link: one.Link}
	}
	added, err := a.getChanges(v).AddLinks.Execute(ctx, v, in)
	if err != nil {
		return nil, err
	}
	out := make([]plugin.LinkAdded, len(added))
	isWritten := false
	for i, one := range added {
		out[i] = plugin.LinkAdded{Stat: one.FileStat, Err: one.Error}
		isWritten = isWritten || one.Error == nil || errors.Is(one.Error, edit.ErrIndexOutdated)
	}
	if isWritten {
		a.session.notifyFileWritten()
	}
	return out, nil
}

func (a Arranger) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := a.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := a.getChanges(v).Links.Update(ctx, v, from, to, change, baseline)
	return stat, a.session.notifyAfterWrite(err)
}

func (a Arranger) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := a.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := a.getChanges(v).Links.RemoveTargetLinks(ctx, v, from, to, role, baseline)
	return stat, a.session.notifyAfterWrite(err)
}

// GetParseProblems returns the notes of the vault that could not be read.
func (a Arranger) GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error) {
	v, err := a.session.open()
	if err != nil || a.Problems == nil {
		return nil, err
	}
	return a.Problems.Execute(ctx, v)
}
