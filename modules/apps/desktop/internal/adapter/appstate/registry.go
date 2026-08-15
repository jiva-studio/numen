// Package registry stores which vaults exist. This is application state: not
// derivable from any vault, and never stored inside one.
//
// JSON rather than a database, deliberately — it is a handful of entries whose
// most important moment is when the application will not start, and a file a
// human can open and fix beats a database that needs the application to read it.
package appstate

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

type file struct {
	V      int            `json:"v"`
	Vaults []domain.Vault `json:"vaults"`
}

type VaultRegistry struct{ path string }

// Open uses the platform's configuration location.
func Open() (*VaultRegistry, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return At(filepath.Join(dir, "numen", "vaults.json")), nil
}

// At is Open with an explicit path, so that a test does not touch the machine's
// real configuration.
func At(path string) *VaultRegistry { return &VaultRegistry{path: path} }

func (r *VaultRegistry) Path() string { return r.path }

func (r *VaultRegistry) load() (file, error) {
	raw, err := os.ReadFile(r.path)
	if errors.Is(err, fs.ErrNotExist) {
		return file{V: 1}, nil
	}
	if err != nil {
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return file{}, err
	}
	if f.V == 0 {
		f.V = 1
	}
	return f, nil
}

func (r *VaultRegistry) save(f file) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	// Written through a temporary file: a half-written registry is the one
	// thing here that costs the user manual work to recover from.
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func (r *VaultRegistry) List() ([]domain.Vault, error) {
	f, err := r.load()
	return f.Vaults, err
}

// Add records a vault. Re-adding the same identity updates its path, which is
// how a moved vault is recognised: the identity travels with the folder, the
// registry only remembers where it was last seen.
func (r *VaultRegistry) Add(v domain.Vault) error {
	f, err := r.load()
	if err != nil {
		return err
	}
	for i := range f.Vaults {
		if f.Vaults[i].ID == v.ID {
			f.Vaults[i] = v
			return r.save(f)
		}
	}
	f.Vaults = append(f.Vaults, v)
	return r.save(f)
}

// Find resolves what the user typed: an identity, a name, or a path.
func (r *VaultRegistry) Find(nameOrPath string) (domain.Vault, bool, error) {
	f, err := r.load()
	if err != nil {
		return domain.Vault{}, false, err
	}
	abs, _ := filepath.Abs(nameOrPath)
	for _, v := range f.Vaults {
		if v.ID == nameOrPath || strings.EqualFold(v.Name, nameOrPath) || v.Path == abs {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}
