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
	Held map[string][]byte
}

// New returns the store over the folder; an empty folder keeps the bodies in memory.
func New(dir string) *Store {
	return &Store{dir: dir, Held: map[string][]byte{}}
}

// Write keeps the body under the name.
func (b *Store) Write(name string, body []byte) error {
	if b.dir == "" {
		b.Held[name] = slices.Clone(body)
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
		body, found := b.Held[name]
		return body, found
	}
	body, err := os.ReadFile(filepath.Join(b.dir, name))
	return body, err == nil
}

// Remove drops the body kept under the name.
func (b *Store) Remove(name string) {
	if b.dir == "" {
		delete(b.Held, name)
		return
	}
	if err := os.Remove(filepath.Join(b.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
}
