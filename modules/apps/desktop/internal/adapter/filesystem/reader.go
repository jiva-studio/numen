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
// them, and by the same rules: a path the vault ignores does not exist.
func (s *VaultReader) Stat(ctx context.Context, path string) (domain.FileRef, error) {
	if ctx.Err() != nil {
		return domain.FileRef{}, ctx.Err()
	}
	kind, held := s.holds(path)
	if !held {
		return domain.FileRef{}, fs.ErrNotExist
	}
	info, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(path)))
	if err != nil {
		return domain.FileRef{}, err
	}
	if info.IsDir() {
		return domain.FileRef{}, fs.ErrNotExist
	}
	return domain.FileRef{
		Path:  path,
		Kind:  kind,
		Size:  info.Size(),
		MTime: info.ModTime().UnixNano(),
	}, nil
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
