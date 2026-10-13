package transcription

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// puts is a proofreader answering with what it was told to say about a batch,
// and keeping the lines it was asked about.
type puts struct {
	says map[int]string

	mu    sync.Mutex
	asked []int
}

func (p *puts) GetName() string { return "a proofreader" }

func (p *puts) Proofread(_ context.Context, batches []proofread.Batch) (map[int]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := map[int]string{}
	for _, batch := range batches {
		for _, line := range batch.Lines {
			p.asked = append(p.asked, line.Number)
		}
		if said, held := p.says[batch.Number]; held {
			out[batch.Number] = said
		}
	}
	return out, nil
}

// lines is what this proofreader was asked about.
func (p *puts) lines() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.asked)
}

// A shelf is what a recognition is written to, in memory.
type shelf struct {
	mu     sync.Mutex
	files  map[string][]byte
	held   map[string]bool
	opens  int
	closes int
}

func newShelf() *shelf { return &shelf{files: map[string][]byte{}, held: map[string]bool{}} }

// shelves hands the one shelf to whatever opens a vault's store.
type shelves struct{ *shelf }

func (s shelves) Open(domain.Vault) (port.DerivedStore, error) { return s.shelf, nil }

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

// claims says whether a name is held by anybody.
func (s *shelf) claims(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[name]
}

func (s *shelf) names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for name := range s.files {
		out = append(out, name)
	}
	return out
}

// A corrector answers about a page with whatever it was told to say about it,
// and keeps the pages of every request it was given.
type corrector struct {
	name  string
	says  map[int]string
	asked [][]int
	// about is what each batch it was given said the text holds.
	about []string
	fail  error
	// stop is called with the number of requests made so far.
	stop func(int)
}

func (c *corrector) GetName() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) Proofread(ctx context.Context, pages []proofread.Batch) (map[int]string, error) {
	var at []int
	for _, page := range pages {
		at = append(at, page.Number)
		c.about = append(c.about, page.Context)
	}
	c.asked = append(c.asked, at)
	if c.stop != nil {
		c.stop(len(c.asked))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.fail != nil {
		return nil, c.fail
	}
	out := map[int]string{}
	for _, page := range pages {
		if said, has := c.says[page.Number]; has {
			out[page.Number] = said
		}
	}
	return out, nil
}

// corrects is a reply putting one line right.
func corrects(at int, text string) string { return fmt.Sprintf("%d|%s", at, text) }

func readShelf(t *testing.T, shelved *shelf, name string) []byte {
	t.Helper()
	raw, err := shelved.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// brokenReaders fails any attempt to open a vault reader.
type brokenReaders struct{}

func (brokenReaders) Open(domain.Vault) (port.VaultReader, error) {
	return nil, errors.New("broken reader: must not be opened")
}
