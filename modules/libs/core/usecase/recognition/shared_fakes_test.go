package recognition

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// puts is a proofreader answering with the correction it was told to make on a
// page, and keeping the pages it was asked about.
type puts struct {
	says map[int]domain.ProofreadCorrection

	mu    sync.Mutex
	asked []int
}

func (p *puts) GetName() string { return "a proofreader" }

func (p *puts) ProofreadScan(
	_ context.Context, pages []domain.ProofreadPage,
) (domain.ScanProofreading, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out domain.ScanProofreading
	for _, page := range pages {
		p.asked = append(p.asked, page.Number)
		if response, target := p.says[page.Number]; target {
			out.Corrections = append(out.Corrections, response)
		}
	}
	return out, nil
}

func (p *puts) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, errSpeechNotProofread
}

// errSpeechNotProofread is what the fakes answer to a transcript, which a
// recognition never sends.
var errSpeechNotProofread = errors.New("a recognition has no speech to proofread")

// lines is the pages this proofreader was asked about.
func (p *puts) lines() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.asked)
}

// fakeStorage is what a recognition is written to, in memory.
type fakeStorage struct {
	mu     sync.Mutex
	files  map[domain.ArtifactName][]byte
	held   map[domain.ArtifactName]bool
	opens  int
	closes int
}

func newShelf() *fakeStorage {
	return &fakeStorage{files: map[domain.ArtifactName][]byte{}, held: map[domain.ArtifactName]bool{}}
}

// fakeStores hands the one store to whatever opens a vault's store.
type fakeStores struct{ *fakeStorage }

func (s fakeStores) Open(domain.Vault) (port.DerivedStore, error) { return s.fakeStorage, nil }

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

func (s *fakeStorage) AppendArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append(s.files[name], content...)
	return nil
}

// Remove takes a name out of the store, along with the claim on it: a claim is
// held on a file, and taking that file away frees the name.
func (s *fakeStorage) RemoveArtifact(_ context.Context, name domain.ArtifactName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, name)
	delete(s.held, name)
	return nil
}

func (s *fakeStorage) ClaimArtifact(_ context.Context, name domain.ArtifactName) (func() error, error) {
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
func (s *fakeStorage) hold(name domain.ArtifactName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[name] = true
}

func (s *fakeStorage) artifactNames() []domain.ArtifactName {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.ArtifactName
	for name := range s.files {
		out = append(out, name)
	}
	return out
}

// A corrector answers about a page with the correction it was told to make on
// it, and keeps the pages of every request it was given.
type corrector struct {
	name string
	says map[int]domain.ProofreadCorrection
	// refuses is the pages whose reply is no answer.
	refuses map[int]bool
	asked   [][]int
	fail    error
	// stop is called with the number of requests made so far.
	stop func(int)
}

func (c *corrector) GetName() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) ProofreadScan(
	ctx context.Context, pages []domain.ProofreadPage,
) (domain.ScanProofreading, error) {
	var at []int
	for _, page := range pages {
		at = append(at, page.Number)
	}
	c.asked = append(c.asked, at)
	if c.stop != nil {
		c.stop(len(c.asked))
	}
	if err := ctx.Err(); err != nil {
		return domain.ScanProofreading{}, err
	}
	if c.fail != nil {
		return domain.ScanProofreading{}, c.fail
	}
	var out domain.ScanProofreading
	for _, page := range pages {
		if c.refuses[page.Number] {
			out.Uncorrected++
			continue
		}
		if response, has := c.says[page.Number]; has {
			out.Corrections = append(out.Corrections, response)
		}
	}
	return out, nil
}

func (c *corrector) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, errSpeechNotProofread
}

// corrects is the correction putting one line right.
func corrects(at int, text string) domain.ProofreadCorrection {
	return domain.ProofreadCorrection{Number: at, Last: at, Text: text}
}

func readShelf(t *testing.T, shelved *fakeStorage, name domain.ArtifactName) []byte {
	t.Helper()
	raw, err := shelved.ReadArtifact(t.Context(), name)
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
