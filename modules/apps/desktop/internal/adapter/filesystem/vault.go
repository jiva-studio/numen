// Package vaultfs reads a vault from the filesystem and manages the identity a
// vault carries inside itself.
package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// DefaultServiceDir is the one folder the application writes into a vault. It is
// named after the application so that it cannot collide with another tool that
// had the same idea — a vault opened by both ends up with .numen beside
// .obsidian.
const DefaultServiceDir = ".numen"

const configName = "config.json"

// Config is what a vault knows about itself. It is deliberately tiny: an
// identity and a format version, nothing that could be recomputed.
type Config struct {
	V  int    `json:"v"`
	ID string `json:"id"`
}

// Source reads one vault.
type VaultReader struct {
	root       string
	serviceDir string
}

// Open prepares a vault for reading. It does not write anything: scanning a
// folder and adding a vault are different acts, and only the second may touch
// the user's files.
func Open(root, serviceDir string) (*VaultReader, error) {
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
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	return &VaultReader{root: abs, serviceDir: serviceDir}, nil
}

func (s *VaultReader) Root() string { return s.root }

// Walk reports every markdown file in the vault.
//
// Directories whose name begins with a dot are skipped whole. The service
// folder is skipped because it is not vault content; the rest are
// skipped because a hidden directory at the top of a vault belongs to some
// tool — .git, .obsidian — and indexing another tool's state as if the user had
// written it is worse than missing a note nobody keeps there.
func (s *VaultReader) Walk(ctx context.Context, fn func(domain.FileRef) error) error {
	return filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
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
			if strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(name), ".md") {
			return nil
		}
		info, err := d.Info()
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

// ErrNotAVault is returned when a folder has no identity, which means it has
// never been added to the application.
var ErrNotAVault = errors.New("no vault configuration in this folder")

// ReadConfig returns the identity the vault carries.
func ReadConfig(root, serviceDir string) (Config, error) {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	raw, err := os.ReadFile(filepath.Join(root, serviceDir, configName))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, ErrNotAVault
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("vault configuration is unreadable: %w", err)
	}
	if !ulid.Valid(c.ID) {
		return Config{}, fmt.Errorf("vault configuration holds %q: %w", c.ID, ulid.ErrInvalid)
	}
	return c, nil
}

// Initialize gives a folder an identity, which is the moment it becomes a vault.
// This is the one write the application makes into a vault without being asked
// to edit something, and it happens because the user added the vault.
//
// An existing configuration is returned untouched rather than replaced: the
// identity is what every row in the index points at.
func Initialize(root, serviceDir string, now time.Time) (Config, error) {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	if c, err := ReadConfig(root, serviceDir); err == nil {
		return c, nil
	} else if !errors.Is(err, ErrNotAVault) {
		return Config{}, err
	}

	id, err := ulid.New(now)
	if err != nil {
		return Config{}, err
	}
	c := Config{V: 1, ID: id}

	dir := filepath.Join(root, serviceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Config{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, configName), raw, 0o644); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Readers opens vaults from the filesystem. It is what a use case is handed
// when the vault it works on is chosen while it runs.
type Readers struct{ ServiceDir string }

func (r Readers) Open(v domain.Vault) (port.VaultReader, error) {
	return Open(v.Path, r.ServiceDir)
}

// Identity gives folders their identity and answers whether one can be read at
// all.
type Identity struct{ ServiceDir string }

func (i Identity) Readable(root string) error {
	_, err := Open(root, i.ServiceDir)
	return err
}

func (i Identity) Ensure(root string, at time.Time) (string, error) {
	cfg, err := Initialize(root, i.ServiceDir, at)
	if err != nil {
		return "", err
	}
	return cfg.ID, nil
}
