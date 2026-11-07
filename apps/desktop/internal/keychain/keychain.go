// Package keychain keeps keys in the system keychain: the Keychain on macOS, the
// Credential Manager on Windows and the Secret Service on Linux.
package keychain

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/zalando/go-keyring"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// service is the name every key of this application is kept under.
const service = "md.numen"

var (
	errEmptyName = errors.New("a key needs a name")
	errTooLong   = errors.New("the value is too long")
)

// maxValueBytes is the longest value a store keeps, in bytes: the smallest limit of the system
// keychains (2560 on Windows). The file keeps the same limit, so a file moved between machines
// stays valid.
const maxValueBytes = 2560

// Store keeps keys in the system keychain and, where the machine has none, in a file next to the
// settings file. A store made by NewStore probes the keychain once, at the first call. The zero
// Store is keychain only: a machine without one fails every call with port.ErrNoKeyStore and no
// key is written anywhere else.
type Store struct {
	// dir is the folder of the settings file. It is empty for a keychain only store.
	dir string

	mu       sync.Mutex
	isProbed bool
	// isKeychainUp is the result of the probe.
	isKeychainUp bool
}

// NewStore returns a store that keeps keys in the system keychain, and in
// <dir>/credentials/credentials.json where the keychain cannot be reached. dir is the folder of
// the settings file.
func NewStore(dir string) *Store { return &Store{dir: dir} }

// GetKey returns the key stored under the name: the one in the keychain, else the one in the file.
func (s *Store) GetKey(ctx context.Context, name string) (string, error) {
	if err := check(ctx, name); err != nil {
		return "", err
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return "", err
	}
	if isUp {
		key, err := getFromKeychain(ctx, name)
		if err == nil || !errors.Is(err, pluginport.ErrNoKey) || s.dir == "" {
			return key, err
		}
	}
	return s.readFromFile(name)
}

// SetKey stores the key under the name and replaces the one already there. A key over the limit of
// the smallest keychain is refused and stored nowhere.
func (s *Store) SetKey(ctx context.Context, name, key string) error {
	if err := check(ctx, name); err != nil {
		return err
	}
	if len(key) > maxValueBytes {
		return fmt.Errorf("%w: a value is over %d bytes, the smallest limit of the system keychains", errTooLong, maxValueBytes)
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return err
	}
	if isUp {
		_, err := await(ctx, func() (struct{}, error) { return struct{}{}, keyring.Set(service, name, key) })
		return readError(name, err)
	}
	return s.writeToFile(name, key)
}

// DeleteKey removes the key stored under the name, from the keychain and from the file.
func (s *Store) DeleteKey(ctx context.Context, name string) error {
	if err := check(ctx, name); err != nil {
		return err
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return err
	}
	var failures []error
	hasRemoved := false
	if isUp {
		_, err := await(ctx, func() (struct{}, error) { return struct{}{}, keyring.Delete(service, name) })
		if err = readError(name, err); err == nil {
			hasRemoved = true
		} else if !errors.Is(err, pluginport.ErrNoKey) {
			failures = append(failures, err)
		}
	}
	if s.dir != "" {
		switch err := s.removeFromFile(name); {
		case err == nil:
			hasRemoved = true
		case !errors.Is(err, pluginport.ErrNoKey):
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	if !hasRemoved {
		return fmt.Errorf("%w: %q", pluginport.ErrNoKey, name)
	}
	return nil
}

// GetCredentialsPath returns the file the keys are kept in while the keychain cannot be reached,
// and the empty text while the keychain is in use.
func (s *Store) GetCredentialsPath(ctx context.Context) string {
	if s.dir == "" {
		return ""
	}
	if isUp, err := s.isKeychainReachable(ctx); err != nil || isUp {
		return ""
	}
	return s.getCredentialsFile()
}

// isKeychainReachable probes the keychain at the first call and answers from the probe after that.
// A keychain only store has no file to fall back to, so it is never probed and its errors reach the
// caller.
func (s *Store) isKeychainReachable(ctx context.Context) (bool, error) {
	if s.dir == "" {
		return true, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isProbed {
		return s.isKeychainUp, nil
	}
	_, err := await(ctx, func() (string, error) { return keyring.Get(service, probeName) })
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false, err
	}
	s.isProbed = true
	s.isKeychainUp = err == nil || errors.Is(err, keyring.ErrNotFound)
	return s.isKeychainUp, nil
}

// probeName is the entry the probe asks the keychain for. No key is kept under it.
const probeName = "probe"

// getFromKeychain reads the key from the keychain.
func getFromKeychain(ctx context.Context, name string) (string, error) {
	key, err := await(ctx, func() (string, error) { return keyring.Get(service, name) })
	if err != nil {
		return "", readError(name, err)
	}
	return key, nil
}

// await returns when the call does or the context ends. A call the context ends
// on is left to finish by itself: the keychain offers no way to stop it, and a
// write may still land.
func await[T any](ctx context.Context, call func() (T, error)) (T, error) {
	type outcome struct {
		value T
		err   error
	}
	// gomutants:disable-next-line
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
