package vaultwriter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

// VaultWriter changes one vault on disk.
//
// It is separate from the reader because reading and writing are different
// rights: everything in the application may read a vault, and only a use case
// holding the writer may change one.
type VaultWriter struct {
	root    string
	opts    vaultconfig.Options
	ignored *vaultconfig.IgnoreRules
	lock    *vaultLock
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
	return &VaultWriter{root: abs, opts: opts, ignored: opts.CompileIgnoring(), lock: newVaultLock()}, nil
}

// newFileMode is the mode of a created note. An existing note keeps its mode,
// which may be read-only.
const newFileMode fs.FileMode = 0o644

// Write replaces a note. A baseline that is not the zero stat must have the hash of the note on disk, read at
// the write, or the write changes nothing and returns edit.ErrStale.
func (w *VaultWriter) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (_ vault.WriteResult, err error) {
	if ctx.Err() != nil {
		return vault.WriteResult{}, ctx.Err()
	}
	target, err := w.resolveWritableFile(path, w.isWritableNote)
	if err != nil {
		return vault.WriteResult{}, err
	}
	root, name, err := w.openRoot(target)
	if err != nil {
		return vault.WriteResult{}, err
	}
	defer closeAndJoin(&err, root)

	// The baseline is compared and the file replaced under one lock: with two locks, two writers with one
	// baseline both pass the comparison.
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	mode, err := getWriteMode(ctx, root, name, path, stat)
	if err != nil {
		return vault.WriteResult{}, err
	}

	if err := createParents(root, filepath.Dir(name), path); err != nil {
		return vault.WriteResult{}, err
	}
	written, err := atomicfile.Replace(ctx, root, name, bytes.NewReader(content), mode)
	if err != nil {
		return vault.WriteResult{}, err
	}
	version := w.lock.version.Add(1)
	replaced := domain.FileStat{Path: path, Size: written.Size, ModTime: written.ModTime, Hash: written.Hash, Version: version}
	return vault.WriteResult{Stat: replaced, Version: version}, nil
}

func (w *VaultWriter) Move(ctx context.Context, from, to string) error {
	return movePath(ctx, w, from, to, nil)
}

// MoveFileWithHash compares the source hash and renames the file under one vault write lock.
func (w *VaultWriter) MoveFileWithHash(ctx context.Context, from, to, hash string) error {
	return movePath(ctx, w, from, to, &hash)
}

func movePath(ctx context.Context, w *VaultWriter, from, to string, hash *string) (err error) {
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
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	leaves, err := getRelativeName(w.root, source)
	if err != nil {
		return err
	}

	if hash != nil {
		if err := checkMoveHash(ctx, root, leaves, from, *hash); err != nil {
			return err
		}
	}
	if err := checkMoveTarget(root, leaves, arrives, from, to); err != nil {
		return err
	}
	return renameToTarget(ctx, root, leaves, arrives, to, hash != nil)
}

func renameToTarget(ctx context.Context, root *os.Root, leaves, arrives, to string, hasBaseline bool) error {
	if err := createParents(root, filepath.Dir(arrives), to); err != nil {
		return err
	}
	if hasBaseline {
		if err := ctx.Err(); err != nil {
			return err
		}
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
	leaves, err := getRelativeName(w.root, source)
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
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	switch info, err := root.Stat(name); {
	case err == nil && !info.IsDir():
		return fmt.Errorf("make %s: %w", path, vault.ErrAlreadyExists)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return createParents(root, name, path)
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
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// A note that is already absent is a successful delete.
	return nil
}

// RemoveUnchanged deletes the file at path when its size and modification time are the ones of was, and
// returns vault.ErrChanged and keeps the file when they are not. The comparison and the deletion happen under
// the write lock, so a save through the writer cannot come between them.
func (w *VaultWriter) RemoveUnchanged(ctx context.Context, path string, was domain.FileStat) (err error) {
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
	w.lock.write.Lock()
	defer w.lock.write.Unlock()
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() || info.Size() != was.Size || !info.ModTime().Equal(was.ModTime) {
		return fmt.Errorf("remove %s: %w", path, vault.ErrChanged)
	}
	return root.Remove(name)
}

func (w *VaultWriter) resolveNotePath(path string) (string, error) {
	target, err := vaultconfig.GetContainedPath(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	return target, nil
}

func (w *VaultWriter) resolvePath(path string) (string, error) {
	target, err := vaultconfig.GetContainedPath(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !w.isReachable(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	return target, nil
}

func (w *VaultWriter) resolveWritableFile(path string, isWritable func(string) bool) (string, error) {
	resolved, err := vaultconfig.ResolveLinks(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !isWritable(path) {
		return "", fmt.Errorf("%s: %w", path, edit.ErrNotANote)
	}
	// Both the requested and resolved path must be writable.
	inside, err := filepath.Rel(w.root, resolved)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, vault.ErrOutside)
	}
	if !isWritable(inside) {
		return "", fmt.Errorf("%s: %w", inside, edit.ErrNotANote)
	}
	return resolved, nil
}

func (w *VaultWriter) openRoot(target string) (*os.Root, string, error) {
	name, err := getRelativeName(w.root, target)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(w.root)
	if err != nil {
		return nil, "", err
	}
	return root, name, nil
}
