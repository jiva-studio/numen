package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultReader reads one vault from disk.
type VaultReader struct {
	root string
	opts Options
	// ignored is compiled once: it is asked of every path of every event, and
	// of every path component.
	ignored *ignore.GitIgnore
}

// Open prepares a vault for reading. It does not write anything: looking at a
// folder and adding a vault are different acts, and only the second may touch
// the user's files.
func Open(root string, opts Options) (*VaultReader, error) {
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
	// What the vault says not to look at is the vault's to say. A folder that
	// has never been added has nothing to say, and reads as the default.
	if cfg, err := ReadConfig(abs, opts.ServiceDir); err == nil && len(cfg.Ignore) > 0 {
		opts.Ignore = cfg.Ignore
	}
	return &VaultReader{root: abs, opts: opts, ignored: opts.ignored()}, nil
}

func (s *VaultReader) Root() string { return s.root }

// Walk reports every source in the vault, each saying which kind it is. A file
// of no kind the application reads is not reported at all.
//
// What the vault's ignore rules name is skipped, and so is the service folder.
// The service folder is named separately because its name is a setting, and a
// name without a leading dot must still be skipped.
func (s *VaultReader) Walk(ctx context.Context, fn func(domain.FileRef) error) error {
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
			if d.Name() == s.opts.serviceDir() || ignored.MatchesPath(rel+"/") {
				return fs.SkipDir
			}
			return nil
		}
		kind, ok := s.opts.kind(d.Name())
		if !ok || ignored.MatchesPath(rel) {
			return nil
		}
		info, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return fn(domain.FileRef{
			Path:  rel,
			Kind:  kind,
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
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
			if d.Name() == s.opts.serviceDir() || s.ignored.MatchesPath(rel+"/") {
				continue
			}
		} else {
			if s.ignored.MatchesPath(rel) {
				continue
			}
			kind, _ = s.opts.kind(d.Name())
		}
		entries = append(entries, domain.Entry{
			Path:   rel,
			Name:   d.Name(),
			Folder: d.IsDir(),
			Kind:   kind,
		})
	}
	slices.SortFunc(entries, inOrder)
	return entries, nil
}

// inOrder is folders before files, and names compared without regard to case.
// Two names differing only in case keep a settled order of their own.
func inOrder(a, b domain.Entry) int {
	if a.Folder != b.Folder {
		if a.Folder {
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
	return os.ReadFile(target)
}

// Stat answers the same question about one path that Walk answers about all of
// them, and by the same rules: a path the vault ignores holds no note. Where
// something is there all the same, the answer names it as a file the vault
// leaves alone.
func (s *VaultReader) Stat(ctx context.Context, path string) (domain.FileRef, error) {
	if ctx.Err() != nil {
		return domain.FileRef{}, ctx.Err()
	}
	kind, held := s.holds(path)
	if !held {
		return domain.FileRef{}, s.leftAlone(path)
	}
	info, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(path)))
	if err != nil {
		return domain.FileRef{}, err
	}
	if info.IsDir() {
		return domain.FileRef{}, s.leftAlone(path)
	}
	return domain.FileRef{
		Path:  path,
		Kind:  kind,
		Size:  info.Size(),
		MTime: info.ModTime().UnixNano(),
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
		if pathpkg.Base(dir) == s.opts.serviceDir() || s.ignored.MatchesPath(dir+"/") {
			return "", false
		}
	}
	return kind, true
}
