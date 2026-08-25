package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// VaultWriter changes one vault on disk.
//
// It is separate from the reader because reading and writing are different
// rights: everything in the application may read a vault, and four use cases
// may write one.
type VaultWriter struct {
	root    string
	opts    Options
	ignored *ignore.GitIgnore
}

// OpenForWriting prepares a vault to be changed. The folder must already be
// one: a vault is given its identity by being added, not by being written to.
func OpenForWriting(root string, opts Options) (*VaultWriter, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
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
	if cfg, err := ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultWriter{root: abs, opts: opts, ignored: opts.ignored()}, nil
}

// newFileMode is what a note is created with. An existing note keeps the mode
// it already had: the person may have made one read-only on purpose.
const newFileMode fs.FileMode = 0o644

func (w *VaultWriter) Write(ctx context.Context, path string, content []byte, fingerprint domain.FileRef) (domain.FileRef, error) {
	if ctx.Err() != nil {
		return domain.FileRef{}, ctx.Err()
	}
	target, err := w.file(path)
	if err != nil {
		return domain.FileRef{}, err
	}

	mode := newFileMode
	switch info, err := os.Stat(target); {
	case err == nil:
		if info.IsDir() {
			return domain.FileRef{}, fmt.Errorf("write %s: it is a directory", path)
		}
		mode = info.Mode().Perm()
		if fingerprint != (domain.FileRef{}) &&
			(info.Size() != fingerprint.Size || info.ModTime().UnixNano() != fingerprint.MTime) {
			return domain.FileRef{}, fmt.Errorf("write %s: %w", path, port.ErrChanged)
		}
	case errors.Is(err, fs.ErrNotExist):
		// A note that is not there yet cannot have changed, and a caller that
		// believed it was there is told so.
		if fingerprint != (domain.FileRef{}) {
			return domain.FileRef{}, fmt.Errorf("write %s: %w", path, port.ErrChanged)
		}
	default:
		return domain.FileRef{}, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.FileRef{}, err
	}
	written, err := replace(target, content, mode)
	if err != nil {
		return domain.FileRef{}, err
	}
	written.Path = path
	return written, nil
}

// replace writes content beside the target and renames it over the top.
//
// The temporary file is named with a leading dot so that the watcher never
// reports it: a name beginning with a dot is not a note, which is
// the same rule that keeps an editor's own temporary files out of the index.
//
// The contents are flushed before the rename. Without that, a machine that
// loses power between the two can leave the rename recorded and the bytes not,
// which is the one outcome this whole arrangement exists to prevent.
//
// The fingerprint that comes back is taken from the temporary file's own
// descriptor. The rename carries the file across whole, so its size and its
// modification time are the ones at the target from the moment the rename
// lands, and a caller holding them is holding the file it just wrote.
func replace(target string, content []byte, mode fs.FileMode) (domain.FileRef, error) {
	dir, name := filepath.Split(target)
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return domain.FileRef{}, err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return domain.FileRef{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return domain.FileRef{}, err
	}
	info, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return domain.FileRef{}, err
	}
	if err := tmp.Close(); err != nil {
		return domain.FileRef{}, err
	}
	// Changing the mode moves no modification time.
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return domain.FileRef{}, err
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return domain.FileRef{}, err
	}
	written := domain.FileRef{Size: info.Size(), MTime: info.ModTime().UnixNano()}
	return written, settle(dir)
}

// settle flushes the folder the rename was recorded in.
//
// Flushing the file is what keeps its contents whole; flushing the folder is
// what keeps the rename itself. Without this a machine that loses power can
// come back with the old note and the temporary one beside it, having durably
// written neither the swap nor anything worse.
//
// Not every filesystem lets a directory be opened for this, and the ones that
// refuse are the ones that did not need it. A refusal is not an error to hand
// back: the note is written either way.
func settle(dir string) error {
	folder, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer folder.Close()
	_ = folder.Sync()
	return nil
}

func (w *VaultWriter) Move(ctx context.Context, from, to string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// A file of any kind moves, and so does a folder. Both ends have to be a
	// place this vault keeps the person's files or one this application keeps
	// for itself.
	source, err := w.reach(from)
	if err != nil {
		return err
	}
	target, err := w.reach(to)
	if err != nil {
		return err
	}
	if source == target {
		return nil
	}

	switch _, err := os.Lstat(target); {
	case err == nil:
		return fmt.Errorf("move %s to %s: %w", from, to, port.ErrOccupied)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(source, target)
}

func (w *VaultWriter) MakeFolder(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.reach(path)
	if err != nil {
		return err
	}
	switch info, err := os.Stat(target); {
	case err == nil && !info.IsDir():
		return fmt.Errorf("make %s: %w", path, port.ErrOccupied)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return os.MkdirAll(target, 0o755)
}

func (w *VaultWriter) Remove(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.reach(path)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// A note that is already gone is the outcome that was asked for.
	return nil
}

// inside is where a note goes on this machine, and refuses a path this vault
// does not hold as a note.
//
// A writer driven by an agent is why the second half is here. Containment alone
// leaves everything else in the folder — attachments, another tool's state, the
// repository the vault is kept in — writable and removable by whatever asks,
// while the reader is already saying those paths do not exist.
func (w *VaultWriter) note(path string) (string, error) {
	target, err := w.inside(path)
	if err != nil {
		return "", err
	}
	if !w.holds(path) {
		return "", fmt.Errorf("%s: %w", path, ErrNotANote)
	}
	return target, nil
}

// reach is where a path is on this machine, for work that takes a file of any
// kind and a folder alike.
func (w *VaultWriter) reach(path string) (string, error) {
	target, err := w.inside(path)
	if err != nil {
		return "", err
	}
	if !w.reachable(path) {
		return "", fmt.Errorf("%s: %w", path, ErrNotANote)
	}
	return target, nil
}

// file is where the bytes of a note are, by the same rules note goes by. A note
// kept as a link to another file in the vault has its bytes at the other end,
// and that is what a rename replaces.
func (w *VaultWriter) file(path string) (string, error) {
	real, err := followed(w.root, path, w.opts.serviceDir())
	if err != nil {
		return "", err
	}
	if !w.holds(path) {
		return "", fmt.Errorf("%s: %w", path, ErrNotANote)
	}
	// The rule is asked of where the bytes land as well as of the name they were
	// asked for by. A link is a name for another place, and the place is what a
	// rename replaces.
	inside, err := filepath.Rel(w.root, real)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, ErrOutside)
	}
	if !w.holds(inside) {
		return "", fmt.Errorf("%s: %w", inside, ErrNotANote)
	}
	return real, nil
}

// inside is only that: somewhere in this vault. Where a note goes when it is
// taken out of the vault's sight is such a place and is deliberately not a note.
func (w *VaultWriter) inside(path string) (string, error) {
	return inside(w.root, path, w.opts.serviceDir())
}

// TrashDir is the folder a removed note is kept in. It is named here as well as
// where removal decides to use it: this is what may be written to, and that is
// what writes there.
const TrashDir = ".trash"

// ours is a folder this application keeps for itself inside the vault: the one
// it writes its own state into, and the one a note goes to when it is taken out
// of the vault's sight.
func (w *VaultWriter) ours(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		switch pathpkg.Base(dir) {
		case w.opts.serviceDir(), TrashDir:
			return true
		}
	}
	return false
}

// reachable answers whether a path in this vault is one the writer may put a
// file at or take one away from, whatever the file is and whether it is a
// folder. What the vault says to leave alone is left alone, and the folders
// this application keeps for itself are its own.
func (w *VaultWriter) reachable(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if w.ours(clean) {
		return true
	}
	if w.ignored.MatchesPath(clean) {
		return false
	}
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if pathpkg.Base(dir) == w.opts.serviceDir() {
			return false
		}
	}
	return true
}

// holds answers the same question about a path that a walk answers about the
// files it reports, and by the same rules.
func (w *VaultWriter) holds(path string) bool {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	if !w.opts.isNote(pathpkg.Base(clean)) {
		return false
	}
	if w.ignored.MatchesPath(clean) {
		return false
	}
	for dir := pathpkg.Dir(clean); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if pathpkg.Base(dir) == w.opts.serviceDir() {
			return false
		}
	}
	return true
}

// Root is where this vault is on disk.
func (w *VaultWriter) Root() string { return w.root }

// Writers opens vaults for writing.
type Writers struct{ Options Options }

func (w Writers) Open(v domain.Vault) (port.VaultWriter, error) {
	return OpenForWriting(v.Path, w.Options)
}

// Create writes a note where there is none, and refuses where there is one.
//
// The refusal comes from the filesystem: O_EXCL either creates the file or
// does not, in one step, so two callers racing cannot both be told the path
// was free.
func (w *VaultWriter) Create(ctx context.Context, path string, content []byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.note(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("create %s: %w", path, port.ErrOccupied)
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
	return settle(filepath.Dir(target))
}
