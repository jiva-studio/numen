// Package keyvault is the key store of the identity on a phone, over the vault of keys the platform keeps.
package keyvault

import (
	"context"
	"errors"
)

// MaxValueBytes is the longest key the store takes, as the keychain of the desktop.
const MaxValueBytes = 2560

// Vault is what the platform keeps keys in. A name with no key reads as the empty string and Has is false.
type Vault interface {
	Get(name string) (string, error)
	Has(name string) bool
	Set(name, value string) error
	Delete(name string) error
}

// Store answers the key store port of the core over a Vault.
type Store struct{ vault Vault }

// New returns the store over the vault.
func New(vault Vault) *Store { return &Store{vault: vault} }

// GetKey returns the key under the name, or plugin.ErrNoKey.
func (s *Store) GetKey(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

// SetKey stores the key under the name and replaces the one there.
func (s *Store) SetKey(context.Context, string, string) error { return errors.New("not implemented") }

// DeleteKey removes the key under the name, or returns plugin.ErrNoKey.
func (s *Store) DeleteKey(context.Context, string) error { return errors.New("not implemented") }
