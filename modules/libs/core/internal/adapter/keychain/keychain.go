// Package keychain keeps keys in the system keychain: the Keychain on macOS, the
// Credential Manager on Windows and the Secret Service on Linux.
package keychain

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// service is the name every key of this application is kept under.
const service = "md.numen"

var errEmptyName = errors.New("a key needs a name")

// Store is the system keychain. A machine without one fails every call with
// port.ErrNoKeyStore, and no key is written anywhere else.
type Store struct{}

// GetKey returns the key stored under the name.
func (Store) GetKey(ctx context.Context, name string) (string, error) {
	if err := check(ctx, name); err != nil {
		return "", err
	}
	key, err := await(ctx, func() (string, error) { return keyring.Get(service, name) })
	if err != nil {
		return "", readError(name, err)
	}
	return key, nil
}

// SetKey stores the key under the name and replaces the one already there.
func (Store) SetKey(ctx context.Context, name, key string) error {
	if err := check(ctx, name); err != nil {
		return err
	}
	_, err := await(ctx, func() (struct{}, error) { return struct{}{}, keyring.Set(service, name, key) })
	return readError(name, err)
}

// DeleteKey removes the key stored under the name.
func (Store) DeleteKey(ctx context.Context, name string) error {
	if err := check(ctx, name); err != nil {
		return err
	}
	_, err := await(ctx, func() (struct{}, error) { return struct{}{}, keyring.Delete(service, name) })
	return readError(name, err)
}

// await returns when the call does or the context ends. A call the context ends
// on is left to finish by itself: the keychain offers no way to stop it, and a
// write may still land.
func await[T any](ctx context.Context, call func() (T, error)) (T, error) {
	type outcome struct {
		value T
		err   error
	}
	isDone := make(chan outcome, 1)
	go func() {
		value, err := call()
		isDone <- outcome{value, err}
	}()
	select {
	case result := <-isDone:
		return result.value, result.err
	case <-ctx.Done():
		var none T
		return none, ctx.Err()
	}
}

func check(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return errEmptyName
	}
	return nil
}

// readError maps a failure of the keychain to the core's own: a key that is not
// there is port.ErrNoKey, and a keychain that cannot be reached is
// port.ErrNoKeyStore with what the machine said.
func readError(name string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, keyring.ErrNotFound):
		return fmt.Errorf("%w: %q", pluginport.ErrNoKey, name)
	default:
		return fmt.Errorf("%w: %w", pluginport.ErrNoKeyStore, err)
	}
}
