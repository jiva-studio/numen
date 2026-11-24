// Package bodystore keeps the bodies of blobs, parts and finished multipart objects.
package bodystore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// Store keeps bodies in files of a folder, so a process that stops and runs again over the folder still has them and
// holds no body in memory. With no folder it keeps them in memory, for tests.
type Store struct {
	dir  string
	held map[string][]byte
}

// New returns the store over the folder; an empty folder keeps the bodies in memory.
func New(dir string) *Store {
	return &Store{dir: dir, held: map[string][]byte{}}
}

// Staged is a body that was received but is not yet kept under a name.
type Staged struct {
	// Size is the count of bytes received.
	Size  int64
	store *Store
	file  string
	body  []byte
}

// Stage reads the body from the reader into a temporary file, at most limit bytes, without holding it in memory.
func (b *Store) Stage(from io.Reader, limit int64) (*Staged, error) {
	if b.dir == "" {
		body, err := io.ReadAll(io.LimitReader(from, limit))
		return &Staged{Size: int64(len(body)), store: b, body: body}, err
	}
	if err := os.MkdirAll(b.dir, 0o750); err != nil {
		return nil, fmt.Errorf("make the body folder: %w", err)
	}
	file, err := os.CreateTemp(b.dir, "upload-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("stage a body: %w", err)
	}
	size, copyErr := io.Copy(file, io.LimitReader(from, limit))
	closeErr := file.Close()
	staged := &Staged{Size: size, store: b, file: file.Name()}
	if err := errors.Join(copyErr, closeErr); err != nil {
		staged.Discard()
		return nil, err
	}
	return staged, nil
}

// Commit keeps the staged body under the name.
func (s *Staged) Commit(name string) error {
	if s.store.dir == "" {
		s.store.held[name] = s.body
		return nil
	}
	target := filepath.Join(s.store.dir, name)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("make the folder of a body: %w", err)
	}
	return os.Rename(s.file, target)
}

// Discard drops the staged body.
func (s *Staged) Discard() {
	if s.file != "" {
		_ = os.Remove(s.file)
	}
}

// Write keeps the body under the name.
func (b *Store) Write(name string, body []byte) error {
	staged, err := b.Stage(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	return staged.Commit(name)
}

// Join keeps the bodies of the parts, one after another, as the body of the name, and drops the parts.
func (b *Store) Join(name string, parts []string) error {
	if b.dir == "" {
		var all []byte
		for _, part := range parts {
			body, found := b.held[part]
			if !found {
				return fmt.Errorf("part %s is not stored", part)
			}
			all = append(all, body...)
		}
		b.held[name] = all
		for _, part := range parts {
			delete(b.held, part)
		}
		return nil
	}
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		for _, part := range parts {
			file, err := os.Open(filepath.Join(b.dir, part))
			if err == nil {
				_, err = io.Copy(pipeWriter, file)
				_ = file.Close()
			}
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
		}
		_ = pipeWriter.Close()
	}()
	staged, err := b.Stage(pipeReader, 1<<62)
	if err != nil {
		return err
	}
	if err := staged.Commit(name); err != nil {
		return err
	}
	for _, part := range parts {
		b.Remove(part)
	}
	return nil
}

// Has tells whether a body is kept under the name.
func (b *Store) Has(name string) bool {
	if b.dir == "" {
		_, found := b.held[name]
		return found
	}
	_, err := os.Stat(filepath.Join(b.dir, name))
	return err == nil
}

// Open returns a reader of the body kept under the name and its size, read from the file on demand.
func (b *Store) Open(name string) (io.ReadSeekCloser, int64, bool) {
	if b.dir == "" {
		body, found := b.held[name]
		return nopCloser{bytes.NewReader(slices.Clone(body))}, int64(len(body)), found
	}
	file, err := os.Open(filepath.Join(b.dir, name))
	if err != nil {
		return nil, 0, false
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, false
	}
	return file, info.Size(), true
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

// Read returns the body kept under the name, whole; it is for the test endpoint that lists every body.
func (b *Store) Read(name string) ([]byte, bool) {
	reader, _, found := b.Open(name)
	if !found {
		return nil, false
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
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
