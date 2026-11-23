package hostvault

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

// Session chooses the vault a call works, follows it, and tells the window that a call changed a file.
// Each other group of the vault reads the vault of a call through it.
type Session struct {
	GetCurrent func() (domain.Vault, bool)
	ListVaults func() ([]domain.Vault, error)
	Find       func(id domain.VaultID) (domain.Vault, error)
	IndexState func(ctx context.Context, v domain.Vault) (plugin.IndexState, error)
	WatchMoves func(ctx context.Context) (<-chan struct{}, error)
	indexing.WriteWake

	rebuild func(found domain.Vault) *vault.Access
}

func (s *Session) selectVault(found domain.Vault) Session {
	return Session{
		GetCurrent: func() (domain.Vault, bool) { return found, true }, ListVaults: s.ListVaults, Find: s.Find,
		IndexState: s.IndexState, WatchMoves: s.WatchMoves, WriteWake: s.WriteWake,
	}
}

// GetVault returns the vault the window shows, if it shows one.
func (s *Session) GetVault() (domain.Vault, bool) {
	if s.GetCurrent == nil {
		return domain.Vault{}, false
	}
	return s.GetCurrent()
}

// GetVaults returns the vaults of the installation, or the vault shown when it lists none.
func (s *Session) GetVaults() ([]domain.Vault, error) {
	if s.ListVaults != nil {
		return s.ListVaults()
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

// WatchIndexChanges reports each time the window moves.
func (s *Session) WatchIndexChanges(ctx context.Context) (<-chan struct{}, error) {
	if s.WatchMoves == nil {
		return nil, fmt.Errorf("watch moves: %w", domain.ErrUnavailable)
	}
	return s.WatchMoves(ctx)
}

func (s *Session) open() (domain.Vault, error) {
	v, showing := s.GetVault()
	if !showing {
		return domain.Vault{}, plugin.ErrNoVault
	}
	return v, nil
}
