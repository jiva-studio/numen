package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strconv"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// TrashDir is where a removed note is kept. A name beginning with a dot is not
// a note, so a note put here leaves the index, the search and the
// plex by machinery that already exists, without being destroyed.
const TrashDir = ".trash"

// Remove takes a file or a folder out of the vault.
//
// It moves it into the vault's trash. Everything the index holds is rebuilt
// from the file, so losing the index costs a scan; the file is the one thing
// nothing rebuilds.
type Remove struct {
	writers vaultport.FileMovers
	links   noteport.LinkQueries
	queries sourceport.SourcesUnder
	index   edit.IndexUpdater
	gate    windowport.WriteGate
}

// NewRemove is what takes a note out of the vault: the vault it is moved
// within, the links that pointed at it, what the index holds about the files
// under it, and what updates the index for the path it left, and the gate a removal passes.
func NewRemove(
	writers vaultport.FileMovers, links noteport.LinkQueries, queries sourceport.SourcesUnder, index edit.IndexUpdater,
	gate windowport.WriteGate,
) Remove {
	return Remove{writers: writers, links: links, queries: queries, index: index, gate: gate}
}

type RemoveResult struct {
	Path     string
	Trashed  string
	Dangling []string
}

// Execute puts the file or the folder in the trash, with everything a folder
// holds. Destroy takes one note out of the world.
// It fails with window.ErrClosing once the window is closing.
func (u Remove) Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, error) {
	return writeUnder(u.gate, func() (RemoveResult, error) { return u.trash(ctx, v, path) })
}

func (u Remove) trash(ctx context.Context, v domain.Vault, path string) (RemoveResult, error) {
	res := RemoveResult{Path: path}

	went, err := u.queries.GetSourcesUnder(ctx, v.ID, path)
	if err != nil {
		return res, err
	}
	// Queried before the move, because afterwards nothing points at the old paths.
	pointing, err := u.getBacklinksOfNotes(ctx, v, went)
	if err != nil {
		return res, err
	}

	writer, err := u.writers.OpenFileMover(v)
	if err != nil {
		return res, err
	}
	target, err := moveToTrash(ctx, writer, path)
	if err != nil {
		return res, err
	}
	res.Trashed = target

	// The index is updated for what the vault held at each path, and with
	// the path itself where it held nothing.
	indexPaths := make([]string, 0, len(went))
	inside := make(map[string]bool, len(went))
	for _, source := range went {
		indexPaths = append(indexPaths, source.Path)
		inside[source.Path] = true
	}
	if len(indexPaths) == 0 {
		indexPaths = append(indexPaths, path)
	}
	// The file is in the trash from here on, so the index update result is reported
	// alongside the links that now resolve to nothing.
	indexErr := u.updateIndex(ctx, v, indexPaths...)
	for _, was := range pointing {
		// A note that wrote a link and went to the trash beside its target has
		// nothing left to reach from.
		if inside[was.From] {
			continue
		}
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, indexErr
}

func (u Remove) getBacklinksOfNotes(ctx context.Context, v domain.Vault, sources []domain.FileStat) ([]domain.ResolvedLink, error) {
	var pointing []domain.ResolvedLink
	for _, source := range sources {
		if source.Kind != domain.KindNote {
			continue
		}
		links, err := u.links.GetBacklinks(ctx, v.ID, source.Path)
		if err != nil {
			return nil, err
		}
		pointing = append(pointing, links...)
	}
	return pointing, nil
}

func moveToTrash(ctx context.Context, writer vaultport.FileMover, path string) (string, error) {
	target := pathpkg.Join(TrashDir, path)
	for attempt := 2; ; attempt++ {
		err := writer.Move(ctx, path, target)
		if err == nil {
			return target, nil
		}
		if !errors.Is(err, vault.ErrAlreadyExists) {
			return "", edit.MapMissingNote(err)
		}
		if attempt > 100 {
			return "", fmt.Errorf("%s cannot be moved to the trash; the trash already has 100 files of that name", path)
		}
		target = pathpkg.Join(TrashDir, withSuffix(path, "-"+strconv.Itoa(attempt)))
	}
}

// Destroy takes the file off the disk. Nothing brings it back.
func (u Remove) Destroy(ctx context.Context, v domain.Vault, path string) (RemoveResult, error) {
	res := RemoveResult{Path: path}

	pointing, err := u.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return res, err
	}
	writer, err := u.writers.OpenFileMover(v)
	if err != nil {
		return res, err
	}
	if err := writer.Remove(ctx, path); err != nil {
		return res, edit.MapMissingNote(err)
	}
	// The file is off the disk from here on, so the index update result is reported
	// alongside the links that now resolve to nothing.
	indexErr := u.updateIndex(ctx, v, path)
	for _, was := range pointing {
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, indexErr
}

func (u Remove) updateIndex(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.index == nil {
		return nil
	}
	// The files are written, so the index follows them even when the caller is canceled.
	return edit.WrapIndexOutdated(u.index(context.WithoutCancel(ctx), v, paths), paths...)
}

func withSuffix(path, suffix string) string {
	ext := pathpkg.Ext(path)
	return path[:len(path)-len(ext)] + suffix + ext
}
