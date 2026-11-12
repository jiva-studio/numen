// Package keychain keeps keys in the system keychain: the Keychain on macOS, the
// Credential Manager on Windows and the Secret Service on Linux.
package keychain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
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

// Store keeps keys in the system keychain and, where the machine has none, in a file next to the settings file.
// A store made by NewStore probes the keychain until it is reached. While it is reachable the file is not read, and a key stored in the keychain is removed from the file.
// A keychain that is there and refuses fails the call with port.ErrNoKeyStore, and nothing is written to the file.
// The zero Store is keychain only: a machine without one fails every call with port.ErrNoKeyStore.
type Store struct {
	dir     string
	writer  chan struct{}
	wait    machine.Wait
	backend keychainBackend

	mu           sync.Mutex
	isKeychainUp bool
	probe        *probeCall
}

// keychainBackend is the keychain a store calls. The system keychain is the default.
type keychainBackend interface {
	Get(name string) (string, error)
	Set(name, key string) error
	Delete(name string) error
}

type systemKeychain struct{}

func (systemKeychain) Get(name string) (string, error) { return keyring.Get(service, name) }
func (systemKeychain) Set(name, key string) error      { return keyring.Set(service, name, key) }
func (systemKeychain) Delete(name string) error        { return keyring.Delete(service, name) }

func (s *Store) getBackend() keychainBackend {
	if s.backend == nil {
		return systemKeychain{}
	}
	return s.backend
}

// probeCall is one probe of the keychain that every caller waiting for the answer shares.
type probeCall struct {
	finished chan struct{}
	isUp     bool
	err      error
}

// NewStore returns a store that keeps keys in the system keychain, and in
// <dir>/credentials/credentials.json where the keychain cannot be reached. dir is the folder of
// the settings file.
func NewStore(dir string, wait machine.Wait) *Store {
	return &Store{dir: dir, wait: wait, writer: make(chan struct{}, 1)}
}

// GetKey returns the key stored under the name: the one in the keychain, or, on a machine that has
// no keychain, the one in the file.
func (s *Store) GetKey(ctx context.Context, name string) (string, error) {
	if err := checkName(ctx, name); err != nil {
		return "", err
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return "", err
	}
	if isUp {
		return s.getFromKeychain(ctx, name)
	}
	return s.readFromFile(name)
}

// SetKey stores the key under the name and replaces the one already there. A key over the limit of
// the smallest keychain is refused and stored nowhere.
func (s *Store) SetKey(ctx context.Context, name, key string) error {
	if err := checkName(ctx, name); err != nil {
		return err
	}
	if len(key) > maxValueBytes {
		return fmt.Errorf("%w: a value is over %d bytes, the smallest limit of the system keychains", errTooLong, maxValueBytes)
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return err
	}
	if !isUp {
		return s.writeToFile(ctx, name, key)
	}
	_, err = await(ctx, func() (struct{}, error) { return struct{}{}, s.getBackend().Set(name, key) })
	if err = mapError(name, err); err != nil {
		return err
	}
	return s.removeStaleCopy(ctx, name)
}

func (s *Store) removeStaleCopy(ctx context.Context, name string) error {
	if s.dir == "" {
		return nil
	}
	if _, err := os.Stat(s.getCredentialsFile()); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := s.removeFromFile(ctx, name); err != nil && !errors.Is(err, plugin.ErrNoKey) {
		return err
	}
	return nil
}

// DeleteKey removes the key stored under the name, from the keychain and from the file.
func (s *Store) DeleteKey(ctx context.Context, name string) error {
	if err := checkName(ctx, name); err != nil {
		return err
	}
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil {
		return err
	}
	var failures []error
	hasRemoved := false
	if isUp {
		_, err := await(ctx, func() (struct{}, error) { return struct{}{}, s.getBackend().Delete(name) })
		if err = mapError(name, err); err == nil {
			hasRemoved = true
		} else if !errors.Is(err, plugin.ErrNoKey) {
			failures = append(failures, err)
		}
	}
	if s.dir != "" {
		switch err := s.removeFromFile(ctx, name); {
		case err == nil:
			hasRemoved = true
		case !errors.Is(err, plugin.ErrNoKey):
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	if !hasRemoved {
		return fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
	}
	return nil
}

func (s *Store) isKeychainReachable(ctx context.Context) (bool, error) {
	if s.dir == "" {
		return true, nil
	}
	s.mu.Lock()
	if s.isKeychainUp {
		s.mu.Unlock()
		return true, nil
	}
	call := s.probe
	if call == nil {
		call = &probeCall{finished: make(chan struct{})}
		s.probe = call
		go s.runProbe(call)
	}
	s.mu.Unlock()
	select {
	case <-call.finished:
		return call.isUp, call.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (s *Store) runProbe(call *probeCall) {
	_, err := s.getBackend().Get(probeName)
	switch {
	case err == nil, errors.Is(err, keyring.ErrNotFound):
		call.isUp = true
	case isKeychainAbsent(err):
	default:
		call.err = mapError(probeName, err)
	}
	s.mu.Lock()
	s.isKeychainUp = call.isUp
	s.probe = nil
	s.mu.Unlock()
	close(call.finished)
}

// probeName is the entry the probe asks the keychain for. No key is kept under it.
const probeName = "probe"

func (s *Store) getFromKeychain(ctx context.Context, name string) (string, error) {
	key, err := await(ctx, func() (string, error) { return s.getBackend().Get(name) })
	if err != nil {
		return "", mapError(name, err)
	}
	return key, nil
}

func await[T any](ctx context.Context, call func() (T, error)) (T, error) {
	type outcome struct {
		value T
		err   error
	}
	results := make(chan outcome, 1)
	go func() {
		value, err := call()
		results <- outcome{value, err}
	}()
	select {
	case result := <-results:
		return result.value, result.err
	case <-ctx.Done():
		var none T
		return none, ctx.Err()
	}
}

func checkName(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return errEmptyName
	}
	return nil
}

func mapError(name string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, keyring.ErrNotFound):
		return fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
	default:
		return fmt.Errorf("%w: %w", plugin.ErrNoKeyStore, err)
	}
}
