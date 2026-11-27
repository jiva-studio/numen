package vaultwriter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

func getRelativeName(root, target string) (string, error) {
	name, err := filepath.Rel(root, target)
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
	if w.ignored.IsMatch(clean) {
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
	return w.opts.IsNote(pathpkg.Base(clean)) && w.isWritableFile(clean)
}

func (w *VaultWriter) isWritableFile(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if w.ignored.IsMatch(clean) {
		return false
	}
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if w.opts.IsService(pathpkg.Base(dir)) {
			return false
		}
	}
	return true
}

// VaultWriters opens vaults for writing. Writers that must exclude each other share Locks.
type VaultWriters struct {
	Options vaultconfig.Options
	Locks   *Locks
}

func (w VaultWriters) Open(v domain.Vault) (*VaultWriter, error) {
	opened, err := OpenForWriting(v.Path, w.Options)
	if err != nil {
		return nil, err
	}
	if w.Locks != nil {
		opened.lock = w.Locks.getLock(opened.root)
	}
	return opened, nil
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
	return w.createFile(path, target, content)
}

// CreateFile creates a visible vault file exclusively.
func (w VaultWriters) CreateFile(ctx context.Context, v domain.Vault, path string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	writer, err := w.Open(v)
	if err != nil {
		return err
	}
	target, err := writer.resolveWritableFile(path, writer.isWritableFile)
	if err != nil {
		return err
	}
	return writer.createFile(path, target, content)
}

func (w *VaultWriter) createFile(path, target string, content []byte) (err error) {
	root, name, err := w.openRoot(target)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, root)
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	if err := createParents(root, filepath.Dir(name), path); err != nil {
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
	if err := checkImportTarget(root, name, path); err != nil {
		return err
	}
	if err := createParents(root, filepath.Dir(name), path); err != nil {
		return err
	}
	_, err = atomicfile.ReplaceUnder(ctx, root, name, content, newFileMode, func(rename func() error) error {
		w.lock.write.Lock()
		defer w.lock.write.Unlock()
		if err := checkImportTarget(root, name, path); err != nil {
			return err
		}
		return rename()
	})
	return err
}

// Replace streams content over the file at path, whole or not at all, and keeps the permissions of the file. A path
// with no file returns fs.ErrNotExist.
func (w *VaultWriter) Replace(ctx context.Context, path string, content io.Reader, baseline domain.FileStat) (err error) {
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
	info, err := root.Stat(name)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("replace %s: it is a directory", path)
	}
	_, err = atomicfile.ReplaceUnder(ctx, root, name, content, info.Mode().Perm(), func(rename func() error) error {
		w.lock.write.Lock()
		defer w.lock.write.Unlock()
		if err := checkReplaceBaseline(root, name, path, baseline); err != nil {
			return err
		}
		return rename()
	})
	return err
}

func checkReplaceBaseline(root *os.Root, name, path string, baseline domain.FileStat) error {
	if baseline.IsZero() {
		return nil
	}
	stale, err := isStale(root, name, baseline)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("replace %s: %w", path, edit.ErrStale)
	}
	if err != nil {
		return err
	}
	if stale {
		return fmt.Errorf("replace %s: %w", path, edit.ErrStale)
	}
	return nil
}

func getWriteMode(root *os.Root, name, path string, stat domain.FileStat) (fs.FileMode, error) {
	info, err := root.Stat(name)
	switch {
	case err == nil:
		if info.IsDir() {
			return 0, fmt.Errorf("write %s: it is a directory", path)
		}
		stale, err := isStale(root, name, stat)
		if err != nil {
			return 0, err
		}
		if stale {
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

func checkMoveHash(root *os.Root, name, path, hash string) error {
	info, err := root.Stat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || hash == "" {
		return fmt.Errorf("move %s: %w", path, edit.ErrStale)
	}
	stale, err := isStale(root, name, domain.FileStat{Hash: hash})
	if err != nil {
		return err
	}
	if stale {
		return fmt.Errorf("move %s: %w", path, edit.ErrStale)
	}
	return nil
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

func createParents(root *os.Root, dir, path string) error {
	err := root.MkdirAll(dir, 0o755)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("make %s: %w", path, vault.ErrAlreadyExists)
	}
	return err
}

func isStale(root *os.Root, name string, baseline domain.FileStat) (_ bool, err error) {
	if baseline.IsZero() {
		return false, nil
	}
	file, err := root.Open(name)
	if err != nil {
		return false, err
	}
	defer closeAndJoin(&err, file)
	hash, err := domain.HashReader(file)
	if err != nil {
		return false, err
	}
	return hash != baseline.Hash, nil
}

// GetVersion returns the number of writes the vault has taken since the core started. Writers made
// without locks to share count apart, so it is zero for them.
func (w VaultWriters) GetVersion(v domain.Vault) uint64 {
	if w.Locks == nil {
		return 0
	}
	return w.Locks.getLock(v.Path).version.Load()
}

func checkImportTarget(root *os.Root, name, path string) error {
	_, err := root.Lstat(name)
	switch {
	case err == nil:
		return fmt.Errorf("bring %s: %w", path, vault.ErrAlreadyExists)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return nil
}
