package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
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

	// The shape is taken before the first event, because the first event may be
	// a folder leaving: what it was can only be known from before it went.
	shape := remembered(reader)

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
		fold(ctx, shape, opts, raw, changes, lost)
	}()

	return &Watching{Changes: changes, Lost: lost}, nil
}

// fold collects events for a window and reports each path once.
//
// Reading the events and delivering them are kept apart. Whoever listens takes
// as long as it takes to reindex what it was told about, and the operating
// system goes on producing events meanwhile: a fold that waited for its listener
// would stop emptying the backlog, and everything past the end of it is dropped
// by the watcher with nothing said.
func fold(
	ctx context.Context,
	shape *folders,
	opts Options,
	raw <-chan notify.EventInfo,
	changes chan<- []string,
	lost chan<- struct{},
) {
	pending := map[string]bool{}
	// ready is what the window has already closed over, waiting to be taken. It
	// keeps growing while nobody takes it, rather than a second batch waiting
	// behind the first.
	var ready []string
	inReady := map[string]bool{}
	var window <-chan time.Time

	forget := func() {
		clear(pending)
		clear(inReady)
		ready, window = nil, nil
	}

	rescan := func() {
		forget()
		select {
		case lost <- struct{}{}:
		default:
		}
	}

	for {
		// The delivery arm is only in the select when there is something to
		// deliver; a nil channel is never chosen.
		var out chan<- []string
		if len(ready) > 0 {
			out = changes
		}

		select {
		case <-ctx.Done():
			return

		case out <- ready:
			ready = nil
			clear(inReady)

		case event, open := <-raw:
			if !open {
				return
			}
			// A full backlog means events were dropped. What was missed is
			// unknowable, so the vault is read again.
			if len(raw) == cap(raw) {
				rescan()
				continue
			}
			paths, whole := shape.concerns(event.Path())
			if whole {
				// A folder that is gone takes notes with it, and their paths
				// are known to the index rather than to the disk.
				rescan()
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
			for path := range pending {
				if !inReady[path] {
					inReady[path] = true
					ready = append(ready, path)
				}
			}
			clear(pending)
		}
	}
}

// folders remembers which paths in the vault are folders, so that a path which
// has gone can still be told apart from a file that has.
//
// A name cannot answer it: `2026.archive` is a folder and `notes.txt` is not,
// and every atomic save leaves a gone temporary file behind. The disk cannot
// answer it either — the thing that would say is the thing that no longer
// exists. So it is remembered while it is there, which is the one moment the
// question is answerable.
type folders struct {
	reader *VaultReader
	are    map[string]bool
}

// remembered walks the vault once for its shape. A folder made later is learnt
// from the event that makes it.
func remembered(reader *VaultReader) *folders {
	f := &folders{reader: reader, are: map[string]bool{".": true}}
	_ = filepath.WalkDir(reader.Root(), func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path, inside := reader.relative(p); inside {
			f.are[path] = true
		}
		return nil
	})
	return f
}

func (f *folders) forget(path string) {
	delete(f.are, path)
	for held := range f.are {
		if strings.HasPrefix(held, path+"/") {
			delete(f.are, held)
		}
	}
}

// concerns turns an absolute path from the operating system into the notes of
// this vault that it names.
//
// A file is itself, when the vault holds it. A folder is everything under it:
// a folder arrives with its contents already in place — copied, restored,
// checked out — and where the system has no recursion of its own the watch on
// it is established after the fact.
//
// `whole` is set when the answer cannot be worked out from the disk: a folder
// that has gone took notes with it, and their paths are known to the index
// rather than to anything still there. A path outside the vault is that case
// too, which is what arrives when a watched folder is renamed away.
func (f *folders) concerns(absolute string) (paths []string, whole bool) {
	path, inside := f.reader.relative(absolute)
	if !inside {
		return nil, true
	}

	info, err := os.Stat(absolute)
	switch {
	case err == nil && info.IsDir():
		f.are[path] = true
		var found []string
		_ = filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			held, inside := f.reader.relative(p)
			if !inside {
				return nil
			}
			if d.IsDir() {
				f.are[held] = true
				return nil
			}
			if f.reader.holds(held) {
				found = append(found, held)
			}
			return nil
		})
		return found, false

	case err != nil && f.are[path]:
		f.forget(path)
		return nil, true
	}

	if !f.reader.holds(path) {
		return nil, false
	}
	return []string{path}, false
}

// relative names a path the way the vault does. Anything outside it is not the
// vault's to answer for.
func (s *VaultReader) relative(absolute string) (path string, inside bool) {
	rel, err := filepath.Rel(s.root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
