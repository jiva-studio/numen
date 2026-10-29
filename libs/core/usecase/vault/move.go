package vault

import (
	"context"
	"errors"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// NoteMover settles each note that travelled in a vault move.
type NoteMover interface {
	WriteFilenameAsTitle(ctx context.Context, v domain.Vault, path string) error
	RepairLinks(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (domain.MoveResult, error)
}

// Move files anything the vault holds somewhere else inside it: one file of any
// kind, or a folder with everything under it. Renaming is a move within one
// folder.
//
// Every note that travelled is filed where it now is, and a link that stopped
// reaching one is written again by its name. A note given a different name
// inside its own folder is called by that name, where a title and a filename
// are kept as one name.
type Move struct {
	writers vaultport.VaultWriters
	links   noteport.LinkQueries
	// queries is what the index holds about each file, and determines which sources
	// are under the path being moved.
	queries sourceport.SourceQueries
	// sources is where the index files each file. A folder and everything under
	// it are filed at their new paths in one write.
	sources sourceport.SourceRepository
	// notes reconciles each moved note: its open views, and the links written with
	// its old name.
	notes NoteMover
}

// NewMove is what files anything the vault holds somewhere else: the vault it
// is moved within, the links that pointed at what travelled, what the index
// holds about the files under the path and where it files them, and the one
// NoteMover every note that travelled settles through.
func NewMove(
	writers vaultport.VaultWriters,
	links noteport.LinkQueries,
	queries sourceport.SourceQueries,
	sources sourceport.SourceRepository,
	notes NoteMover,
) Move {
	return Move{writers: writers, links: links, queries: queries, sources: sources, notes: notes}
}

// WithSources returns a copy of Move with the given SourceRepository.
func (u Move) WithSources(sources sourceport.SourceRepository) Move {
	u.sources = sources
	return u
}

// Sources returns the SourceRepository Move files through.
func (u Move) Sources() sourceport.SourceRepository {
	return u.sources
}

// Execute moves the path and repairs what pointed at the notes under it.
//
// A folder is one rename, so what it holds arrives whole or stays where it was.
// A link repair that fails afterwards leaves that link broken and visible as a
// problem.
func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, error) {
	res := domain.MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}

	travelling, err := u.queries.GetSourcesUnder(ctx, v.ID, from)
	if err != nil {
		return res, err
	}

	// Queried before the move, because afterwards nothing points at the old paths.
	pointing, err := u.getBacklinksByPath(ctx, v, travelling)
	if err != nil {
		return res, err
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return res, err
	}
	// Only the core reports a path as holding nothing: the writer returns the same
	// error for a vault it cannot reach at all.
	if err := writer.Move(ctx, from, to); errors.Is(err, vaultport.ErrNotExist) {
		return res, edit.ErrNoNote
	} else if err != nil {
		return res, err
	}
	res.IsLanded = true

	// Notes and books are filed under their new paths in one write, and what was
	// derived from each of them moves with it.
	if err := u.sources.MoveSources(ctx, v.ID, from, to); err != nil {
		return res, err
	}

	// sent, and every view of the note must be notified wherever the note's own name
	// ended up.
	var called error
	if isRename(travelling, from, to) {
		called = u.notes.WriteFilenameAsTitle(ctx, v, to)
	}

	if err := u.rewriteNoteLinks(ctx, v, from, to, travelling, pointing, &res); err != nil {
		return res, err
	}
	return res, called
}

// getBacklinksByPath is the links that point at each note among the sources, by
// the note's path.
func (u Move) getBacklinksByPath(
	ctx context.Context, v domain.Vault, sources []domain.FileStat,
) (map[string][]domain.ResolvedLink, error) {
	linksBySource := make(map[string][]domain.ResolvedLink, len(sources))
	for _, source := range sources {
		if source.Kind != domain.KindNote {
			continue
		}
		links, err := u.links.GetBacklinks(ctx, v.ID, source.Path)
		if err != nil {
			return nil, err
		}
		linksBySource[source.Path] = links
	}
	return linksBySource, nil
}

// rewriteNoteLinks rewrites the links of each note among the sources at its new path and adds what
// it repaired and what it left dangling to res.
func (u Move) rewriteNoteLinks(
	ctx context.Context,
	v domain.Vault,
	from, to string,
	sources []domain.FileStat,
	pointing map[string][]domain.ResolvedLink,
	res *domain.MoveResult,
) error {
	for _, source := range sources {
		if source.Kind != domain.KindNote {
			continue
		}
		resolved, err := u.notes.RepairLinks(ctx, v, source.Path, getMovedPath(from, to, source.Path), pointing[source.Path])
		res.Repaired = append(res.Repaired, resolved.Repaired...)
		res.Dangling = append(res.Dangling, resolved.Dangling...)
		if err != nil {
			return err
		}
	}
	return nil
}

// isRename is whether this move is one note given a different name, in the
// folder and under the extension it already has. Travelling is everything the
// index files under from.
func isRename(travelling []domain.FileStat, from, to string) bool {
	if pathpkg.Dir(from) != pathpkg.Dir(to) || pathpkg.Ext(from) != pathpkg.Ext(to) {
		return false
	}
	return len(travelling) == 1 &&
		travelling[0].Path == from && travelling[0].Kind == domain.KindNote
}

// getMovedPath is where a path under from is once from is at to.
func getMovedPath(from, to, path string) string {
	return to + path[len(from):]
}
