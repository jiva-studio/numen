package transcription

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// puts is a proofreader answering with what it was told to say about a batch,
// and keeping the lines it was asked about.
type puts struct {
	says map[int]string

	mu    sync.Mutex
	asked []int
}

func (p *puts) GetName() string { return "a proofreader" }

func (p *puts) Proofread(_ context.Context, request domain.ProofreadRequest) (domain.ProofreadAnswer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	answer, asked := answerSpeech(p.says, request)
	p.asked = append(p.asked, asked...)
	return answer, nil
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

// A corrector answers about a page with whatever it was told to say about it,
// and keeps the pages of every request it was given.
type corrector struct {
	name  string
	says  map[int]string
	asked [][]int
	// requests are the requests it was given.
	requests []domain.ProofreadRequest
	fail     error
	// stop is called with the number of requests made so far.
	stop func(int)
}

func (c *corrector) GetName() string {
	if c.name == "" {
		return "a proofreader"
	}
	return c.name
}

func (c *corrector) Proofread(ctx context.Context, request domain.ProofreadRequest) (domain.ProofreadAnswer, error) {
	c.requests = append(c.requests, request)
	answer, asked := answerSpeech(c.says, request)
	c.asked = append(c.asked, asked)
	if c.stop != nil {
		c.stop(len(c.asked))
	}
	if err := ctx.Err(); err != nil {
		return domain.ProofreadAnswer{}, err
	}
	if c.fail != nil {
		return domain.ProofreadAnswer{}, c.fail
	}
	return answer, nil
}

// refused is the reply a proofreader gives that is no correction.
const refused = "refused"

// answerSpeech answers a request the way a proofreader of speech does: it takes
// the lines from the first one asked for, at most the limit of them, and puts
// right those that were told what to say. A saying stands for the line it is
// keyed by. It is a row of the form "12|words" or "12-14|words", or refused.
// The lines asked about come back with it.
func answerSpeech(says map[int]string, request domain.ProofreadRequest) (domain.ProofreadAnswer, []int) {
	var lines []domain.ProofreadLine
	for _, group := range request.Groups {
		lines = append(lines, group.Lines...)
	}
	at := 0
	for at < len(lines) && lines[at].Number < request.FromLine {
		at++
	}
	if at >= len(lines) {
		return domain.ProofreadAnswer{NextLine: request.FromLine, IsDone: true}, nil
	}
	end := len(lines)
	if request.LineLimit > 0 {
		end = min(end, at+request.LineLimit)
	}

	var answer domain.ProofreadAnswer
	var asked []int
	for _, line := range lines[at:end] {
		asked = append(asked, line.Number)
		said, held := says[line.Number]
		if !held {
			continue
		}
		if said == refused {
			answer.Uncorrected++
			continue
		}
		for _, row := range strings.Split(said, "\n") {
			head, words, _ := strings.Cut(row, "|")
			first, last, joined := strings.Cut(head, "-")
			number, _ := strconv.Atoi(first)
			through := number
			if joined {
				through, _ = strconv.Atoi(last)
			}
			answer.Corrections = append(answer.Corrections, domain.ProofreadCorrection{Number: number, Last: through, Text: words})
		}
	}
	if end == len(lines) {
		answer.NextLine, answer.IsDone = lines[end-1].Number+1, true
		return answer, asked
	}
	shared := min(max(request.Overlap, 0), max(request.Size-1, 0))
	answer.NextLine = lines[max(end-shared, at+1)].Number
	return answer, asked
}

// corrects is a reply putting one line right.
func corrects(at int, text string) string { return fmt.Sprintf("%d|%s", at, text) }

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
