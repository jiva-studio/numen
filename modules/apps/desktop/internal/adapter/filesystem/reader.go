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
	"strings"

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

// Walk reports every note in the vault.
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
		if !s.opts.isNote(d.Name()) || ignored.MatchesPath(rel) {
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
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
		})
	})
}

func (s *VaultReader) Read(ctx context.Context, path string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return os.ReadFile(filepath.Join(s.root, filepath.FromSlash(path)))
}

// Stat answers the same question about one path that Walk answers about all of
// them, and by the same rules: a path the vault ignores does not exist.
func (s *VaultReader) Stat(ctx context.Context, path string) (domain.FileRef, error) {
	if ctx.Err() != nil {
		return domain.FileRef{}, ctx.Err()
	}
	if !s.holds(path) {
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
		Size:  info.Size(),
		MTime: info.ModTime().UnixNano(),
	}, nil
}

// holds reports whether a path inside this vault is one a walk would report.
func (s *VaultReader) holds(path string) bool {
	if path == "" || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
		return false
	}
	if !s.opts.isNote(pathpkg.Base(path)) {
		return false
	}
	if s.ignored.MatchesPath(path) {
		return false
	}
	for dir := pathpkg.Dir(path); dir != "." && dir != "/"; dir = pathpkg.Dir(dir) {
		if pathpkg.Base(dir) == s.opts.serviceDir() || s.ignored.MatchesPath(dir+"/") {
			return false
		}
	}
	return true
}
