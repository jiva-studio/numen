package fakeserver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// bodyStore keeps the bodies of parts and of finished multipart objects, in memory or in files of a folder, so a
// process that stops and runs again over the folder still has them.
type bodyStore struct {
	dir  string
	held map[string][]byte
}

func newBodyStore(dir string) *bodyStore {
	return &bodyStore{dir: dir, held: map[string][]byte{}}
}

func (b *bodyStore) write(name string, body []byte) error {
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

func (b *bodyStore) read(name string) ([]byte, bool) {
	if b.dir == "" {
		body, found := b.held[name]
		return body, found
	}
	body, err := os.ReadFile(filepath.Join(b.dir, name))
	return body, err == nil
}

func (b *bodyStore) remove(name string) {
	if b.dir == "" {
		delete(b.held, name)
		return
	}
	if err := os.Remove(filepath.Join(b.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
}
