package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Backlog is how many events are held while they are being folded. Past it the
// vault is read from scratch, which is cheaper than working out what was
// missed.
const Backlog = 4096

// Watching is a vault being followed. Changes carries the paths that changed,
// folded, and Lost says the vault has to be read again.
type Watching struct {
	Changes <-chan []string
	Lost    <-chan struct{}
}

// Watch follows one vault and reports the notes that change in it.
//
// The watch is on the tree, not on any file: editors save by writing a
// temporary file and renaming it over the original, so the file a watch was
// placed on stops existing at the first save.
func Watch(ctx context.Context, v domain.Vault, opts Options) (*Watching, error) {
	reader, err := Open(v.Path, opts)
	if err != nil {
		return nil, err
	}

	raw := make(chan notify.EventInfo, Backlog)
	tree := filepath.Join(reader.Root(), "...")
	if err := notify.Watch(tree, raw, notify.All); err != nil {
		return nil, fmt.Errorf("watch %s: %w", v.Path, err)
	}

	changes := make(chan []string)
	lost := make(chan struct{}, 1)

	go func() {
		defer notify.Stop(raw)
		defer close(changes)
		fold(ctx, reader, opts, raw, changes, lost)
	}()

	return &Watching{Changes: changes, Lost: lost}, nil
}

// fold collects events for a window and reports each path once.
func fold(
	ctx context.Context,
	reader *VaultReader,
	opts Options,
	raw <-chan notify.EventInfo,
	changes chan<- []string,
	lost chan<- struct{},
) {
	pending := map[string]bool{}
	var window <-chan time.Time

	send := func() {
		if len(pending) == 0 {
			return
		}
		paths := make([]string, 0, len(pending))
		for path := range pending {
			paths = append(paths, path)
		}
		clear(pending)
		select {
		case changes <- paths:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, open := <-raw:
			if !open {
				return
			}
			// A full backlog means events were dropped. What was missed is
			// unknowable, so the vault is read again.
			if len(raw) == cap(raw) {
				clear(pending)
				window = nil
				select {
				case lost <- struct{}{}:
				default:
				}
				continue
			}
			paths, whole := reader.within(event.Path())
			if whole {
				// A folder that is gone takes notes with it, and their paths
				// are known to the index rather than to the disk.
				clear(pending)
				window = nil
				select {
				case lost <- struct{}{}:
				default:
				}
				continue
			}
			for _, path := range paths {
				pending[path] = true
			}
			// Measured from the first event of a batch, not the last. A vault
			// being written to continuously — a sync client, a checkout — never
			// stops long enough for a deadline that moves with it.
			if window == nil && len(pending) > 0 {
				window = time.After(opts.window())
			}

		case <-window:
			window = nil
			send()
		}
	}
}

// within turns an absolute path from the operating system into the notes of
// this vault that it concerns.
//
// A file is itself, when the vault holds it. A folder is everything under it:
// a folder arrives with its contents already in place — copied, restored,
// checked out — and where the system has no recursion of its own the watch on
// it is established after the fact.
//
// `whole` is set when the answer cannot be worked out from the disk: a path
// that is gone and is not a note was a folder, and what it held is a question
// for the index. A path outside the vault is that case too, which is what
// arrives when a watched folder is renamed away.
func (s *VaultReader) within(absolute string) (paths []string, whole bool) {
	rel, err := filepath.Rel(s.root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, true
	}
	path := filepath.ToSlash(rel)

	info, err := os.Stat(absolute)
	switch {
	case err == nil && info.IsDir():
		var found []string
		_ = filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if inside, _ := s.within(p); len(inside) == 1 {
				found = append(found, inside[0])
			}
			return nil
		})
		return found, false

	case err != nil && !s.holds(path):
		// Gone, and never a note by its name: a folder. Nothing on disk can
		// say what was under it.
		return nil, s.wasFolder(path)
	}

	if !s.holds(path) {
		return nil, false
	}
	return []string{path}, false
}

// wasFolder guesses whether a path that is gone was a folder.
//
// Nothing can answer it properly: the thing that would say is the thing that no
// longer exists. The guess is that a folder has no extension, which is what
// separates the two cases that arrive — a vault folder being renamed or moved
// away, and an editor's temporary file being renamed over its original.
//
// Guessing high costs a reading of the vault that was not needed. Guessing low
// leaves notes in the index that are no longer on disk, and nothing to correct
// it, so the guess leans high.
func (s *VaultReader) wasFolder(path string) bool {
	if path == "." || path == "" {
		return true
	}
	if pathpkg.Ext(path) != "" {
		return false
	}
	if s.ignored.MatchesPath(path + "/") {
		return false
	}
	for dir := pathpkg.Dir(path); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if pathpkg.Base(dir) == s.opts.serviceDir() || s.ignored.MatchesPath(dir+"/") {
			return false
		}
	}
	return pathpkg.Base(path) != s.opts.serviceDir()
}
