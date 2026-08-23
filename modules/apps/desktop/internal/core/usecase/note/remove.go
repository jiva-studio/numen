package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strconv"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// TrashDir is where a removed note is kept. A name beginning with a dot is not
// a note, so a note put here leaves the index, the search and the
// plex by machinery that already exists, without being destroyed.
const TrashDir = ".trash"

// Remove takes a note out of the vault.
//
// It moves the file into the vault's trash. Everything the index knows is
// rebuilt from the file, so losing the index costs a scan; the file is the one
// thing nothing rebuilds.
type Remove struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Links   port.LinkQueries
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
}

// Removed says what happened to one note and what it leaves behind.
type Removed struct {
	Path string
	// Trashed is where the note now sits, empty when it was destroyed.
	Trashed string
	// Dangling is the notes whose links pointed here and now reach nothing.
	// They are reported and not repaired: the link is not wrong, its target is
	// gone, and only the person knows what they meant.
	Dangling []string
}

// Execute puts the note in the trash. Destroy takes it out of the world.
func (u Remove) Execute(ctx context.Context, v domain.Vault, path string) (Removed, error) {
	res := Removed{Path: path}

	pointing, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return res, err
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
			return res, err
		}
		if attempt > 100 {
			return res, fmt.Errorf("remove %s: the trash already holds it", path)
		}
		target = pathpkg.Join(TrashDir, withSuffix(path, "-"+strconv.Itoa(attempt)))
	}
	res.Trashed = target

	if err := u.index(ctx, v, path); err != nil {
		return res, err
	}
	for _, was := range pointing {
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, nil
}

// Destroy takes the file off the disk. Nothing brings it back.
func (u Remove) Destroy(ctx context.Context, v domain.Vault, path string) (Removed, error) {
	res := Removed{Path: path}

	pointing, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return res, err
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return res, err
	}
	if err := writer.Remove(ctx, path); err != nil {
		return res, err
	}
	if err := u.index(ctx, v, path); err != nil {
		return res, err
	}
	for _, was := range pointing {
		res.Dangling = append(res.Dangling, was.From)
	}
	return res, nil
}

func (u Remove) index(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.Index == nil {
		return nil
	}
	return u.Index(ctx, v, paths)
}

// withSuffix puts something before the extension: `note.md` and `-2` make
// `note-2.md`, which reads as a copy.
func withSuffix(path, suffix string) string {
	ext := pathpkg.Ext(path)
	return path[:len(path)-len(ext)] + suffix + ext
}
