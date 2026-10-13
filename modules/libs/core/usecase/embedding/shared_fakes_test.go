package embedding

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A shelf is what a recognition is written to, in memory.
type shelf struct {
	mu     sync.Mutex
	files  map[string][]byte
	held   map[string]bool
	opens  int
	closes int
}

func newShelf() *shelf { return &shelf{files: map[string][]byte{}, held: map[string]bool{}} }

// Open is one file of the store, which a copy of a video is played from.
func (s *shelf) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, held := s.files[name]
	if !held {
		return nil, 0, fs.ErrNotExist
	}
	s.opens++
	return trackedCloser{Reader: bytes.NewReader(raw), of: s}, int64(len(raw)), nil
}

// WriteStream puts what a reader gives under a name.
func (s *shelf) WriteStream(_ context.Context, name string, from io.Reader) (int64, error) {
	raw, err := io.ReadAll(from)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = raw
	return int64(len(raw)), nil
}

// trackedCloser is one file opened from a shelf, counted against it once it
// is closed.
type trackedCloser struct {
	*bytes.Reader
	of *shelf
}

func (c trackedCloser) Close() error {
	c.of.mu.Lock()
	defer c.of.mu.Unlock()
	c.of.closes++
	return nil
}

func (s *shelf) Read(_ context.Context, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, held := s.files[name]
	if !held {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s *shelf) Write(_ context.Context, name string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append([]byte(nil), content...)
	return nil
}

func (s *shelf) Append(_ context.Context, name string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append(s.files[name], content...)
	return nil
}

func (s *shelf) List(_ context.Context, name string) ([]port.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []port.Entry
	for held, content := range s.files {
		if strings.HasPrefix(held, name+"/") && !strings.Contains(held[len(name)+1:], "/") {
			out = append(out, port.Entry{Name: held, Size: len(content)})
		}
	}
	slices.SortFunc(out, func(a, b port.Entry) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Remove takes a name out of the store, along with the claim on it: a claim is
// held on a file, and taking that file away frees the name.
func (s *shelf) Remove(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, name)
	delete(s.held, name)
	return nil
}

func (s *shelf) ClaimName(_ context.Context, name string) (func() error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held[name] {
		return nil, port.ErrClaimed
	}
	s.held[name] = true
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.held, name)
		return nil
	}, nil
}

// hold takes a name the way another run holds it.
func (s *shelf) hold(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[name] = true
}

// bookPath is where the book sits in the vault under test.
const bookPath = "library/book.epub"
