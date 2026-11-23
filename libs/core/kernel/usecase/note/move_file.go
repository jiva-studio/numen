package note

import (
	"context"
	"errors"
	pathpkg "path"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// MoveFile files anything the vault holds somewhere else inside it: one file of any
// kind, or a folder with everything under it. Renaming is a move within one folder.
// Every note that traveled is filed where it now is, and a link that stopped
// reaching one is written again by its name. A note given a different name
// inside its own folder is called by that name, where a title and a filename are one name.
type MoveFile struct {
	notes     Move
	queries   SourcesUnder
	admission vaultport.WriteAdmission
	follow    IndexFollow
}

// NewMoveFile is what files anything the vault holds somewhere else: the Move
// every note that traveled settles through, which also holds the vault the path
// is moved within, the links that pointed at what traveled and where the index
// files them, what the index holds about the files under the path, and the
// admission and the index follow of a write.
func NewMoveFile(notes Move, queries SourcesUnder, admission vaultport.WriteAdmission, follow IndexFollow) MoveFile {
	return MoveFile{notes: notes, queries: queries, admission: admission, follow: follow}
}

// Execute moves the file or folder from from to to, repairs what pointed at the
// notes under it, and reports whether the index did not follow.
//
// A folder is one rename, so its contents arrive whole or keep their position.
// A link repair that fails afterwards leaves that link broken and visible as a
// problem.
func (u MoveFile) Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, bool, error) {
	release, err := u.admission.Begin()
	if err != nil {
		return domain.MoveResult{}, false, err
	}
	defer release()
	moved, err := u.move(ctx, v, from, to)
	if isIndexOutdated(u.follow, err) {
		return moved, true, nil
	}
	return moved, false, err
}

func (u MoveFile) move(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, error) {
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

	mover, err := u.notes.writers.OpenFileMover(v)
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
	if err := u.notes.sources.MoveSources(ctx, v.ID, from, to); err != nil {
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

func (u MoveFile) getBacklinksByPath(
	ctx context.Context, v domain.Vault, sources []domain.FileStat,
) (map[string][]domain.ResolvedLink, error) {
	linksBySource := make(map[string][]domain.ResolvedLink, len(sources))
	for _, source := range sources {
		if source.Kind != domain.KindNote {
			continue
		}
		links, err := u.notes.links.ListBacklinks(ctx, v.ID, source.Path)
		if err != nil {
			return nil, err
		}
		linksBySource[source.Path] = links
	}
	return linksBySource, nil
}

func (u MoveFile) rewriteNoteLinks(
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
