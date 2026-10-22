package transcription

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

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

// claims says whether a name is held by anybody.
func (s *fakeStorage) claims(name domain.ArtifactName) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[name]
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

// noScan is the half of a proofreader that puts printed pages right, which a
// transcript never asks.
type noScan struct{}

func (noScan) ProofreadScan(context.Context, []domain.ProofreadPage) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, errors.New("a transcript has no pages")
}

// A corrector answers about the lines with the corrections it was told to make
// and keeps what every call asked.
type corrector struct {
	noScan

	name string
	// says are the corrections by the number of the line they begin at.
	says map[int]domain.ProofreadCorrection
	// uncorrected is what each call says about the requests it could not use.
	uncorrected int
	fail        error
	// stop is called with the number of calls made so far.
	stop func(int)

	// limit is how many lines to answer at once. Zero answers all lines from from.
	limit int

	// froms are the starting line numbers of every call, given the lines of every call, and
	// asked the numbers of the lines each call answered.
	froms []int
	given [][]domain.ProofreadLine
	asked [][]int
}

func (c *corrector) GetName() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) ProofreadSpeech(
	ctx context.Context, lines []domain.ProofreadLine, from int,
) (domain.SpeechProofreading, error) {
	c.froms = append(c.froms, from)
	c.given = append(c.given, lines)
	if c.stop != nil {
		c.stop(len(c.froms))
	}
	if err := ctx.Err(); err != nil {
		return domain.SpeechProofreading{}, err
	}
	if c.fail != nil {
		return domain.SpeechProofreading{}, c.fail
	}

	pos := 0
	for pos < len(lines) && lines[pos].Number < from {
		pos++
	}
	end := len(lines)
	if c.limit > 0 {
		end = min(end, pos+c.limit)
	}
	out := domain.SpeechProofreading{Uncorrected: c.uncorrected}
	var reply []int
	for _, line := range lines[pos:end] {
		reply = append(reply, line.Number)
		if one, has := c.says[line.Number]; has {
			out.Corrections = append(out.Corrections, one)
		}
	}
	c.asked = append(c.asked, reply)
	if end == len(lines) {
		out.IsDone = true
		if end > 0 {
			out.Next = lines[end-1].Number + 1
		}
		return out, nil
	}
	out.Next = lines[end].Number
	return out, nil
}

// lines are the numbers of the lines every call answered, in order.
func (c *corrector) lines() []int {
	var out []int
	for _, one := range c.asked {
		out = append(out, one...)
	}
	return out
}

// fixes is a correction putting one line right.
func fixes(pos int, text string) domain.ProofreadCorrection {
	return domain.ProofreadCorrection{Number: pos, Last: pos, Text: text}
}

// joins is a correction putting a run of lines together as one.
func joins(pos, through int, text string) domain.ProofreadCorrection {
	return domain.ProofreadCorrection{Number: pos, Last: through, Text: text}
}

// says is the corrections as a corrector is told them.
func says(corrections ...domain.ProofreadCorrection) map[int]domain.ProofreadCorrection {
	out := map[int]domain.ProofreadCorrection{}
	for _, one := range corrections {
		out[one.Number] = one
	}
	return out
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
