package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Move files a note somewhere else, which is also how it is renamed: a note is
// named by the file it is in, so the two are one operation.
//
// The bytes do not change. A note that carried no identifier still carries
// none afterwards, because moving is not editing.
type Move struct {
	readers vaultport.VaultReaders
	writers MoveWriting
	links   noteport.LinkQueries
	names   edit.NameQueries
	sources sourceport.SourceRepository
	index   edit.IndexUpdater
	now     machineport.Clock
	drawing TellMove
	sync    naming.SyncSetting
}

// MoveDeps is what NewMove takes: the vault readers and writers, the queries that find the links to a
// note and the names that resolve to it, the index and its updater, and the clock. Drawing receives each
// move when set, and Sync reports whether a title and a filename are kept as one name.
type MoveDeps struct {
	Readers vaultport.VaultReaders
	Writers MoveWriting
	Links   noteport.LinkQueries
	Names   edit.NameQueries
	Sources sourceport.SourceRepository
	Index   edit.IndexUpdater
	Now     machineport.Clock
	Drawing TellMove
	Sync    naming.SyncSetting
}

func NewMove(d MoveDeps) Move {
	return Move{
		readers: d.Readers, writers: d.Writers, links: d.Links,
		names: d.Names, sources: d.Sources, index: d.Index, now: d.Now,
		drawing: d.Drawing, sync: d.Sync,
	}
}

func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, error) {
	res := domain.MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}
	if err := domain.CheckPath(to); err != nil {
		return res, err
	}

	// Backlinks are read before the move; afterwards nothing points at the old
	// path.
	pointing, err := u.links.GetBacklinks(ctx, v.ID, from)
	if err != nil {
		return res, err
	}

	mover, err := u.writers.OpenFileMover(v)
	if err != nil {
		return res, err
	}
	if err := mover.Move(ctx, from, to); err != nil {
		if errors.Is(err, vault.ErrAlreadyExists) {
			return res, domain.NewDetailError(fmt.Sprintf("%s already exists; %s is not moved", to, from), err)
		}
		return res, edit.MapMissingNote(err)
	}
	res.IsMoved = true
	if err := u.moveInIndex(ctx, v, from, to); err != nil {
		return res, err
	}
	return u.RepairLinks(ctx, v, from, to, pointing)
}

func (u Move) moveInIndex(ctx context.Context, v domain.Vault, from, to string) error {
	return u.sources.MoveSources(ctx, v.ID, from, to)
}

// RepairLinks completes a move after the file is at its new path and the index
// is in sync: it notifies the views of the note and rewrites the links that
// used the old name.
//
// pointing holds the links that resolved to the note before the move.
func (u Move) RepairLinks(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (domain.MoveResult, error) {
	res := domain.MoveResult{From: from, To: to, IsMoved: true}

	// The file is at its new path and the index is in sync. A view still showing
	// the old name refers to a path with no file.
	if u.drawing != nil {
		if err := u.drawing(ctx, v, domain.Move{From: from, To: to}); err != nil {
			return res, fmt.Errorf("the window cannot show the move of %s to %s: %w", from, to, err)
		}
	}

	// What a repaired link is pointed at: the name the note is filed under, or
	// its path where that name is shared with another note.
	var address string
	if len(pointing) > 0 {
		var err error
		if address, err = u.getAddress(ctx, v, to); err != nil {
			return res, err
		}
	}
	for _, was := range pointing {
		if err := u.rewriteLink(ctx, v, address, was, &res); err != nil {
			return res, err
		}
	}
	if len(res.Repaired) > 0 {
		if err := u.updateIndex(ctx, v, res.Repaired...); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (u Move) rewriteLink(
	ctx context.Context, v domain.Vault, address string, was domain.ResolvedLink, res *domain.MoveResult,
) error {
	lands, still, err := u.getLandingPath(ctx, v, was)
	if err != nil {
		return err
	}
	if !still {
		// The note that wrote it has changed since, and the link is not in
		// it any more. There is nothing to repair and nothing to report.
		return nil
	}
	// A link that resolves to the moved note followed it. A link that resolves to
	// another note of the same name is not broken; which note a name denotes is the
	// user's decision.
	if lands != "" {
		return nil
	}
	remaining, err := u.repair(ctx, v, was.From, was.Target, address)
	switch {
	case err != nil:
		return err
	case remaining == linkRepaired:
		res.Repaired = append(res.Repaired, was.From)
	case remaining == linkDangling:
		res.Dangling = append(res.Dangling, was.From)
	}
	return nil
}

func (u Move) getAddress(ctx context.Context, v domain.Vault, path string) (string, error) {
	to, err := edit.GetAddress(ctx, u.names, v.ID, path)
	if errors.Is(err, edit.ErrUnaddressable) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return to.Value, nil
}

func (u Move) getLandingPath(ctx context.Context, v domain.Vault, was domain.ResolvedLink) (lands string, exists bool, err error) {
	links, err := u.links.GetLinks(ctx, v.ID, was.From)
	if err != nil {
		return "", false, err
	}
	for _, l := range links {
		if l.Target == was.Target {
			return l.To, true, nil
		}
	}
	return "", false, nil
}

// linkState is what became of a link that stopped resolving.
type linkState int

const (
	// linkUntouched is a link nothing was written for.
	linkUntouched linkState = iota
	// linkRepaired is a link written again.
	linkRepaired
	// linkDangling is a link left reaching nothing in a note that could not be
	// written.
	linkDangling
)

func (u Move) repair(
	ctx context.Context, v domain.Vault, in string, address domain.Address, destination string,
) (linkState, error) {
	release, err := u.writers.Lock(ctx, v)
	if err != nil {
		return linkUntouched, err
	}
	defer release()

	reader, err := u.readers.Open(v)
	if err != nil {
		return linkUntouched, err
	}
	raw, err := reader.Read(ctx, in)
	if errors.Is(err, edit.ErrNotExist) {
		// The note that wrote the link went between the backlinks being read
		// and this. Its link went with it.
		return linkUntouched, nil
	}
	if err != nil {
		return linkUntouched, fmt.Errorf("%s cannot be read: %w", in, err)
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		// A note whose frontmatter cannot be read is never written, and the
		// link it holds is left reaching nothing.
		return linkDangling, nil
	}
	inBlock, err := doc.RedirectLinks(address, destination)
	if err != nil {
		return linkUntouched, err
	}
	inProse, err := doc.RedirectProse(address, destination)
	if err != nil {
		return linkUntouched, err
	}
	moved := inBlock + inProse
	if moved == 0 {
		return linkUntouched, nil
	}

	writer, err := u.writers.OpenNoteWriter(v)
	if err != nil {
		return linkUntouched, err
	}
	if _, err := writer.Write(ctx, in, doc.Bytes(), domain.FileStat{}); err != nil {
		return linkUntouched, err
	}
	return linkRepaired, nil
}

func (u Move) updateIndex(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.index == nil {
		return nil
	}
	// The files are written, so the index follows them even when the caller is canceled.
	return edit.WrapIndexOutdated(u.index(context.WithoutCancel(ctx), v, paths), paths...)
}

// GetPathUnder is where a note lands when it is filed under a folder, keeping
// its name.
func GetPathUnder(folder, path string) string {
	return pathpkg.Join(folder, pathpkg.Base(path))
}
