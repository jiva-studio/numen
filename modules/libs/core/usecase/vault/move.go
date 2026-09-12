package vault

import (
	"context"
	"errors"
	"io/fs"
	pathpkg "path"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Move files anything the vault holds somewhere else inside it: one file of any
// kind, or a folder with everything under it. Renaming is a move within one
// folder.
//
// Every note that travelled is filed where it now is, and a link that stopped
// reaching one is written again by its name. A note given a different name
// inside its own folder is called by that name, where a title and a filename
// are kept as one name.
type Move struct {
	Writers port.VaultWriters
	Links   port.LinkQueries
	// Queries is what the index holds about each file, and is what says which
	// sources sit under the path being moved.
	Queries port.SourceQueries
	// Sources is where the index files each file. A folder and everything under
	// it are filed at their new paths in one write.
	Sources port.SourceRepository
	// Notes is what settles each note that travelled: whoever is drawing it, and
	// the links written by its old name.
	Notes note.Move
}

// NewMove is what files anything the vault holds somewhere else: the vault it
// is moved within, the links that pointed at what travelled, what the index
// holds about the files under the path and where it files them, and the one
// note.Move every note that travelled settles through.
func NewMove(
	writers port.VaultWriters,
	links port.LinkQueries,
	queries port.SourceQueries,
	sources port.SourceRepository,
	notes note.Move,
) Move {
	return Move{Writers: writers, Links: links, Queries: queries, Sources: sources, Notes: notes}
}

// Execute moves the path and repairs what pointed at the notes under it.
//
// A folder is one rename, so what it holds arrives whole or stays where it was.
// A link repair that fails afterwards leaves that link broken and visible as a
// problem.
func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (note.MoveResult, error) {
	res := note.MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}

	travelling, err := u.Queries.Under(ctx, v.ID, from)
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

	// Notes and books alike are filed under their new paths in one write, and
	// what was derived from each of them travels with it.
	if err := u.Sources.MoveSources(ctx, v.ID, from, to); err != nil {
		return res, err
	}

	// A refusal here is carried past the settling: the file is where it was
	// sent, and whoever is drawing the note has to be told wherever the note's
	// own name ended up.
	var called error
	if isRename(travelling, from, to) {
		called = u.Notes.Called(ctx, v, to)
	}

	for _, source := range travelling {
		if source.Kind != domain.KindNote {
			continue
		}
		settled, err := u.Notes.Settle(ctx, v, source.Path, getMovedPath(from, to, source.Path), pointing[source.Path])
		res.Repaired = append(res.Repaired, settled.Repaired...)
		res.Dangling = append(res.Dangling, settled.Dangling...)
		if err != nil {
			return res, err
		}
	}
	return res, called
}

// isRename is whether this move is one note given a different name, in the
// folder and under the extension it already has. Travelling is everything the
// index files under from.
func isRename(travelling []domain.Fingerprint, from, to string) bool {
	if pathpkg.Dir(from) != pathpkg.Dir(to) || pathpkg.Ext(from) != pathpkg.Ext(to) {
		return false
	}
	if domain.Basename(from) == domain.Basename(to) {
		return false
	}
	return len(travelling) == 1 &&
		travelling[0].Path == from && travelling[0].Kind == domain.KindNote
}

// getMovedPath is where a path under from is once from is at to.
func getMovedPath(from, to, path string) string {
	if path == from {
		return to
	}
	return to + path[len(from):]
}
