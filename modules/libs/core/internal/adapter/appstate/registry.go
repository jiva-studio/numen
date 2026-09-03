package appstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultRegistry is the list of vaults, kept in one file that every write
// rewrites whole. The mutex is held from reading that file to writing it back,
// so two writes in this process are one after the other.
type VaultRegistry struct {
	mu   sync.Mutex
	path string
}

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

func (r *VaultRegistry) load() (registryFile, error) {
	raw, err := os.ReadFile(r.path)
	if errors.Is(err, fs.ErrNotExist) {
		return registryFile{V: 1}, nil
	}
	if err != nil {
		return registryFile{}, err
	}
	var f registryFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return registryFile{}, err
	}
	if f.V == 0 {
		f.V = 1
	}
	return f, nil
}

func (r *VaultRegistry) save(f registryFile) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return replace(r.path, raw)
}

// replace writes the registry beside itself and renames it over the top: a
// half-written registry is the one thing here that costs the user manual work
// to recover from.
//
// Every write gets a temporary file of its own, so two processes writing the
// list at once each rename a whole one. The bytes are flushed before the
// rename and the folder after it, so a machine that loses power has either the
// old list or the new one.
func replace(path string, content []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	return settle(dir)
}

// settle flushes the folder the rename was recorded in. Flushing the file is
// what keeps its contents; flushing the folder is what keeps the rename.
//
// Not every filesystem lets a folder be opened for this, and the ones that
// refuse are the ones that did not need it.
func settle(dir string) error {
	folder, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer folder.Close()
	_ = folder.Sync()
	return nil
}

func (r *VaultRegistry) All() ([]domain.Vault, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.load()
	return f.Vaults, err
}

// Save records a vault. Re-adding the same identity updates its path, which is
// how a moved vault is recognised: the identity travels with the folder, the
// registry only remembers where it was last seen.
func (r *VaultRegistry) Save(v domain.Vault) error {
	r.mu.Lock()
	defer r.mu.Unlock()
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

// Remove takes a vault off the list. The folder and the identity inside it stay
// as they are. An identity the list does not hold is already off it.
func (r *VaultRegistry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.load()
	if err != nil {
		return err
	}
	at := slices.IndexFunc(f.Vaults, func(v domain.Vault) bool { return v.ID == id })
	if at < 0 {
		return nil
	}
	f.Vaults = slices.Delete(f.Vaults, at, at+1)
	if f.Last == id {
		f.Last = ""
	}
	return r.save(f)
}

// Opened records the vault a window is showing. Recording the vault already
// recorded writes nothing.
func (r *VaultRegistry) Opened(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.load()
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(f.Vaults, func(v domain.Vault) bool { return v.ID == id }) {
		return fmt.Errorf("no vault on the list carries the identity %s", id)
	}
	if f.Last == id {
		return nil
	}
	f.Last = id
	return r.save(f)
}

// Last is the vault opened most recently. Nothing has been opened until a vault
// is recorded, and an identity that names nothing on the list answers the same.
func (r *VaultRegistry) Last() (domain.Vault, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.load()
	if err != nil {
		return domain.Vault{}, false, err
	}
	for _, v := range f.Vaults {
		if v.ID == f.Last && f.Last != "" {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}

// Find resolves what the user typed: an identity, a name, or a path. Nothing
// typed names no vault: an empty path is the folder this process was started
// in.
func (r *VaultRegistry) Find(nameOrPath string) (domain.Vault, bool, error) {
	if nameOrPath == "" {
		return domain.Vault{}, false, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.load()
	if err != nil {
		return domain.Vault{}, false, err
	}
	abs, _ := filepath.Abs(nameOrPath)
	for _, v := range f.Vaults {
		if v.ID == nameOrPath || domain.FoldName(v.Name) == domain.FoldName(nameOrPath) || v.Path == abs {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}
