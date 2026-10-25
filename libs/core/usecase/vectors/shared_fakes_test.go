package vectors

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// fakeStorage is what a recognition is written to, in memory.
type fakeStorage struct {
	mu         sync.Mutex
	files      map[domain.ArtifactName][]byte
	activeItem map[domain.ArtifactName]bool
	opens      int
	closes     int
}

func newShelf() *fakeStorage {
	return &fakeStorage{files: map[domain.ArtifactName][]byte{}, activeItem: map[domain.ArtifactName]bool{}}
}

// Open is one file of the store, which a copy of a video is played from.
func (s *fakeStorage) OpenArtifact(_ context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, target := s.files[name]
	if !target {
		return nil, 0, fs.ErrNotExist
	}
	s.opens++
	return trackedCloser{Reader: bytes.NewReader(raw), of: s}, int64(len(raw)), nil
}

// WriteStream puts what a reader gives under a name.
func (s *fakeStorage) WriteArtifactStream(_ context.Context, name domain.ArtifactName, from io.Reader) (int64, error) {
	raw, err := io.ReadAll(from)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = raw
	return int64(len(raw)), nil
}

// trackedCloser is one file opened from a store, counted against it once it
// is closed.
type trackedCloser struct {
	*bytes.Reader
	of *fakeStorage
}

func (c trackedCloser) Close() error {
	c.of.mu.Lock()
	defer c.of.mu.Unlock()
	c.of.closes++
	return nil
}

func (s *fakeStorage) ReadArtifact(_ context.Context, name domain.ArtifactName) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, target := s.files[name]
	if !target {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s *fakeStorage) WriteArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append([]byte(nil), content...)
	return nil
}

// Remove takes a name out of the store, along with the claim on it: a claim is
// held on a file, and taking that file away frees the name.
func (s *fakeStorage) RemoveArtifact(_ context.Context, name domain.ArtifactName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, name)
	delete(s.activeItem, name)
	return nil
}

func (s *fakeStorage) ClaimArtifact(_ context.Context, name domain.ArtifactName) (func() error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeItem[name] {
		return nil, vaultport.ErrClaimed
	}
	s.activeItem[name] = true
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.activeItem, name)
		return nil
	}, nil
}

// hold takes a name the way another run holds it.
func (s *fakeStorage) hold(name domain.ArtifactName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeItem[name] = true
}

const bookPath = "library/book.epub"
