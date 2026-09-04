package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Backlog is how many events are held while they are being folded. Past it the
// vault is read from scratch, which is cheaper than working out what was
// missed.
const Backlog = 4096

// Entries is how many entries a folder that arrives is followed through. Past
// it the vault is read from scratch, which is cheaper than the rest of the
// walk.
const Entries = 4096

// Watcher follows vaults on disk.
type Watcher struct {
	Options Options
}

// Watch follows one vault and reports the sources that change in it.
//
// The watch is on the tree, not on any file: editors save by writing a
// temporary file and renaming it over the original, so the file a watch was
// placed on stops existing at the first save.
func (w Watcher) Watch(
	ctx context.Context,
	v domain.Vault,
) (changes <-chan []string, lost <-chan struct{}, err error) {
	reader, err := Open(v.Path, w.Options)
	if err != nil {
		return nil, nil, err
	}
	opts := w.Options

	// The shape is taken before the first event, because the first event may be
	// a folder leaving: what it was can only be known from before it went.
	shape, why := remembered(reader)

	raw := make(chan notify.EventInfo, Backlog)
	tree := filepath.Join(reader.Root(), "...")
	if err := notify.Watch(tree, raw, notify.All); err != nil {
		return nil, nil, fmt.Errorf("watch %s: %w", v.Path, err)
	}

	folded := make(chan []string)
	gone := make(chan struct{}, 1)

	// A shape short of a folder the walk could not enter cannot tell a folder
	// that has gone from a file that has, so the vault is read again.
	if why != nil {
		gone <- struct{}{}
	}

	go func() {
		defer notify.Stop(raw)
		defer close(folded)
		fold(ctx, shape, opts, raw, folded, gone)
	}()

	return folded, gone, nil
}

// File follows one file and reports each time it changes.
//
// The watch is on the folder and not on the file: a file is replaced by being
// written beside itself and renamed over, so a watch placed on the file itself
// stops existing at the first write. Everything under that name is one file
// here — a database and the journals beside it are written together and are one
// change.
func (w Watcher) File(ctx context.Context, path string) (<-chan struct{}, error) {
	at, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(at)

	raw := make(chan notify.EventInfo, Backlog)
	if err := notify.Watch(filepath.Dir(at), raw, notify.All); err != nil {
		return nil, fmt.Errorf("watch %s: %w", path, err)
	}

	moved := make(chan struct{}, 1)
	go func() {
		defer notify.Stop(raw)
		defer close(moved)
		for {
			select {
			case <-ctx.Done():
				return
			case event, open := <-raw:
				if !open {
					return
				}
				if !strings.HasPrefix(filepath.Base(event.Path()), name) {
					continue
				}
				// One waiting message is as much as this says: a listener that
				// has not read the last one is a listener about to ask anyway.
				select {
				case moved <- struct{}{}:
				default:
				}
			}
		}
	}()
	return moved, nil
}

// fold collects events for a hold and reports each path once.
//
// Reading the events and delivering them are kept apart. Whoever listens takes
// as long as it takes to refresh what it was told about, and the operating
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
	// ready is what the hold has already closed over, waiting to be taken. It
	// keeps growing while nobody takes it: one batch, never two.
	var ready []string
	inReady := map[string]bool{}
	var hold <-chan time.Time
	// queued is the folders new to the watch that are waiting to be walked, and
	// walked is what a walk of one came to. A walk made here would stop the
	// backlog emptying for as long as it took, which is the one thing this
	// goroutine may never do.
	var queued []string
	asked := make(chan string)
	walked := make(chan found)

	going := make(chan struct{})
	var walking sync.WaitGroup
	walking.Add(1)
	go func() {
		defer walking.Done()
		walks(ctx, shape, going, asked, walked)
	}()
	defer walking.Wait()
	defer close(going)

	forget := func() {
		clear(pending)
		clear(inReady)
		ready, hold, queued = nil, nil, nil
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
		// deliver; a nil channel is never chosen. The walking arm the same.
		var out chan<- []string
		if len(ready) > 0 {
			out = changes
		}
		var walk chan<- string
		var next string
		if len(queued) > 0 {
			walk, next = asked, queued[0]
		}

		select {
		case <-ctx.Done():
			return

		case out <- ready:
			ready = nil
			clear(inReady)

		case walk <- next:
			queued = queued[1:]

		case one := <-walked:
			if one.whole {
				rescan()
				continue
			}
			for _, path := range one.paths {
				pending[path] = true
			}
			if hold == nil && len(pending) > 0 {
				hold = time.After(opts.hold())
			}

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
			paths, whole, folder := shape.concerns(event.Path())
			if whole {
				// A folder that is gone takes sources with it, and their paths
				// are known only to the index.
				rescan()
				continue
			}
			if folder != "" {
				queued = append(queued, folder)
			}
			for _, path := range paths {
				pending[path] = true
			}
			// Measured from the first event of a batch, not the last. A vault
			// being written to continuously — a sync client, a checkout — never
			// stops long enough for a deadline that moves with it.
			if hold == nil && len(pending) > 0 {
				hold = time.After(opts.hold())
			}

		case <-hold:
			hold = nil
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

// found is what a walk of one folder new to the watch came to.
type found struct {
	paths []string
	whole bool
}

// walks walks the folders handed to it, one after another, and answers with
// what each holds. It stands beside the fold rather than inside it, so a folder
// that arrives with a thousand files in it is walked while the backlog goes on
// emptying.
func walks(
	ctx context.Context,
	shape *folders,
	going <-chan struct{},
	asked <-chan string,
	walked chan<- found,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-going:
			return
		case at := <-asked:
			paths, whole := shape.inside(ctx, at)
			select {
			case <-ctx.Done():
				return
			case <-going:
				return
			case walked <- found{paths: paths, whole: whole}:
			}
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
// The shape is read by the goroutine that drains the events and written by the
// one that walks a folder new to the watch, so what is remembered is held under
// a lock.
type folders struct {
	reader *VaultReader

	mu    sync.Mutex
	known map[string]bool
}

// knows is whether this path was a folder when it was last there.
func (f *folders) knows(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.known[path]
}

// learn remembers a path as a folder, which is the one moment the question is
// answerable.
func (f *folders) learn(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.known[path] = true
}

// remembered walks the vault once for its shape, stopping where the vault's own
// walk stops. A folder made later is learnt from the event that makes it.
//
// A folder the walk could not enter is missing from the shape, and comes back
// as the first error it met.
func remembered(reader *VaultReader) (*folders, error) {
	f := &folders{reader: reader, known: map[string]bool{".": true}}
	var why error
	walk := filepath.WalkDir(reader.Root(), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if why == nil {
				why = err
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		path, inside := reader.relative(p)
		if !inside {
			return nil
		}
		if p != reader.Root() && reader.skipped(path, d.Name()) {
			return fs.SkipDir
		}
		f.learn(path)
		return nil
	})
	if why == nil {
		why = walk
	}
	return f, why
}

func (f *folders) forget(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.known, path)
	for held := range f.known {
		if strings.HasPrefix(held, path+"/") {
			delete(f.known, held)
		}
	}
}

// concerns turns an absolute path from the operating system into the sources of
// this vault that it names. What the vault holds is asked of the reader, so the
// watcher and the walk answer alike.
//
// A file is itself, when the vault holds it. A folder new to the watch is
// everything under it: such a folder arrives with its contents already in place
// — copied, restored, checked out — and where the system has no recursion of
// its own the watch on it is established after the fact. A folder already known
// names nothing: it is named because something inside it changed, and that
// something arrives as its own event. A folder the walk stops at names nothing,
// and neither does anything under it.
//
// `whole` is set when the answer cannot be worked out from the disk: a folder
// that has gone took sources with it, and their paths are known only to the
// index. A path outside the vault is that case too, and is what arrives when a
// watched folder is renamed away.
//
// `walk` is a folder new to the watch, handed back rather than walked: the
// walking is what takes the time, and this runs where nothing may take any.
func (f *folders) concerns(absolute string) (paths []string, whole bool, walk string) {
	path, inside := f.reader.relative(absolute)
	if !inside {
		return nil, true, ""
	}

	info, err := os.Stat(absolute)
	switch {
	case err == nil && info.IsDir():
		if path != "." && f.reader.skipped(path, filepath.Base(absolute)) {
			return nil, false, ""
		}
		if f.knows(path) {
			return nil, false, ""
		}
		// Remembered before the walk, so a second event about the same folder
		// does not ask for it a second time.
		f.learn(path)
		return nil, false, absolute

	case err != nil && f.knows(path):
		f.forget(path)
		return nil, true, ""
	}

	if _, holds := f.reader.holds(path); !holds {
		return nil, false, ""
	}
	return []string{path}, false, ""
}

// inside is what the vault holds under a folder new to the watch, and the
// folders under it remembered as it goes.
//
// A folder holding more entries than Entries is answered as the whole vault:
// the rest of the walk costs more than reading the vault again. So is a folder
// the walk could not enter: what it holds is missing from the shape, and a
// shape short of a folder cannot tell a folder that has gone from a file that
// has — the index would go on answering with files that are not there.
func (f *folders) inside(ctx context.Context, absolute string) (paths []string, whole bool) {
	var held []string
	seen, over, short := 0, false, false
	_ = filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return fs.SkipAll
		}
		if err != nil {
			// The rest of the walk still stands: as much of the shape as can be
			// learnt is learnt, and the vault is read again for what cannot.
			short = true
			return nil
		}
		seen++
		if seen > Entries {
			over = true
			return fs.SkipAll
		}
		path, inside := f.reader.relative(p)
		if !inside {
			return nil
		}
		if d.IsDir() {
			if p != absolute && f.reader.skipped(path, d.Name()) {
				return fs.SkipDir
			}
			f.learn(path)
			return nil
		}
		if _, holds := f.reader.holds(path); holds {
			held = append(held, path)
		}
		return nil
	})
	if over || short {
		return nil, true
	}
	return held, false
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
