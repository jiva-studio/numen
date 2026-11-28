//go:build numen_debug

package keychain

import "github.com/zalando/go-keyring"

// NewDebugFileStore opens a plaintext credentials file for an isolated debug profile.
func NewDebugFileStore(dir string) *Store {
	store := NewStore(dir, waitOnTimer)
	store.backend = absentDebugKeychain{}
	store.UsePlaintextFile()
	return store
}

type absentDebugKeychain struct{}

func (absentDebugKeychain) Get(string) (string, error) { return "", keyring.ErrUnsupportedPlatform }
func (absentDebugKeychain) Set(string, string) error   { return keyring.ErrUnsupportedPlatform }
func (absentDebugKeychain) Delete(string) error        { return keyring.ErrUnsupportedPlatform }
