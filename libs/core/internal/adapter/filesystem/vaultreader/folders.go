package vaultreader

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

// Folders records which paths in the vault are folders, so that a deleted path
// can still be classified as a folder or a file.
//
// A name does not determine the type — `2026.archive` is a folder and
// `notes.txt` is not — and a deleted path cannot be inspected, so the type is
// recorded while the path exists.
//
// The goroutine that drains the events reads the shape and the goroutine that
// walks a folder new to the watch writes it, so access is guarded by a lock.
type Folders struct {
	reader *VaultReader

	mu      sync.Mutex
	Indexed map[string]bool
}

// IsKnownFolder is whether this path was a folder when it was last there.
func (f *Folders) IsKnownFolder(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Indexed[path]
}

// addFolder remembers a path as a folder, which is the one moment the question is
// answerable.
func (f *Folders) addFolder(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Indexed[path] = true
}

// ReadShape walks the vault once to record its shape, stopping where the
// vault's own walk stops. A folder created later is recorded from the event that
// creates it.
//
// A folder the walk cannot enter is missing from the shape, and the first such
// error is returned.
func ReadShape(reader *VaultReader) (*Folders, error) {
	f := &Folders{reader: reader, Indexed: map[string]bool{".": true}}
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
		f.addFolder(path)
		return nil
	})
	if why == nil {
		why = walk
	}
	return f, why
}

// ReloadShape reads the shape from the vault currently now. It answers for events
// that were dropped, some of which may have made folders or taken them away.
// A folder the walk could not enter is missing from the shape it returns; the
// rescan that goes with this call already answers for it.
func (f *Folders) ReloadShape() {
	fresh, _ := ReadShape(f.reader)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Indexed = fresh.Indexed
}

func (f *Folders) removeFolder(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.Indexed, path)
	for target := range f.Indexed {
		if strings.HasPrefix(target, path+"/") {
			delete(f.Indexed, target)
		}
	}
}

// GetConcernedPaths converts an absolute operating-system path into the sources
// of this vault that it names. The reader decides what the vault holds, so the
// watcher and the walk give the same answer.
//
// A file maps to itself when the vault holds it. A folder new to the watch maps
// to everything under it, because such a folder arrives with its contents in
// place. A folder already known maps to nothing, and so does a folder the walk
// skips together with everything under it.
//
// A path that no longer exists and that the vault skips as a folder yields no
// source.
//
// `whole` is set when the answer cannot be derived from the disk: a deleted
// folder took sources with it, and only the index knows their paths. A path
// outside the vault is the same case.
//
// `walk` is a folder new to the watch, returned by name. This function runs
// where nothing may block.
func (f *Folders) GetConcernedPaths(absolute string) (paths []string, whole bool, walk string) {
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
		if f.IsKnownFolder(path) {
			return nil, false, ""
		}
		// Remembered before the walk, so a second event about the same folder
		// does not ask for it a second time.
		f.addFolder(path)
		return nil, false, absolute

	case err != nil && f.IsKnownFolder(path):
		f.removeFolder(path)
		return nil, true, ""

	case err != nil && f.reader.isSkipped(path, filepath.Base(absolute)):
		return nil, false, ""
	}

	if _, holds := f.reader.getKind(path); !holds {
		return nil, false, ""
	}
	return []string{path}, false, ""
}

// GetPathsUnder is what the vault holds under a folder new to the watch, and
// the folders under it remembered as it goes.
//
// A folder holding more entries than Entries is answered as the whole vault:
// the rest of the walk costs more than reading the vault again. So is a folder
// the walk could not enter: what it holds is missing from the shape, and a
// shape short of a folder cannot tell a folder that has gone from a file that
// has — the index would go on answering with files that are not there.
func (f *Folders) GetPathsUnder(ctx context.Context, absolute string) (paths []string, whole bool) {
	w := &underWalk{folders: f, absolute: absolute}
	_ = filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
		return w.visit(ctx, p, d, err)
	})
	if w.exceeded || w.short {
		return nil, true
	}
	return w.discovered, false
}

// underWalk is the walk of a folder new to the watch: the files found, and
// whether the walk went past the entries it may read or met a folder it could
// not enter.
type underWalk struct {
	folders    *Folders
	absolute   string
	discovered []string
	entryCount int
	exceeded   bool
	short      bool
}

// visit is one entry of the walk.
func (w *underWalk) visit(ctx context.Context, p string, d fs.DirEntry, err error) error {
	if ctx.Err() != nil {
		return fs.SkipAll
	}
	if err != nil {
		// The walk continues: as much of the shape as possible is recorded, and the
		// vault is read again for the rest.
		w.short = true
		return fs.SkipDir
	}
	w.entryCount++
	if w.entryCount > Entries {
		w.exceeded = true
		return fs.SkipAll
	}
	reader := w.folders.reader
	path, inside := reader.relative(p)
	if !inside {
		return nil
	}
	if d.IsDir() {
		if p != w.absolute && reader.isSkipped(path, d.Name()) {
			return fs.SkipDir
		}
		w.folders.addFolder(path)
		return nil
	}
	if _, holds := reader.getKind(path); holds {
		w.discovered = append(w.discovered, path)
	}
	return nil
}
