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
	// The vault is where the links lead. A write lands at the resolved path, and
	// the rules about what the vault holds are asked of a name relative to it.
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
	// What the vault says not to touch is the vault's to say, and it says it to
	// the writer for the same reason it says it to the reader.
	if cfg, err := vaultconfig.ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultWriter{root: abs, opts: opts, ignored: opts.CompileIgnoring()}, nil
}

// newFileMode is what a note is created with. An existing note keeps the mode
// it already had: the person may have made one read-only on purpose.
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
		// A note that is not there yet cannot have changed, and a caller that
		// believed it was there is told so.
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
	// A file of any kind moves, and so does a folder. Both ends have to be a
	// place this vault keeps the person's files or one this application keeps
	// for itself.
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
		// A file already at the name is the name being taken, unless it is this
		// file: a filesystem that tells neither capitalisation nor the spelling
		// of an accent apart answers the new name with the file being renamed.
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
	// A note that is already gone is the outcome that was asked for.
	return nil
}

// resolveNotePath is where a resolveNotePath goes on this machine, and refuses a path this vault
// does not hold as a resolveNotePath.
//
// A writer driven by an agent is why the second half is here. Containment alone
// leaves everything else in the folder — attachments, another tool's state, the
// repository the vault is kept in — writable and removable by whatever asks,
// while the reader is already saying those paths do not exist.
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

// resolvePath is where a path is on this machine, for work that takes a file of any
// kind and a folder alike.
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

// resolveNoteFile is where the bytes of a note are, by the same rules note goes by. A note
// kept as a link to another resolveNoteFile in the vault has its bytes at the other end,
// and that is what a rename replaces.
func (w *VaultWriter) resolveNoteFile(path string) (string, error) {
	resolved, err := vaultconfig.ResolveLinks(w.root, path, w.opts.GetServiceDir())
	if err != nil {
		return "", err
	}
	if !w.isWritableNote(path) {
		return "", fmt.Errorf("%s: %w", path, vaultconfig.ErrNotANote)
	}
	// The rule is asked of where the bytes land as well as of the name they were
	// asked for by. A link is a name for another place, and the place is what a
	// rename replaces.
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

// isReachable answers whether a path in this vault is one the writer may put a
// file at or take one away from, whatever the file is and whether it is a
// folder. What the vault says to leave alone is left alone, and the folders
// this application keeps for itself are its own.
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

// Create writes a note where there is none, and refuses where there is one.
//
// does not, in one step, so two callers racing cannot both be told the path
// was free.
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
// The bytes are streamed, so a recording or an archive crosses in whatever room
// the machine has. They land beside the destination and are renamed over it, so
// the watcher reports the file once and reports it whole.
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
