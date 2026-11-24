// Package keyfile lets the person choose how a computer without a system keychain keeps its keys.
package keyfile

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// ErrInvalidMode is a choice of a mode that cannot be chosen.
var ErrInvalidMode = errors.New("the mode cannot be chosen")

// ErrNotSaved is a choice that could not be kept for the next start.
var ErrNotSaved = errors.New("the choice could not be saved")

// GetState reads how the key file is kept.
type GetState struct {
	file pluginport.KeyFile
}

// NewGetState returns the use case over the key file.
func NewGetState(file pluginport.KeyFile) *GetState { return &GetState{file: file} }

// Execute returns how the key file is kept now.
func (u *GetState) Execute(ctx context.Context) (pluginport.KeyFileState, error) {
	return u.file.GetState(ctx)
}

// Choose sets how the key file keeps the keys and remembers it for the next start.
type Choose struct {
	file     pluginport.KeyFile
	remember func(mode pluginport.KeyFileMode) error
}

// NewChoose returns the use case over the key file; remember keeps the mode for the next start.
func NewChoose(file pluginport.KeyFile, remember func(mode pluginport.KeyFileMode) error) *Choose {
	return &Choose{file: file, remember: remember}
}

// Execute applies the mode and then remembers it. It returns ErrInvalidMode for any mode but plaintext
// and passphrase, pluginport.ErrEmptyPassphrase for a passphrase mode without one, and ErrNotSaved when
// the choice stays in force for this run only.
func (u *Choose) Execute(ctx context.Context, mode pluginport.KeyFileMode, passphrase string) error {
	switch mode {
	case pluginport.KeyFilePlaintext:
		u.file.UsePlaintext()
	case pluginport.KeyFilePassphrase:
		if passphrase == "" {
			return pluginport.ErrEmptyPassphrase
		}
		if err := u.file.UsePassphrase(ctx, passphrase); err != nil {
			return err
		}
	case pluginport.KeyFileUnset:
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
	file pluginport.KeyFile
}

// NewUnlock returns the use case over the key file.
func NewUnlock(file pluginport.KeyFile) *Unlock { return &Unlock{file: file} }

// Execute opens the file with the passphrase. It returns pluginport.ErrEmptyPassphrase for none and
// pluginport.ErrWrongPassphrase for one that does not open the file.
func (u *Unlock) Execute(ctx context.Context, passphrase string) error {
	if passphrase == "" {
		return pluginport.ErrEmptyPassphrase
	}
	return u.file.Unlock(ctx, passphrase)
}
