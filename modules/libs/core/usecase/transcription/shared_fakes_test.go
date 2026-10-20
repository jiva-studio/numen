package transcription

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A shelf is what a recognition is written to, in memory.
type shelf struct {
	mu     sync.Mutex
	files  map[domain.ArtifactName][]byte
	held   map[domain.ArtifactName]bool
	opens  int
	closes int
}

func newShelf() *shelf {
	return &shelf{files: map[domain.ArtifactName][]byte{}, held: map[domain.ArtifactName]bool{}}
}

// shelves hands the one shelf to whatever opens a vault's store.
type shelves struct{ *shelf }

func (s shelves) Open(domain.Vault) (port.DerivedStore, error) { return s.shelf, nil }

// Open is one file of the store, which a copy of a video is played from.
func (s *shelf) OpenArtifact(_ context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error) {
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
func (s *shelf) WriteArtifactStream(_ context.Context, name domain.ArtifactName, from io.Reader) (int64, error) {
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

func (s *shelf) ReadArtifact(_ context.Context, name domain.ArtifactName) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, held := s.files[name]
	if !held {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s *shelf) WriteArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append([]byte(nil), content...)
	return nil
}

func (s *shelf) AppendArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[name] = append(s.files[name], content...)
	return nil
}

func (s *shelf) ListReviewLogs(context.Context) ([]port.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []port.Entry
	for held, content := range s.files {
		if held.File == domain.ArtifactReviewLog {
			out = append(out, port.Entry{Name: held, Size: len(content)})
		}
	}
	slices.SortFunc(out, func(a, b port.Entry) int { return strings.Compare(a.Name.Hash, b.Name.Hash) })
	return out, nil
}

// Remove takes a name out of the store, along with the claim on it: a claim is
// held on a file, and taking that file away frees the name.
func (s *shelf) RemoveArtifact(_ context.Context, name domain.ArtifactName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, name)
	delete(s.held, name)
	return nil
}

func (s *shelf) ClaimArtifact(_ context.Context, name domain.ArtifactName) (func() error, error) {
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
func (s *shelf) hold(name domain.ArtifactName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[name] = true
}

// claims says whether a name is held by anybody.
func (s *shelf) claims(name domain.ArtifactName) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[name]
}

func (s *shelf) names() []domain.ArtifactName {
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

func (noScan) ProofreadScan(context.Context, []domain.ProofreadPage, float64) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, errors.New("a transcript has no pages")
}

// A corrector answers about the lines of a window with the corrections it was
// told to make, the way a proofreader reaching the end of each window does, and
// keeps what every call asked.
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

	// windows are the windows of every call, given the lines of every call, and
	// asked the numbers of the lines each call answered.
	windows []domain.SpeechWindow
	given   [][]domain.ProofreadLine
	asked   [][]int
}

func (c *corrector) GetName() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) ProofreadSpeech(
	ctx context.Context, lines []domain.ProofreadLine, window domain.SpeechWindow,
) (domain.SpeechProofreading, error) {
	c.windows = append(c.windows, window)
	c.given = append(c.given, lines)
	if c.stop != nil {
		c.stop(len(c.windows))
	}
	if err := ctx.Err(); err != nil {
		return domain.SpeechProofreading{}, err
	}
	if c.fail != nil {
		return domain.SpeechProofreading{}, c.fail
	}

	at := 0
	for at < len(lines) && lines[at].Number < window.From {
		at++
	}
	end := len(lines)
	if window.Limit > 0 {
		end = min(end, at+window.Limit)
	}
	out := domain.SpeechProofreading{Uncorrected: c.uncorrected}
	var answered []int
	for _, line := range lines[at:end] {
		answered = append(answered, line.Number)
		if one, has := c.says[line.Number]; has {
			out.Corrections = append(out.Corrections, one)
		}
	}
	c.asked = append(c.asked, answered)
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
func fixes(at int, text string) domain.ProofreadCorrection {
	return domain.ProofreadCorrection{Number: at, Last: at, Text: text}
}

// joins is a correction putting a run of lines together as one.
func joins(at, through int, text string) domain.ProofreadCorrection {
	return domain.ProofreadCorrection{Number: at, Last: through, Text: text}
}

// says is the corrections as a corrector is told them.
func says(corrections ...domain.ProofreadCorrection) map[int]domain.ProofreadCorrection {
	out := map[int]domain.ProofreadCorrection{}
	for _, one := range corrections {
		out[one.Number] = one
	}
	return out
}

func readShelf(t *testing.T, shelved *shelf, name domain.ArtifactName) []byte {
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
