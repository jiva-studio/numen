package vault

import (
	"context"
	"errors"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	BeginWrite() bool
	FinishWrite()
}

// IndexFollow reports whether the index followed a write.
type IndexFollow interface {
	IsIndexOutdated(err error) bool
}

// MoveFile moves a file or a folder in the vault as a write of the window.
type MoveFile struct {
	move   *Move
	gate   WriteGate
	follow IndexFollow
}

// NewMoveFile returns the use case over the moving of a file or a folder.
func NewMoveFile(move *Move, gate WriteGate, follow IndexFollow) MoveFile {
	return MoveFile{move: move, gate: gate, follow: follow}
}

// Execute moves the file or folder from from to to and reports whether the index did not follow. It
// fails with window.ErrClosing once the window is closing.
func (u MoveFile) Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, bool, error) {
	if !u.gate.BeginWrite() {
		return domain.MoveResult{}, false, windowdomain.ErrClosing
	}
	defer u.gate.FinishWrite()
	moved, err := u.move.Execute(ctx, v, from, to)
	return moved, u.follow.IsIndexOutdated(err), err
}

// NoteMover settles each note that traveled in a vault move.
type NoteMover interface {
	WriteFilenameAsTitle(ctx context.Context, v domain.Vault, path string) error
	RepairLinks(ctx context.Context, v domain.Vault, from, to string, pointing []domain.ResolvedLink) (domain.MoveResult, error)
}

// Move files anything the vault holds somewhere else inside it: one file of any
// kind, or a folder with everything under it. Renaming is a move within one folder.
// Every note that traveled is filed where it now is, and a link that stopped
// reaching one is written again by its name. A note given a different name
// inside its own folder is called by that name, where a title and a filename are one name.
type Move struct {
	movers  vaultport.FileMovers
	links   noteport.LinkQueries
	queries sourceport.SourcesUnder
	sources sourceport.SourceRepository
	notes   NoteMover
}

// NewMove is what files anything the vault holds somewhere else: the vault it
// is moved within, the links that pointed at what traveled, what the index
// holds about the files under the path and where it files them, and the one
// NoteMover every note that traveled settles through.
func NewMove(
	movers vaultport.FileMovers,
	links noteport.LinkQueries,
	queries sourceport.SourcesUnder,
	sources sourceport.SourceRepository,
	notes NoteMover,
) Move {
	return Move{movers: movers, links: links, queries: queries, sources: sources, notes: notes}
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
// A folder is one rename, so its contents arrive whole or keep their position.
// A link repair that fails afterwards leaves that link broken and visible as a
// problem.
func (u Move) Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, error) {
	res := domain.MoveResult{From: from, To: to}
	if from == to {
		return res, nil
	}

	traveling, err := u.queries.GetSourcesUnder(ctx, v.ID, from)
	if err != nil {
		return res, err
	}

	// Queried before the move, because afterwards nothing points at the old paths.
	pointing, err := u.getBacklinksByPath(ctx, v, traveling)
	if err != nil {
		return res, err
	}

	mover, err := u.movers.OpenFileMover(v)
	if err != nil {
		return res, err
	}
	// Only the core reports a path as holding nothing: the writer returns the same
	// error for a vault it cannot reach at all.
	if err := mover.Move(ctx, from, to); errors.Is(err, edit.ErrNotExist) {
		return res, edit.ErrNoNote
	} else if err != nil {
		return res, err
	}
	res.IsMoved = true

	// Notes and books are filed under their new paths in one write, and what was
	// derived from each of them moves with it.
	if err := u.sources.MoveSources(ctx, v.ID, from, to); err != nil {
		return res, err
	}

	// sent, and every view of the note must be notified wherever the note's own name
	// ended up.
	var called error
	if isRename(traveling, from, to) {
		called = u.notes.WriteFilenameAsTitle(ctx, v, to)
	}

	rewritten, err := u.rewriteNoteLinks(ctx, v, from, to, traveling, pointing)
	res.Repaired = append(res.Repaired, rewritten.Repaired...)
	res.Dangling = append(res.Dangling, rewritten.Dangling...)
	if err != nil {
		return res, err
	}
	return res, called
}

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

func (u Move) rewriteNoteLinks(
	ctx context.Context,
	v domain.Vault,
	from, to string,
	sources []domain.FileStat,
	pointing map[string][]domain.ResolvedLink,
) (domain.MoveResult, error) {
	var res domain.MoveResult
	for _, source := range sources {
		if source.Kind != domain.KindNote {
			continue
		}
		resolved, err := u.notes.RepairLinks(ctx, v, source.Path, getMovedPath(from, to, source.Path), pointing[source.Path])
		res.Repaired = append(res.Repaired, resolved.Repaired...)
		res.Dangling = append(res.Dangling, resolved.Dangling...)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func isRename(traveling []domain.FileStat, from, to string) bool {
	if pathpkg.Dir(from) != pathpkg.Dir(to) || pathpkg.Ext(from) != pathpkg.Ext(to) {
		return false
	}
	return len(traveling) == 1 &&
		traveling[0].Path == from && traveling[0].Kind == domain.KindNote
}

func getMovedPath(from, to, path string) string {
	return to + path[len(from):]
}
