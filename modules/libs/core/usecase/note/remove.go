package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strconv"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// TrashDir is where a removed note is kept. A name beginning with a dot is not
// a note, so a note put here leaves the index, the search and the
// plex by machinery that already exists, without being destroyed.
const TrashDir = ".trash"

// Remove takes a file or a folder out of the vault.
//
// It moves it into the vault's trash. Everything the index knows is rebuilt
// from the file, so losing the index costs a scan; the file is the one thing
// nothing rebuilds.
type Remove struct {
	Writers port.VaultWriters
	Links   port.LinkQueries
	// Queries is what the index holds about each file, and is what says which
	// sources sit under the path being removed.
	Queries port.SourceQueries
	Index   Levels
}

// NewRemove is what takes a note out of the vault: the vault it is moved
// within, the links that pointed at it, what the index holds about the files
// under it, and what brings the path it left level.
func NewRemove(
	writers port.VaultWriters, links port.LinkQueries, queries port.SourceQueries, index Levels,
) Remove {
	return Remove{Writers: writers, Links: links, Queries: queries, Index: index}
}

// RemoveResult says what happened to what was removed and what it leaves behind.
type RemoveResult struct {
	Path string
	// Trashed is where it now sits, empty when it was destroyed.
	Trashed string
	// Dangling is the notes whose links pointed at what went and now reach
	// nothing. They are reported and not repaired: the link is not wrong, its
	// target is gone, and only the person knows what they meant.
	Dangling []string
}

// Execute puts the file or the folder in the trash, with everything a folder
// holds. Destroy takes one note out of the world.
func (u Remove) Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, error) {
	res := RemoveResult{Path: path}

	went, err := u.Queries.GetSourcesUnder(ctx, v.ID, path)
	if err != nil {
		return res, err
	}
	// Asked before the move, because afterwards nothing points at the old paths
	// and there is nothing left to ask about.
	var pointing []domain.ResolvedLink
	for _, source := range went {
		if source.Kind != domain.KindNote {
			continue
		}
		links, err := u.Links.Backlinks(ctx, v.ID, source.Path)
		if err != nil {
			return res, err
		}
		pointing = append(pointing, links...)
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return res, err
	}

	// Keeping the path underneath the trash means two notes of the same name
	// do not land on each other, and that putting one back is obvious.
	target := pathpkg.Join(TrashDir, path)
	for attempt := 2; ; attempt++ {
		err := writer.Move(ctx, path, target)
		if err == nil {
			break
		}
		if !errors.Is(err, port.ErrOccupied) {
			return res, mapMissingNote(err)
		}
		if attempt > 100 {
			return res, fmt.Errorf("remove %s: the trash already holds it", path)
		}
		target = pathpkg.Join(TrashDir, withSuffix(path, "-"+strconv.Itoa(attempt)))
	}
	res.Trashed = target

	// The index is brought level with what the vault held at each path, and with
	// the path itself where it held nothing.
	level := make([]string, 0, len(went))
	inside := make(map[string]bool, len(went))
	for _, source := range went {
		level = append(level, source.Path)
		inside[source.Path] = true
	}
	if len(level) == 0 {
		level = append(level, path)
	}
	// The file is in the trash from here on, so what the levelling came to
	// stands beside the links that now reach nothing.
	levelled := u.index(ctx, v, level...)
	for _, was := range pointing {
		// A note that wrote a link and went to the trash beside its target has
		// nothing left to reach from.
		if inside[was.From] {
			continue
		}
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, levelled
}

// Destroy takes the file off the disk. Nothing brings it back.
func (u Remove) Destroy(ctx context.Context, v domain.Vault, path string) (RemoveResult, error) {
	res := RemoveResult{Path: path}

	pointing, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return res, err
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return res, err
	}
	if err := writer.Remove(ctx, path); err != nil {
		return res, mapMissingNote(err)
	}
	// The file is off the disk from here on, so what the levelling came to
	// stands beside the links that now reach nothing.
	levelled := u.index(ctx, v, path)
	for _, was := range pointing {
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, levelled
}

func (u Remove) index(ctx context.Context, v domain.Vault, paths ...string) error {
	return WrapUnlevelled(u.Index(ctx, v, paths), paths...)
}

// withSuffix puts something before the extension: `note.md` and `-2` make
// `note-2.md`, which reads as a copy.
func withSuffix(path, suffix string) string {
	ext := pathpkg.Ext(path)
	return path[:len(path)-len(ext)] + suffix + ext
}
