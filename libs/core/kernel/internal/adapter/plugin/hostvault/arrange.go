package hostvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
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
}

func (a Arranger) getChanges(v domain.Vault) NoteChanges {
	if a.WindowVault == nil {
		return a.Notes
	}
	if current, isShowing := a.WindowVault(); !isShowing || current.ID != v.ID {
		return a.Notes
	}
	changes := a.Window
	changes.Move.Drawing = a.notifyWindowIfCurrent(v, changes.Move.Drawing)
	changes.Rename.Move.Drawing = a.notifyWindowIfCurrent(v, changes.Rename.Move.Drawing)
	return changes
}

func (a Arranger) notifyWindowIfCurrent(v domain.Vault, draw note.TellMove) note.TellMove {
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
	return rename.Path, rename.By == query.ByFilename, rename.Moved, err
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
		return "", nil, domain.NewDetailError(fmt.Sprintf("%s is a folder; pass the paths of the notes to remove", path), edit.ErrNotANote, err)
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
		return nil, nil, fmt.Errorf("the checks of %s cannot run: %w", v.Name, domain.ErrUnavailable)
	}
	return a.Arranger.Checks(ctx, v, checks...)
}
func (a Vault) AddLinks(ctx context.Context, additions []plugin.LinkAddition) ([]plugin.LinkAdded, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	in := make([]note.LinkAddition, len(additions))
	for i, one := range additions {
		in[i] = note.LinkAddition{From: one.From, FileStat: one.Baseline, Link: one.Link}
	}
	added, err := a.Arranger.getChanges(v).AddLinks.Execute(ctx, v, in)
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
		a.notifyFileWritten()
	}
	return out, nil
}

func (a Vault) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := a.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := a.Arranger.getChanges(v).Links.Update(ctx, v, from, to, change, baseline)
	return stat, a.notifyAfterWrite(err)
}

func (a Vault) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := a.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := a.Arranger.getChanges(v).Links.RemoveTargetLinks(ctx, v, from, to, role, baseline)
	return stat, a.notifyAfterWrite(err)
}
