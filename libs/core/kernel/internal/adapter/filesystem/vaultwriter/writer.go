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

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/atomicfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
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

func (w *VaultWriter) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (_ domain.FileStat, err error) {
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
	defer closeAndJoin(&err, root)

	mode, err := getWriteMode(root, name, path, stat)
	if err != nil {
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

func (w *VaultWriter) Move(ctx context.Context, from, to string) (err error) {
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
		return fmt.Errorf("move %s to %s: %w", from, to, vault.ErrAlreadyExists)
	}
	root, arrives, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	leaves, err := w.getRelativeName(source)
	if err != nil {
		return err
	}

	if err := checkMoveTarget(root, leaves, arrives, from, to); err != nil {
		return err
	}

	if err := atomicfile.CreateParents(root, filepath.Dir(arrives), to); err != nil {
		return err
	}
	return atomicfile.Rename(ctx, root, leaves, arrives)
}

// CheckMoveTarget returns ErrAlreadyExists when a file other than the one at from is at to.
func (w *VaultWriter) CheckMoveTarget(ctx context.Context, from, to string) (err error) {
	if ctx.Err() != nil {
		return ctx.Err()
	}
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
	root, arrives, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	leaves, err := w.getRelativeName(source)
	if err != nil {
		return err
	}
	return checkMoveTarget(root, leaves, arrives, from, to)
}

func (w *VaultWriter) MakeFolder(ctx context.Context, path string) (err error) {
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
	defer closeAndJoin(&err, root)
	switch info, err := root.Stat(name); {
	case err == nil && !info.IsDir():
		return fmt.Errorf("make %s: %w", path, vault.ErrAlreadyExists)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return atomicfile.CreateParents(root, name, path)
}

func (w *VaultWriter) Remove(ctx context.Context, path string) (err error) {
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
	defer closeAndJoin(&err, root)
	if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// A note that is already absent is a successful delete.
	return nil
}

func (w *VaultWriter) resolveNotePath(path string) (string, error) {
	target, err := w.getContainedPath(path)
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	return target, nil
}

func (w *VaultWriter) resolvePath(path string) (string, error) {
	target, err := w.getContainedPath(path)
	if err != nil {
		return "", err
	}
	if !w.isReachable(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	return target, nil
}

func (w *VaultWriter) resolveNoteFile(path string) (string, error) {
	resolved, err := vaultconfig.ResolveLinks(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	// The note rules apply to the resolved path as well as to the requested name:
	// a link is a name for another location, and a rename replaces that location.
	inside, err := filepath.Rel(w.root, resolved)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, vault.ErrOutside)
	}
	if !w.isWritableNote(inside) {
		return "", fmt.Errorf("%s: %w", inside, edit.ErrNotANote)
	}
	return resolved, nil
}

func (w *VaultWriter) getContainedPath(path string) (string, error) {
	return vaultconfig.GetContainedPath(w.root, path, w.opts.GetServiceDir())
}

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

func (w *VaultWriter) getRelativeName(target string) (string, error) {
	name, err := filepath.Rel(w.root, target)
	if err != nil || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: %w", target, vault.ErrOutside)
	}
	return name, nil
}

// TrashDir is the folder a removed note is kept in. It is named here as well as
// where removal decides to use it: this is what may be written to, and that is
// what writes there.
const TrashDir = ".trash"

func (w *VaultWriter) isOurs(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if name := pathpkg.Base(dir); w.opts.IsService(name) || strings.EqualFold(name, TrashDir) {
			return true
		}
	}
	return false
}

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

// VaultWriters opens vaults for writing. Writers that must exclude each other share Locks.
type VaultWriters struct {
	Options vaultconfig.Options
	Locks   *Locks
}

func (w VaultWriters) Open(v domain.Vault) (*VaultWriter, error) {
	return OpenForWriting(v.Path, w.Options)
}

func (w VaultWriters) OpenNoteWriter(v domain.Vault) (vaultport.NoteWriter, error) {
	opened, err := w.Open(v)
	if err != nil {
		return nil, err
	}
	return opened, nil
}

func (w VaultWriters) OpenFolderWriter(v domain.Vault) (vaultport.FolderWriter, error) {
	opened, err := w.Open(v)
	if err != nil {
		return nil, err
	}
	return opened, nil
}

func (w VaultWriters) OpenFileMover(v domain.Vault) (vaultport.FileMover, error) {
	opened, err := w.Open(v)
	if err != nil {
		return nil, err
	}
	return opened, nil
}

func (w VaultWriters) OpenTrashEmptier(v domain.Vault) (vaultport.TrashEmptier, error) {
	opened, err := w.Open(v)
	if err != nil {
		return nil, err
	}
	return opened, nil
}

// Create writes a note and returns an error if one already exists.
//
// The existence check and the creation are one step, so two concurrent callers
// cannot both succeed.
func (w *VaultWriter) Create(ctx context.Context, path string, content []byte) (err error) {
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
	defer closeAndJoin(&err, root)
	if err := atomicfile.CreateParents(root, filepath.Dir(name), path); err != nil {
		return err
	}

	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("create %s: %w", path, vault.ErrAlreadyExists)
	}
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
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
func (w *VaultWriter) Import(ctx context.Context, path string, content io.Reader) (err error) {
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
	defer closeAndJoin(&err, root)
	switch _, err := root.Lstat(name); {
	case err == nil:
		return fmt.Errorf("bring %s: %w", path, vault.ErrAlreadyExists)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := atomicfile.CreateParents(root, filepath.Dir(name), path); err != nil {
		return err
	}
	return streamReplace(ctx, root, name, content)
}

func streamReplace(ctx context.Context, root *os.Root, target string, content io.Reader) (err error) {
	dir := filepath.Dir(target)
	tmp, pos, err := atomicfile.CreateTempFile(root, dir, atomicfile.GetTempPattern(filepath.Base(target)))
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := root.Remove(pos); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()

	if _, err := io.Copy(tmp, content); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Chmod(newFileMode); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := atomicfile.Rename(ctx, root, pos, target); err != nil {
		return err
	}
	return atomicfile.SyncDirectory(root, dir)
}

func getWriteMode(root *os.Root, name, path string, stat domain.FileStat) (fs.FileMode, error) {
	info, err := root.Stat(name)
	switch {
	case err == nil:
		if info.IsDir() {
			return 0, fmt.Errorf("write %s: it is a directory", path)
		}
		if !stat.IsZero() &&
			(info.Size() != stat.Size || !info.ModTime().Equal(stat.ModTime)) {
			return 0, fmt.Errorf("write %s: %w", path, edit.ErrStale)
		}
		return info.Mode().Perm(), nil
	case errors.Is(err, fs.ErrNotExist):
		// A missing note cannot have changed, so a caller holding a stat for it gets
		// ErrStale.
		if !stat.IsZero() {
			return 0, fmt.Errorf("write %s: %w", path, edit.ErrStale)
		}
		return newFileMode, nil
	default:
		return 0, err
	}
}

func checkMoveTarget(root *os.Root, leaves, arrives, from, to string) error {
	there, err := root.Lstat(arrives)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// A filesystem that ignores case or accent normalization reports the new name
	// as the file being renamed.
	here, err := root.Lstat(leaves)
	if err != nil {
		return err
	}
	if !os.SameFile(here, there) {
		return fmt.Errorf("move %s to %s: %w", from, to, vault.ErrAlreadyExists)
	}
	return nil
}
