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

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
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

	if err := createParents(root, filepath.Dir(name), path); err != nil {
		return domain.FileStat{}, err
	}
	written, err := atomicfile.Replace(ctx, root, name, bytes.NewReader(content), mode)
	if err != nil {
		return domain.FileStat{}, err
	}
	return domain.FileStat{Path: path, Size: written.Size, ModTime: written.ModTime}, nil
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

	if err := createParents(root, filepath.Dir(arrives), to); err != nil {
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
