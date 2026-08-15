package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultReader reads one vault from disk.
type VaultReader struct {
	root string
	opts Options
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
	return &VaultReader{root: abs, opts: opts}, nil
}

func (s *VaultReader) Root() string { return s.root }

// Walk reports every markdown file in the vault.
//
// The service folder is skipped because it is not vault content, and so is any
// directory whose name begins with a dot: those hold tool state rather than
// notes the user wrote. The service folder is named separately because its name
// is a setting, and a name without a leading dot must still be skipped.
func (s *VaultReader) Walk(ctx context.Context, fn func(domain.FileRef) error) error {
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
		name := d.Name()
		if d.IsDir() {
			if p == s.root {
				return nil
			}
			if name == s.opts.serviceDir() || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !s.opts.isNote(name) {
			return nil
		}
		info, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(s.root, p)
		if err != nil {
			return err
		}
		return fn(domain.FileRef{
			Path:  filepath.ToSlash(rel),
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
