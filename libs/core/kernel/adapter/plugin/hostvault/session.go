package hostvault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Session chooses the vault a call works, follows it, and tells the window that a call changed a file.
// Each other group of the vault reads the vault of a call through it.
type Session struct {
	Current    func() (domain.Vault, bool)
	Vaults     func() ([]domain.Vault, error)
	Find       func(id domain.VaultID) (domain.Vault, error)
	IndexState func(ctx context.Context, v domain.Vault) (plugin.IndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Watcher    func(ctx context.Context, v domain.Vault) (changes <-chan []string, lost <-chan struct{}, err error)
	Written    func()

	rebuild func(found domain.Vault) *vault.Access
}

func (s Session) showing(found domain.Vault) Session {
	return Session{
		Current: func() (domain.Vault, bool) { return found, true }, Vaults: s.Vaults, Find: s.Find,
		IndexState: s.IndexState, Moved: s.Moved, Watcher: s.Watcher, Written: s.Written,
	}
}

// GetVault returns the vault the window shows, if it shows one.
func (s *Session) GetVault() (domain.Vault, bool) {
	if s.Current == nil {
		return domain.Vault{}, false
	}
	return s.Current()
}

// GetVaults returns the vaults of the installation, or the vault shown when it lists none.
func (s *Session) GetVaults() ([]domain.Vault, error) {
	if s.Vaults != nil {
		return s.Vaults()
	}
	if v, showing := s.GetVault(); showing {
		return []domain.Vault{v}, nil
	}
	return nil, nil
}

// With returns the capabilities over the vault with the identity.
func (s *Session) With(id domain.VaultID) (*vault.Access, error) {
	found, err := s.find(id)
	if err != nil {
		return nil, err
	}
	return s.rebuild(found), nil
}

func (s *Session) find(id domain.VaultID) (domain.Vault, error) {
	if s.Find != nil {
		return s.Find(id)
	}
	if v, showing := s.GetVault(); showing && v.ID == id {
		return v, nil
	}
	return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
}

// EnsureIndexed reports whether the index of the vault is ready.
func (s *Session) EnsureIndexed(ctx context.Context) (plugin.IndexState, error) {
	v, err := s.open()
	if err != nil {
		return plugin.IndexState{}, err
	}
	if s.IndexState == nil {
		return plugin.IndexState{Status: plugin.Indexed}, nil
	}
	return s.IndexState(ctx, v)
}

// WatchMoves reports each time the window moves.
func (s *Session) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	if s.Moved == nil {
		return nil, fmt.Errorf("watch moves: %w", domain.ErrUnavailable)
	}
	return s.Moved(ctx)
}

// Watch reports the paths that change in the vault, and when the watch is lost.
func (s *Session) Watch(ctx context.Context) (<-chan []string, <-chan struct{}, error) {
	v, err := s.open()
	if err != nil {
		return nil, nil, err
	}
	if s.Watcher == nil {
		return nil, nil, fmt.Errorf("watch %s: %w", v.ID, domain.ErrUnavailable)
	}
	return s.Watcher(ctx, v)
}

func (s *Session) open() (domain.Vault, error) {
	v, showing := s.GetVault()
	if !showing {
		return domain.Vault{}, plugin.ErrNoVault
	}
	return v, nil
}

func (s *Session) notifyFileWritten() {
	if s.Written != nil {
		s.Written()
	}
}

func (s *Session) notifyAfterWrite(err error) error {
	if err == nil || errors.Is(err, edit.ErrIndexOutdated) {
		s.notifyFileWritten()
	}
	return err
}
