package filesystem

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Entries is how many entries a folder that arrives is followed through. Past
// it the vault is read from scratch, which is cheaper than the rest of the
// walk.
const Entries = 4096

// folders remembers which paths in the vault are folders, so that a path which
// has gone can still be told apart from a file that has.
//
// A name cannot answer it — `2026.archive` is a folder and `notes.txt` is not —
// and neither can a path that is gone, so it is remembered while it is there.
//
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

// readShape walks the vault once for its shape, stopping where the vault's own
// walk stops. A folder made later is learnt from the event that makes it.
//
// A folder the walk could not enter is missing from the shape, and comes back
// as the first error it met.
func readShape(reader *VaultReader) (*folders, error) {
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
		if p != reader.Root() && reader.isSkipped(path, d.Name()) {
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

// again reads the shape from the vault as it stands now. It answers for events
// that were dropped, some of which may have made folders or taken them away.
// A folder the walk could not enter is missing from the shape it returns; the
// rescan that goes with this call already answers for it.
func (f *folders) again() {
	fresh, _ := readShape(f.reader)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.known = fresh.known
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

// getConcernedPaths turns an absolute path from the operating system into the sources of
// this vault that it names. What the vault holds is asked of the reader, so the
// watcher and the walk answer alike.
//
// A file is itself, when the vault holds it. A folder new to the watch is
// everything under it, since such a folder arrives with its contents already in
// place. A folder already known names nothing, and so does one the walk stops
// at and everything under it.
//
// `whole` is set where the answer cannot be worked out from the disk: a folder
// that has gone took sources with it, and their paths are known only to the
// index. A path outside the vault is that case too.
//
// `walk` is a folder new to the watch, handed back as a name. This runs where
// nothing may take time.
func (f *folders) getConcernedPaths(absolute string) (paths []string, whole bool, walk string) {
	path, inside := f.reader.relative(absolute)
	if !inside {
		return nil, true, ""
	}

	info, err := os.Stat(absolute)
	switch {
	case err == nil && info.IsDir():
		if path != "." && f.reader.isSkipped(path, filepath.Base(absolute)) {
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

// getPathsUnder is what the vault holds under a folder new to the watch, and
// the folders under it remembered as it goes.
//
// A folder holding more entries than Entries is answered as the whole vault:
// the rest of the walk costs more than reading the vault again. So is a folder
// the walk could not enter: what it holds is missing from the shape, and a
// shape short of a folder cannot tell a folder that has gone from a file that
// has — the index would go on answering with files that are not there.
func (f *folders) getPathsUnder(ctx context.Context, absolute string) (paths []string, whole bool) {
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
			return nil //nolint:nilerr // what could not be read is carried out in whole
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
			if p != absolute && f.reader.isSkipped(path, d.Name()) {
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
