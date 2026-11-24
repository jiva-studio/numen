// Package keyfile lets the person choose how a computer without a system keychain keeps its keys.
package keyfile

import (
	"context"
	"errors"
	"fmt"

	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
)

// ErrInvalidMode is a choice of a mode that cannot be chosen.
var ErrInvalidMode = errors.New("the mode cannot be chosen")

// ErrNotSaved is a choice that could not be kept for the next start.
var ErrNotSaved = errors.New("the choice could not be saved")

// File is the file a computer without a system keychain keeps its keys in.
type File interface {
	// GetState returns how the file is kept; IsInUse is false where the system keychain works.
	GetState(ctx context.Context) (domainkeyfile.State, error)
	// UsePlaintext keeps the keys of the file as plaintext from now on. It returns what keeps the file
	// from being written again, such as a passphrase file that is still locked.
	UsePlaintext(ctx context.Context) error
	// UsePassphrase encrypts the keys of the file with the passphrase from now on. It returns
	// domainkeyfile.ErrWrongPassphrase for a passphrase that does not open a file that is already encrypted.
	UsePassphrase(ctx context.Context, passphrase string) error
	// Unlock gives the passphrase of the encrypted file for this run. It returns domainkeyfile.ErrWrongPassphrase
	// for a passphrase that does not open the file.
	Unlock(ctx context.Context, passphrase string) error
}

// GetState reads how the key file is kept.
type GetState struct {
	file File
}

// NewGetState returns the use case over the key file.
func NewGetState(file File) *GetState { return &GetState{file: file} }

// Execute returns how the key file is kept now.
func (u *GetState) Execute(ctx context.Context) (domainkeyfile.State, error) {
	return u.file.GetState(ctx)
}

// Choose sets how the key file keeps the keys and remembers it for the next start.
type Choose struct {
	file     File
	remember func(mode domainkeyfile.Mode) error
}

// NewChoose returns the use case over the key file; remember keeps the mode for the next start.
func NewChoose(file File, remember func(mode domainkeyfile.Mode) error) *Choose {
	return &Choose{file: file, remember: remember}
}

// Execute applies the mode and then remembers it. It returns ErrInvalidMode for any mode but plaintext
// and passphrase, domainkeyfile.ErrEmptyPassphrase for a passphrase mode without one, and ErrNotSaved when
// the choice stays in force for this run only.
func (u *Choose) Execute(ctx context.Context, mode domainkeyfile.Mode, passphrase string) error {
	switch mode {
	case domainkeyfile.Plaintext:
		if err := u.file.UsePlaintext(ctx); err != nil {
			return err
		}
	case domainkeyfile.Passphrase:
		if passphrase == "" {
			return domainkeyfile.ErrEmptyPassphrase
		}
		if err := u.file.UsePassphrase(ctx, passphrase); err != nil {
			return err
		}
	case domainkeyfile.Unset:
		return ErrInvalidMode
	default:
		return ErrInvalidMode
	}
	if err := u.remember(mode); err != nil {
		return fmt.Errorf("%w: %w", ErrNotSaved, err)
	}
	return nil
}

// Unlock gives the passphrase of the encrypted key file for this run.
type Unlock struct {
	file File
}

// NewUnlock returns the use case over the key file.
func NewUnlock(file File) *Unlock { return &Unlock{file: file} }

// Execute opens the file with the passphrase. It returns domainkeyfile.ErrEmptyPassphrase for none and
// domainkeyfile.ErrWrongPassphrase for one that does not open the file.
func (u *Unlock) Execute(ctx context.Context, passphrase string) error {
	if passphrase == "" {
		return domainkeyfile.ErrEmptyPassphrase
	}
	return u.file.Unlock(ctx, passphrase)
}
