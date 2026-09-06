package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// VaultWriter changes one vault on disk.
//
// It is separate from the reader because reading and writing are different
// rights: everything in the application may read a vault, and only a use case
// holding the writer may change one.
type VaultWriter struct {
	root    string
	opts    Options
	ignored *ignoring
}

// OpenForWriting prepares a vault to be changed. The folder must already be
// one: a vault is given its identity by being added, not by being written to.
func OpenForWriting(root string, opts Options) (*VaultWriter, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// The vault is where the links lead. A write lands at the resolved path, and
	// the rules about what the vault holds are asked of a name relative to it.
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
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

func (w *VaultWriter) Write(ctx context.Context, path string, content []byte, fingerprint domain.Fingerprint) (domain.Fingerprint, error) {
	if ctx.Err() != nil {
		return domain.Fingerprint{}, ctx.Err()
	}
	target, err := w.file(path)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	root, name, err := w.beneath(target)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	defer root.Close()

	mode := newFileMode
	switch info, err := root.Stat(name); {
	case err == nil:
		if info.IsDir() {
			return domain.Fingerprint{}, fmt.Errorf("write %s: it is a directory", path)
		}
		mode = info.Mode().Perm()
		if !fingerprint.IsZero() &&
			(info.Size() != fingerprint.Size || !info.ModTime().Equal(fingerprint.ModTime)) {
			return domain.Fingerprint{}, fmt.Errorf("write %s: %w", path, port.ErrStale)
		}
	case errors.Is(err, fs.ErrNotExist):
		// A note that is not there yet cannot have changed, and a caller that
		// believed it was there is told so.
		if !fingerprint.IsZero() {
			return domain.Fingerprint{}, fmt.Errorf("write %s: %w", path, port.ErrStale)
		}
	default:
		return domain.Fingerprint{}, err
	}

	if err := parents(root, filepath.Dir(name), path); err != nil {
		return domain.Fingerprint{}, err
	}
	written, err := replace(root, name, content, mode)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	written.Path = path
	return written, nil
}

// nameMax is how many bytes one component of a path may be. It is 255 on every
// filesystem a vault is kept on.
const nameMax = 255

// beside is the pattern a temporary file next to a target is created under. The
// name is cut on a rune boundary, leaving room for the leading dot and for the
// digits that go where the star is; the rename lands on the full name.
func beside(name string) string {
	const room = len(".") + len(".") + 10
	for len(name)+room > nameMax {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return "." + name + ".*"
}

// parents puts the folders above a name there, and says a file standing where
// one of them would go in the words a caller acts on.
func parents(root *os.Root, dir, path string) error {
	err := root.MkdirAll(dir, 0o755)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("make %s: %w", path, port.ErrOccupied)
	}
	return err
}

// temporary is a file created beside a target, under the vault's own handle.
// It is what os.CreateTemp is, for a root: a name nobody else holds, taken by
// creating it and not by looking first.
func temporary(root *os.Root, dir, pattern string) (*os.File, string, error) {
	prefix, suffix, _ := strings.Cut(pattern, "*")
	for range 10_000 {
		name := filepath.Join(dir, prefix+strconv.FormatUint(uint64(rand.Uint32()), 10)+suffix)
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return file, name, nil
	}
	return nil, "", fmt.Errorf("%s: no free name beside it", dir)
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
func replace(root *os.Root, target string, content []byte, mode fs.FileMode) (domain.Fingerprint, error) {
	dir := filepath.Dir(target)
	tmp, at, err := temporary(root, dir, beside(filepath.Base(target)))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	defer root.Remove(at)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	info, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	// Changing the mode moves no modification time. It is asked of the open
	// file, which is this one and can be no other.
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return domain.Fingerprint{}, err
	}
	if err := tmp.Close(); err != nil {
		return domain.Fingerprint{}, err
	}
	if err := rename(root, at, target); err != nil {
		return domain.Fingerprint{}, err
	}
	written := domain.Fingerprint{Size: info.Size(), ModTime: info.ModTime()}
	return written, settle(root, dir)
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
func settle(root *os.Root, dir string) error {
	folder, err := root.Open(dir)
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
	// A folder does not go inside itself: the destination is a place the folder
	// itself holds.
	if strings.HasPrefix(target, source+string(filepath.Separator)) {
		return fmt.Errorf("move %s to %s: %w", from, to, port.ErrOccupied)
	}
	root, arrives, err := w.beneath(target)
	if err != nil {
		return err
	}
	defer root.Close()
	leaves, err := w.named(source)
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
			return fmt.Errorf("move %s to %s: %w", from, to, port.ErrOccupied)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if err := parents(root, filepath.Dir(arrives), to); err != nil {
		return err
	}
	return rename(root, leaves, arrives)
}

func (w *VaultWriter) MakeFolder(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.reach(path)
	if err != nil {
		return err
	}
	root, name, err := w.beneath(target)
	if err != nil {
		return err
	}
	defer root.Close()
	switch info, err := root.Stat(name); {
	case err == nil && !info.IsDir():
		return fmt.Errorf("make %s: %w", path, port.ErrOccupied)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return parents(root, name, path)
}

func (w *VaultWriter) Remove(ctx context.Context, path string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.reach(path)
	if err != nil {
		return err
	}
	root, name, err := w.beneath(target)
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

// beneath is the vault as a handle, and a place in it as a name that handle
// takes. Every step of a write is made through the handle, so a folder swapped
// for a link while the write is on its way is refused by the machine itself and
// not by a rule read a moment before. The caller closes the handle.
func (w *VaultWriter) beneath(target string) (*os.Root, string, error) {
	name, err := w.named(target)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(w.root)
	if err != nil {
		return nil, "", err
	}
	return root, name, nil
}

// named is a place on this machine as a name under the vault's root.
func (w *VaultWriter) named(target string) (string, error) {
	name, err := filepath.Rel(w.root, target)
	if err != nil || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: %w", target, ErrOutside)
	}
	return name, nil
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
		if name := pathpkg.Base(dir); w.opts.isService(name) || strings.EqualFold(name, TrashDir) {
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
		if w.opts.isService(pathpkg.Base(dir)) {
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
		if w.opts.isService(pathpkg.Base(dir)) {
			return false
		}
	}
	return true
}

// Root is where this vault is on disk.
func (w *VaultWriter) Root() string { return w.root }

// VaultWriters opens vaults for writing.
type VaultWriters struct{ Options Options }

func (w VaultWriters) Open(v domain.Vault) (port.VaultWriter, error) {
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
	root, name, err := w.beneath(target)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := parents(root, filepath.Dir(name), path); err != nil {
		return err
	}

	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
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
	return settle(root, filepath.Dir(name))
}

// Bring copies a file from this machine into the vault.
//
// The bytes are streamed, so a recording or an archive crosses in whatever room
// the machine has. They land beside the destination and are renamed over it, so
// the watcher reports the file once and reports it whole.
func (w *VaultWriter) Bring(ctx context.Context, path string, content io.Reader) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target, err := w.reach(path)
	if err != nil {
		return err
	}
	root, name, err := w.beneath(target)
	if err != nil {
		return err
	}
	defer root.Close()
	switch _, err := root.Lstat(name); {
	case err == nil:
		return fmt.Errorf("bring %s: %w", path, port.ErrOccupied)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := parents(root, filepath.Dir(name), path); err != nil {
		return err
	}
	return arrive(root, name, content)
}

// arrive streams content beside the target and renames it over the top, by the
// same rules replace writes bytes it already holds.
func arrive(root *os.Root, target string, content io.Reader) error {
	dir := filepath.Dir(target)
	tmp, at, err := temporary(root, dir, beside(filepath.Base(target)))
	if err != nil {
		return err
	}
	defer root.Remove(at)

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
	if err := rename(root, at, target); err != nil {
		return err
	}
	return settle(root, dir)
}
