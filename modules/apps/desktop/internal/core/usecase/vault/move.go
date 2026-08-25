package vault

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Move files anything the vault holds somewhere else inside it: one file of any
// kind, or a folder with everything under it. Renaming is a move within one
// folder.
//
// Every note that travelled is filed where it now is, and a link that stopped
// reaching one is written again by its name.
type Move struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Links   port.LinkQueries
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Notes is what settles each note that travelled: the index, whoever is
	// drawing it, and the links written by its old name.
	Notes note.Move
}

// Execute moves the path and repairs what pointed at the notes under it.
//
// A folder is one rename, so what it holds arrives whole or stays where it was.
// A link repair that fails afterwards leaves that link broken and visible as a
// problem.
func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (note.Moved, error) {
	res := note.Moved{From: from, To: to}
	if from == to {
		return res, nil
	}

	travelling, err := u.sources(ctx, v, from)
	if err != nil {
		return res, err
	}

	// Asked before the move, because afterwards nothing points at the old paths
	// and there is nothing left to ask about.
	pointing := make(map[string][]domain.ResolvedLink, len(travelling))
	for _, source := range travelling {
		if source.Kind != domain.KindNote {
			continue
		}
		links, err := u.Links.Backlinks(ctx, v.ID, source.Path)
		if err != nil {
			return res, err
		}
		pointing[source.Path] = links
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return res, err
	}
	// Only the core says a path holds nothing: the writer answers the same way
	// for a vault it cannot reach at all.
	if err := writer.Move(ctx, from, to); errors.Is(err, fs.ErrNotExist) {
		return res, note.ErrNoNote
	} else if err != nil {
		return res, err
	}
	res.Landed = true

	var others []string
	for _, source := range travelling {
		landed := relocated(from, to, source.Path)
		if source.Kind != domain.KindNote {
			// A book that moved is read again from where it is now.
			others = append(others, source.Path, landed)
			continue
		}
		settled, err := u.Notes.Settle(ctx, v, source.Path, landed, pointing[source.Path])
		res.Repaired = append(res.Repaired, settled.Repaired...)
		res.Retargeted = append(res.Retargeted, settled.Retargeted...)
		if err != nil {
			return res, err
		}
	}
	if len(others) > 0 {
		if err := u.index(ctx, v, others...); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (u Move) index(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.Index == nil {
		return nil
	}
	return u.Index(ctx, v, paths)
}

// sources is every source the vault holds at or under a path: the file itself,
// or everything under a folder.
func (u Move) sources(ctx context.Context, v domain.Vault, from string) ([]domain.FileRef, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return nil, err
	}
	var found []domain.FileRef
	if err := reader.Walk(ctx, func(ref domain.FileRef) error {
		if ref.Path == from || strings.HasPrefix(ref.Path, from+"/") {
			found = append(found, ref)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return found, nil
}

// relocated is where a path under from is once from is at to.
func relocated(from, to, path string) string {
	if path == from {
		return to
	}
	return to + path[len(from):]
}
