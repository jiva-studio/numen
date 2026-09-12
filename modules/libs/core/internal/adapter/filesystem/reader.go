package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultReader reads one vault from disk.
type VaultReader struct {
	root string
	opts Options
	// ignored is compiled once: it is asked of every path of every event, and
	// of every path component.
	ignored *ignoring
}

// Open prepares a vault for reading. It does not write anything: looking at a
// folder and adding a vault are different acts, and only the second may touch
// the user's files.
func Open(root string, opts Options) (*VaultReader, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// The vault is where the links lead. The paths the operating system reports
	// changes at are resolved, and they are named against this.
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
	// What the vault says not to look at is the vault's to say. A folder that
	// has never been added has nothing to say, and reads as the default.
	if cfg, err := ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultReader{root: abs, opts: opts, ignored: opts.compileIgnoring()}, nil
}

func (s *VaultReader) Root() string { return s.root }

// Walk reports every source in the vault, each saying which kind it is. A file
// of no kind the application reads is not reported at all.
//
// What the vault's ignore rules name is skipped, and so is the service folder.
// The service folder is named separately because its name is a setting, and a
// name without a leading dot must still be skipped.
func (s *VaultReader) Walk(ctx context.Context, fn func(domain.Fingerprint) error) error {
	ignored := s.ignored
	return filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			// The vault is edited while it is walked. A file or folder that has
			// gone between reading the directory and reaching it is not an
			// error: the next scan sees whatever it became.
			return nil
		}
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, err := filepath.Rel(s.root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if p == s.root {
				return nil
			}
			if s.isSkipped(rel, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		kind, ok := s.opts.kind(d.Name())
		if !ok || ignored.MatchesPath(rel) {
			return nil
		}
		found, err := info(p, d)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		// A source is a regular file. A device, a socket or a FIFO carries no
		// bytes to read whatever it is named.
		if !found.Mode().IsRegular() {
			return nil
		}
		return fn(domain.Fingerprint{
			Path:    rel,
			Kind:    kind,
			Size:    found.Size(),
			ModTime: found.ModTime(),
		})
	})
}

// List reports what one folder holds, without descending. Every file is
// reported whether anything reads it or not, and so is every folder under it;
// what the vault's ignore rules name and the service folder are left out.
//
// Folders come first, then files, each by name compared without regard to
// case. That is the order to draw them in.
//
// A path holding a file holds no folder, and is answered as a folder that is
// not there.
func (s *VaultReader) List(ctx context.Context, folder string) ([]domain.Entry, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target := s.root
	if folder != "" {
		var err error
		if target, err = inside(s.root, folder, s.opts.serviceDir()); err != nil {
			return nil, err
		}
		folder = pathpkg.Clean(filepath.ToSlash(folder))
	}
	switch info, err := os.Stat(target); {
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("list %s: %w", folder, fs.ErrNotExist)
	}
	read, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.Entry, 0, len(read))
	for _, d := range read {
		rel := pathpkg.Join(folder, d.Name())
		var kind domain.SourceKind
		if d.IsDir() {
			if s.opts.isService(d.Name()) || s.ignored.MatchesPath(rel+"/") {
				continue
			}
		} else {
			if s.ignored.MatchesPath(rel) {
				continue
			}
			if found, err := info(filepath.Join(target, d.Name()), d); err == nil && found.Mode().IsRegular() {
				kind, _ = s.opts.kind(d.Name())
			}
		}
		entries = append(entries, domain.Entry{
			Path:     rel,
			Name:     d.Name(),
			IsFolder: d.IsDir(),
			Kind:     kind,
		})
	}
	slices.SortFunc(entries, inOrder)
	return entries, nil
}

// info is what a walk knows about one entry, with a link followed: a note kept
// as a link to a file elsewhere in the vault is that file. A link nothing is at
// the end of, or one that leads round in a circle, is nothing to read.
func info(path string, d fs.DirEntry) (fs.FileInfo, error) {
	if d.Type()&fs.ModeSymlink == 0 {
		return d.Info()
	}
	found, err := os.Stat(path)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	return found, nil
}

// inOrder is folders before files, and names compared without regard to case.
// Two names differing only in case keep a settled order of their own.
func inOrder(a, b domain.Entry) int {
	if a.IsFolder != b.IsFolder {
		if a.IsFolder {
			return -1
		}
		return 1
	}
	if by := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); by != 0 {
		return by
	}
	return strings.Compare(a.Name, b.Name)
}

func (s *VaultReader) Read(ctx context.Context, path string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target, err := inside(s.root, path, s.opts.serviceDir())
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if err := readable(path, info); err != nil {
		return nil, err
	}
	if err := s.checkSize(path, info); err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

// checkSize holds a note to the most one is read whole at. A book and a recording
// are read a part at a time by a caller that knows how large the thing it is
// reading is.
func (s *VaultReader) checkSize(path string, info fs.FileInfo) error {
	bound := s.opts.maxNoteBytes()
	if !s.opts.isNote(pathpkg.Base(path)) || info.Size() <= bound {
		return nil
	}
	return fmt.Errorf("%s is %d bytes, and %d is the most a note is read at: %w",
		path, info.Size(), bound, ErrNotANote)
}

// Open is one file, to read a part of. The path is checked by the same rule
// Read checks it by, and what comes back is the file itself.
func (s *VaultReader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target, err := inside(s.root, path, s.opts.serviceDir())
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if err := readable(path, info); err != nil {
		return nil, err
	}
	return os.Open(target)
}

// readable holds a path to something the operating system hands bytes over for
// without waiting: a device, a socket or a FIFO is not a note. A folder is
// answered by whichever call was going to open it.
func readable(path string, info fs.FileInfo) error {
	if info.Mode().IsRegular() || info.IsDir() {
		return nil
	}
	return fmt.Errorf("%s: %w", path, ErrNotANote)
}

// Stat answers the same question about one path that Walk answers about all of
// them, and by the same rules: a path the vault ignores holds no note. Where
// something is there all the same, the answer names it as a file the vault
// leaves alone.
func (s *VaultReader) Stat(ctx context.Context, path string) (domain.Fingerprint, error) {
	if ctx.Err() != nil {
		return domain.Fingerprint{}, ctx.Err()
	}
	kind, held := s.holds(path)
	if !held {
		return domain.Fingerprint{}, s.leftAlone(path)
	}
	info, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(path)))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	if !info.Mode().IsRegular() {
		return domain.Fingerprint{}, s.leftAlone(path)
	}
	return domain.Fingerprint{
		Path:    path,
		Kind:    kind,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// leftAlone is what a path the vault does not hold as a note is answered with.
// Something at that path is ErrNotANote; nothing at it is fs.ErrNotExist.
//
// It costs a look at the file's metadata and never its bytes, which is what a
// caller deciding whether to open a 400 MB export has to be able to ask.
func (s *VaultReader) leftAlone(path string) error {
	target, err := inside(s.root, path, s.opts.serviceDir())
	if err != nil {
		return fs.ErrNotExist
	}
	if _, err := os.Lstat(target); err != nil {
		return fs.ErrNotExist
	}
	return fmt.Errorf("%s: %w", path, ErrNotANote)
}

// isSkipped says whether a walk stops at a folder and does not descend. The walk
// and the watcher both ask it, so neither of them looks where the other does
// not.
func (s *VaultReader) isSkipped(path, name string) bool {
	return s.opts.isService(name) || s.ignored.MatchesPath(path+"/")
}

// relative names a path the way the vault does. Anything outside it is not the
// vault's to answer for.
func (s *VaultReader) relative(absolute string) (path string, inside bool) {
	rel, err := filepath.Rel(s.root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// holds reports which kind of source a path inside this vault is, and whether a
// walk would report it at all. The walk and the watcher both ask it, so the two
// agree about what the vault holds.
func (s *VaultReader) holds(path string) (domain.SourceKind, bool) {
	if _, err := inside(s.root, path, s.opts.serviceDir()); err != nil {
		return "", false
	}
	kind, ok := s.opts.kind(pathpkg.Base(path))
	if !ok {
		return "", false
	}
	if s.ignored.MatchesPath(path) {
		return "", false
	}
	for dir := pathpkg.Dir(path); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if s.opts.isService(pathpkg.Base(dir)) || s.ignored.MatchesPath(dir+"/") {
			return "", false
		}
	}
	return kind, true
}
