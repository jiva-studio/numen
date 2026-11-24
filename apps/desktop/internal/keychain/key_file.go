package keychain

import (
	"context"
	"errors"
	"sync"

	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
)

// ErrPassphraseNeeded is the credentials file that is encrypted while this run has not been given its
// passphrase. The window asks for it and the call is made again.
var ErrPassphraseNeeded = errors.New("the credentials file is locked: its passphrase is needed")

// Passphrase keeps the passphrase of the credentials file in memory for this run. It never writes it
// down. It is the PassphraseSource of the store: the window gives it through Set.
type Passphrase struct {
	mu    sync.Mutex
	value string
}

// GetPassphrase returns the passphrase the window gave, or ErrPassphraseNeeded.
func (p *Passphrase) GetPassphrase(context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.value == "" {
		return "", ErrPassphraseNeeded
	}
	return p.value, nil
}

// Set keeps the passphrase for this run.
func (p *Passphrase) Set(value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.value = value
}

// KeyFile is the credentials file of a store as the window sees it: the choice of its mode and the
// passphrase that opens it.
type KeyFile struct {
	store      *Store
	passphrase *Passphrase
}

// NewKeyFile returns the key file of the store. The store takes its passphrase from the passphrase it is given here.
func NewKeyFile(store *Store, passphrase *Passphrase) *KeyFile {
	return &KeyFile{store: store, passphrase: passphrase}
}

// GetState returns how the store keeps keys where the keychain cannot be reached.
func (k *KeyFile) GetState(ctx context.Context) (domainkeyfile.State, error) {
	state, err := k.store.GetFileState(ctx)
	if err != nil {
		return domainkeyfile.State{}, err
	}
	return domainkeyfile.State{IsInUse: state.IsInUse, Mode: readMode(state.Mode), Warning: state.Warning, IsLocked: state.IsLocked}, nil
}

func readMode(mode FileMode) domainkeyfile.Mode {
	switch mode {
	case FileModePlaintext:
		return domainkeyfile.Plaintext
	case FileModePassphrase:
		return domainkeyfile.Passphrase
	case FileModeUnset:
		return domainkeyfile.Unset
	}
	return domainkeyfile.Unset
}

// UsePlaintext keeps the keys as plaintext from now on. Keys the file holds are written again as plaintext.
func (k *KeyFile) UsePlaintext(ctx context.Context) error {
	use := func() {
		k.store.UsePlaintextFile()
		k.passphrase.Set("")
	}
	if k.store.getFileMode() == FileModeUnset {
		use()
		return nil
	}
	return k.store.ChangeFileMode(ctx, use)
}

// UsePassphrase encrypts the keys with the passphrase from now on. Keys the file holds are written again
// under it; a file that is encrypted already must be unlocked first.
func (k *KeyFile) UsePassphrase(ctx context.Context, passphrase string) error {
	use := func() {
		k.passphrase.Set(passphrase)
		k.store.UsePassphraseFile(k.passphrase)
	}
	if k.store.getFileMode() != FileModeUnset {
		return k.store.ChangeFileMode(ctx, use)
	}
	use()
	return k.check(ctx)
}

// Unlock gives the passphrase of the encrypted file for this run.
func (k *KeyFile) Unlock(ctx context.Context, passphrase string) error {
	k.passphrase.Set(passphrase)
	return k.check(ctx)
}

func (k *KeyFile) check(ctx context.Context) error {
	err := k.store.CheckFile(ctx)
	if errors.Is(err, ErrWrongPassphrase) {
		k.passphrase.Set("")
		return domainkeyfile.ErrWrongPassphrase
	}
	return err
}
