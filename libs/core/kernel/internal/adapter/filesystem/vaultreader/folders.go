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

// Folders records which paths in the vault are folders, so a deleted path can
// be classified as a folder or a file.
// A name does not determine the type (`2026.archive` is a folder) and a deleted
// path cannot be inspected, so the type is recorded while the path exists.
// A lock guards access: the event goroutine reads, the folder-scanning goroutine writes.
type Folders struct {
	reader *VaultReader

	mu      sync.Mutex
	Indexed map[string]bool
}

func (f *Folders) IsKnownFolder(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Indexed[path]
}

func (f *Folders) addFolder(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Indexed[path] = true
}

// ReadShape scans the vault once to record its shape, ending where the vault's
// own walk ends. A folder created later is recorded from its creation event.
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
// A folder the walk could not enter is missing from the shape it keeps, and the
// first error the walk met is returned.
func (f *Folders) ReloadShape() error {
	fresh, err := ReadShape(f.reader)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Indexed = fresh.Indexed
	return err
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

// GetConcernedPaths converts an absolute path into the sources of this vault it
// names, by the reader's rules so watcher and walk agree. A file maps to itself;
// a folder new to the watch to everything under it; a known or skipped one to
// nothing. `whole` is set when the disk cannot answer (a deleted folder, a path
// outside the vault). `walk` is a folder new to the watch; it must not block.
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

// GetPathsUnder returns what the vault holds under a folder new to the watch,
// remembering the folders under it.
// A folder holding more entries than Entries, or one the walk could not enter,
// is answered as the whole vault: a shape short of a folder cannot tell a gone
// folder from a gone file, and the index would keep answering with absent files.
func (f *Folders) GetPathsUnder(ctx context.Context, absolute string) (paths []string, whole bool) {
	w := &underWalk{folders: f, absolute: absolute}
	if err := filepath.WalkDir(absolute, func(p string, d fs.DirEntry, err error) error {
		return w.visit(ctx, p, d, err)
	}); err != nil {
		w.short = true
	}
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
