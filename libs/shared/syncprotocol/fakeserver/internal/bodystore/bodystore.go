// Package bodystore keeps the bodies of parts and of finished multipart objects.
package bodystore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Store keeps the bodies of parts and of finished multipart objects, in memory or in files of a folder, so a
// process that stops and runs again over the folder still has them. Held is the bodies kept in memory.
type Store struct {
	dir  string
	held map[string][]byte
}

// New returns the store over the folder; an empty folder keeps the bodies in memory.
func New(dir string) *Store {
	return &Store{dir: dir, held: map[string][]byte{}}
}

// Write keeps the body under the name.
func (b *Store) Write(name string, body []byte) error {
	if b.dir == "" {
		b.held[name] = slices.Clone(body)
		return nil
	}
	target := filepath.Join(b.dir, name)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("make the folder of a body: %w", err)
	}
	temporary := target + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return fmt.Errorf("write a body: %w", err)
	}
	return os.Rename(temporary, target)
}

// Read returns the body kept under the name.
func (b *Store) Read(name string) ([]byte, bool) {
	if b.dir == "" {
		body, found := b.held[name]
		return body, found
	}
	body, err := os.ReadFile(filepath.Join(b.dir, name))
	return body, err == nil
}

// Remove drops the body kept under the name.
func (b *Store) Remove(name string) {
	if b.dir == "" {
		delete(b.held, name)
		return
	}
	if err := os.Remove(filepath.Join(b.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
}

// GetHeld returns the bodies kept in memory; it is empty when the store has a folder.
func (b *Store) GetHeld() map[string][]byte { return b.held }

// SetHeld replaces the bodies kept in memory.
func (b *Store) SetHeld(held map[string][]byte) { b.held = held }
