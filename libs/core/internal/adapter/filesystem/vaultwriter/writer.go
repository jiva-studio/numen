package vaultwriter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/atomicfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// VaultWriter changes one vault on disk.
//
// It is separate from the reader because reading and writing are different
// rights: everything in the application may read a vault, and only a use case
// holding the writer may change one.
type VaultWriter struct {
	root    string
	opts    vaultconfig.Options
	ignored *vaultconfig.Ignoring
}

// OpenForWriting prepares a vault to be changed. The folder must already be
// one: a vault is given its identity by being added, not by being written to.
func OpenForWriting(root string, opts vaultconfig.Options) (*VaultWriter, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// Symlinks in the vault root are resolved. A write lands at the resolved path,
	// and the vault rules are evaluated on the path relative to it.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	// The vault's ignore rules apply to the writer as they do to the reader.
	if cfg, err := vaultconfig.ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultWriter{root: abs, opts: opts, ignored: opts.CompileIgnoring()}, nil
}

// newFileMode is the mode of a created note. An existing note keeps its mode,
// which may be read-only.
const newFileMode fs.FileMode = 0o644

func (w *VaultWriter) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	if ctx.Err() != nil {
		return domain.FileStat{}, ctx.Err()
	}
	target, err := w.resolveNoteFile(path)
	if err != nil {
		return domain.FileStat{}, err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer root.Close()

	mode := newFileMode
	switch info, err := root.Stat(name); {
	case err == nil:
		if info.IsDir() {
			return domain.FileStat{}, fmt.Errorf("write %s: it is a directory", path)
		}
		mode = info.Mode().Perm()
		if !stat.IsZero() &&
			(info.Size() != stat.Size || !info.ModTime().Equal(stat.ModTime)) {
			return domain.FileStat{}, fmt.Errorf("write %s: %w", path, vaultport.ErrStale)
		}
	case errors.Is(err, fs.ErrNotExist):
		// A missing note cannot have changed, so a caller holding a stat for it gets
		// ErrStale.
		if !stat.IsZero() {
			return domain.FileStat{}, fmt.Errorf("write %s: %w", path, vaultport.ErrStale)
		}
	default:
		return domain.FileStat{}, err
	}

	if err := atomicfile.CreateParents(root, filepath.Dir(name), path); err != nil {
		return domain.FileStat{}, err
	}
	contentVal, err := atomicfile.Replace(ctx, root, name, bytes.NewReader(content), mode)
	if err != nil {
		return domain.FileStat{}, err
	}
	contentVal.Path = path
	return contentVal, nil
}

func (w *VaultWriter) Move(ctx context.Context, from, to string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// A file or a folder can be moved. Both ends must be in the user's part of the
	// vault or in a folder the application manages.
	source, err := w.resolvePath(from)
	if err != nil {
		return err
	}
	target, err := w.resolvePath(to)
	if err != nil {
		return err
	}
	if source == target {
		return nil
	}
	// A folder does not go inside itself: the destination is a place the folder
	// itself holds.
	if strings.HasPrefix(target, source+string(filepath.Separator)) {
		return fmt.Errorf("move %s to %s: %w", from, to, vaultport.ErrAlreadyExists)
	}
	root, arrives, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	leaves, err := w.getRelativeName(source)
	if err != nil {
		return err
	}

	switch there, err := root.Lstat(arrives); {
	case err == nil:
		// A file already at the target name is a conflict unless it is the source
		// itself: a filesystem that ignores case or accent normalization reports the
		// new name as the file being renamed.
		here, err := root.Lstat(leaves)
		if err != nil {
			return err
		}
		if !os.SameFile(here, there) {
			return fmt.Errorf("move %s to %s: %w", from, to, vaultport.ErrAlreadyExists)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if err := atomicfile.CreateParents(root, filepath.Dir(arrives), to); err != nil {
		return err
	}
	return atomicfile.Rename(ctx, root, leaves, arrives)
}

func (w *VaultWriter) MakeFolder(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.resolvePath(path)
	if err != nil {
		return err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	switch info, err := root.Stat(name); {
	case err == nil && !info.IsDir():
		return fmt.Errorf("make %s: %w", path, vaultport.ErrAlreadyExists)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return atomicfile.CreateParents(root, name, path)
}

func (w *VaultWriter) Remove(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.resolvePath(path)
	if err != nil {
		return err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// A note that is already absent is a successful delete.
	return nil
}

// resolveNotePath returns the filesystem path of a note and rejects a path that is
// not a writable note of this vault.
//
// The second check applies because an agent drives the writer. Containment alone
// leaves every other file in the folder writable and removable (attachments, the
// state of other tools, the repository containing the vault), while the reader
// reports those paths as nonexistent.
func (w *VaultWriter) resolveNotePath(path string) (string, error) {
	target, err := w.getContainedPath(path)
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, vaultconfig.ErrNotANote)
	}
	return target, nil
}

// resolvePath returns the filesystem path of a file or folder.
func (w *VaultWriter) resolvePath(path string) (string, error) {
	target, err := w.getContainedPath(path)
	if err != nil {
		return "", err
	}
	if !w.isReachable(path) {
		return "", fmt.Errorf("%s: %w", path, vaultconfig.ErrNotANote)
	}
	return target, nil
}

// resolveNoteFile returns the file holding the bytes of a note, under the same
// rules as resolveNotePath. A note that is a link to another file in the vault
// has its bytes at the target, and a rename replaces the target.
func (w *VaultWriter) resolveNoteFile(path string) (string, error) {
	resolved, err := vaultconfig.ResolveLinks(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, vaultconfig.ErrNotANote)
	}
	// The note rules apply to the resolved path as well as to the requested name:
	// a link is a name for another location, and a rename replaces that location.
	inside, err := filepath.Rel(w.root, resolved)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, vaultconfig.ErrOutside)
	}
	if !w.isWritableNote(inside) {
		return "", fmt.Errorf("%s: %w", inside, vaultconfig.ErrNotANote)
	}
	return resolved, nil
}

// getContainedPath is only that: somewhere in this vault. Where a note goes when
// it is taken out of the vault's sight is such a place and is deliberately not a
// note.
func (w *VaultWriter) getContainedPath(path string) (string, error) {
	return vaultconfig.GetContainedPath(w.root, path, w.opts.GetServiceDir())
}

// openRoot is the vault as a handle, and a place in it as a name that handle
// takes. Every step of a write is made through the handle, so a folder swapped
// not by a rule read a moment before. The caller closes the handle.
func (w *VaultWriter) openRoot(target string) (*os.Root, string, error) {
	name, err := w.getRelativeName(target)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(w.root)
	if err != nil {
		return nil, "", err
	}
	return root, name, nil
}

// getRelativeName is a place on this machine as a name under the vault's root.
func (w *VaultWriter) getRelativeName(target string) (string, error) {
	name, err := filepath.Rel(w.root, target)
	if err != nil || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: %w", target, vaultconfig.ErrOutside)
	}
	return name, nil
}

// TrashDir is the folder a removed note is kept in. It is named here as well as
// where removal decides to use it: this is what may be written to, and that is
// what writes there.
const TrashDir = ".trash"

// isOurs is a folder this application keeps for itself inside the vault: the one
// it writes its own state into, and the one a note goes to when it is taken out
// of the vault's sight.
func (w *VaultWriter) isOurs(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if name := pathpkg.Base(dir); w.opts.IsService(name) || strings.EqualFold(name, TrashDir) {
			return true
		}
	}
	return false
}

// isReachable reports whether the writer may create or remove the file or folder
// at path. Paths matched by the vault ignore rules are not reachable; folders
// the application manages are.
func (w *VaultWriter) isReachable(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if w.isOurs(clean) {
		return true
	}
	if w.ignored.MatchesPath(clean) {
		return false
	}
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if w.opts.IsService(pathpkg.Base(dir)) {
			return false
		}
	}
	return true
}

// isWritableNote reports whether the path names a note the writer may write: a
// note outside what the vault ignores and outside the service folder.
func (w *VaultWriter) isWritableNote(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if !w.opts.IsNote(pathpkg.Base(clean)) {
		return false
	}
	if w.ignored.MatchesPath(clean) {
		return false
	}
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if w.opts.IsService(pathpkg.Base(dir)) {
			return false
		}
	}
	return true
}

// Root is where this vault is on disk.
func (w *VaultWriter) Root() string { return w.root }

// VaultWriters opens vaults for writing.
type VaultWriters struct{ Options vaultconfig.Options }

func (w VaultWriters) Open(v domain.Vault) (vaultport.VaultWriter, error) {
	return OpenForWriting(v.Path, w.Options)
}

// Create writes a note and returns an error if one already exists.
//
// The existence check and the creation are one step, so two concurrent callers
// cannot both succeed.
func (w *VaultWriter) Create(ctx context.Context, path string, content []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.resolveNotePath(path)
	if err != nil {
		return err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := atomicfile.CreateParents(root, filepath.Dir(name), path); err != nil {
		return err
	}

	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("create %s: %w", path, vaultport.ErrAlreadyExists)
	}
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return atomicfile.SyncDirectory(root, filepath.Dir(name))
}

// Import copies a file from this machine into the vault.
//
// The content is streamed, so a recording or an archive of any size is imported
// within the available memory. It is written to a temporary file next to the
// destination and renamed over it, so the watcher reports the file once, complete.
func (w *VaultWriter) Import(ctx context.Context, path string, content io.Reader) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.resolvePath(path)
	if err != nil {
		return err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	switch _, err := root.Lstat(name); {
	case err == nil:
		return fmt.Errorf("bring %s: %w", path, vaultport.ErrAlreadyExists)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := atomicfile.CreateParents(root, filepath.Dir(name), path); err != nil {
		return err
	}
	return streamReplace(ctx, root, name, content)
}

// streamReplace streams content beside the target and renames it over the top, by the
// same rules replace writes bytes it already holds.
func streamReplace(ctx context.Context, root *os.Root, target string, content io.Reader) error {
	dir := filepath.Dir(target)
	tmp, pos, err := atomicfile.CreateTempFile(root, dir, atomicfile.GetTempPattern(filepath.Base(target)))
	if err != nil {
		return err
	}
	defer root.Remove(pos)

	if _, err := io.Copy(tmp, content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(newFileMode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := atomicfile.Rename(ctx, root, pos, target); err != nil {
		return err
	}
	return atomicfile.SyncDirectory(root, dir)
}
