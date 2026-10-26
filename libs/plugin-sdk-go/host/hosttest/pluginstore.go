package hosttest

import (
	"bytes"
	"context"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Store is the plugin's folder in the store of a vault, held in memory. It answers as the host does:
// a vault the host does not show, does not know, or whose folder is gone refuses it.
type Store struct {
	vault *Vault
}

var _ host.Store = (*Store)(nil)

// folder is what the plugin's folder of a vault holds.
type folder struct {
	files map[string][]byte
	errs  map[string]error
}

// Store is the plugin's folder in the store of the vault.
func (v *Vault) Store() *Store { return &Store{vault: v} }

// getFolder is the vault's folder, made on first use. The caller holds the world's lock.
func (v *Vault) getFolder() *folder {
	if v.folder == nil {
		v.folder = &folder{files: map[string][]byte{}, errs: map[string]error{}}
	}
	return v.folder
}

// Refuse makes every call on the file fail with err, as a file the host may not open
// does. A nil err lets it be worked again.
func (s *Store) Refuse(name string, err error) {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	if err == nil {
		delete(s.vault.getFolder().errs, name)
		return
	}
	s.vault.getFolder().errs[name] = err
}

// File puts a file in the folder, or replaces it, as another program would.
func (s *Store) File(name string, content []byte) {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	s.vault.getFolder().files[name] = bytes.Clone(content)
}

func (s *Store) With(id domain.VaultID) host.Store {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	return &Store{vault: s.vault.find(id)}
}

// reach refuses a call on a vault the host does not show, does not know, or whose folder is gone.
// The caller holds the world's lock.
func (s *Store) reach(ctx context.Context) (*folder, error) {
	if err := s.vault.gate(ctx); err != nil {
		return nil, err
	}
	if err := s.vault.reach(); err != nil {
		return nil, err
	}
	return s.vault.getFolder(), nil
}

// open checks a call on the name and returns the folder. The caller holds the world's lock.
func (s *Store) open(ctx context.Context, name string) (*folder, error) {
	f, err := s.reach(ctx)
	if err != nil {
		return nil, err
	}
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return nil, host.ErrOutside
	}
	return f, f.errs[name]
}

func (s *Store) Read(ctx context.Context, name string) ([]byte, error) {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.open(ctx, name)
	if err != nil {
		return nil, err
	}
	content, there := f.files[name]
	if !there {
		return nil, host.ErrNotExist
	}
	return bytes.Clone(content), nil
}

func (s *Store) Append(ctx context.Context, name string, content []byte) error {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.open(ctx, name)
	if err != nil {
		return err
	}
	f.files[name] = append(f.files[name], content...)
	return nil
}

func (s *Store) List(ctx context.Context) ([]host.StoreFile, error) {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.reach(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]host.StoreFile, 0, len(f.files))
	for name, content := range f.files {
		out = append(out, host.StoreFile{Name: name, Size: int64(len(content))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) Write(ctx context.Context, name string, content []byte) error {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.open(ctx, name)
	if err != nil {
		return err
	}
	f.files[name] = bytes.Clone(content)
	return nil
}

func (s *Store) Delete(ctx context.Context, name string) error {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.open(ctx, name)
	if err != nil {
		return err
	}
	delete(f.files, name)
	return nil
}

func (s *Store) Rename(ctx context.Context, from, to string) error {
	s.vault.world.mu.Lock()
	defer s.vault.world.mu.Unlock()
	f, err := s.open(ctx, from)
	if err != nil {
		return err
	}
	if _, err := s.open(ctx, to); err != nil {
		return err
	}
	content, there := f.files[from]
	if !there {
		return host.ErrNotExist
	}
	f.files[to] = content
	delete(f.files, from)
	return nil
}
