package source

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
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

type (
	shelf   = fakeStorage
	shelves = fakeStores
)

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

// closeCount is how many files opened from this store were closed, so a test
// can tell a leak from one that was not.
func (s *fakeStorage) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closes
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

// A speaker reads every page as the same thing, or as nothing, and counts how
// many pages it was shown.
type speaker struct {
	says string
	// heads is what every page says in the region that opens a part of the
	// document. Empty is a page opening none.
	heads string
	// blank is the page, counted from one, that the model reads nothing on.
	blank int
	pages int
	stop  func(int)
}

func (s *speaker) GetRecognitionModel() port.RecognitionModel {
	return port.RecognitionModel{Layout: "layout", Recogniser: "reader", DPI: 300, From: "a test"}
}

func (s *speaker) Recognise(ctx context.Context, _ image.Image) ([]domain.OCRBlock, error) {
	s.pages++
	if s.stop != nil {
		s.stop(s.pages)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.says == "" || s.pages == s.blank {
		return nil, nil
	}
	// No two pages say the same thing, so a coordinate read at the wrong offset
	// names the wrong words.
	response := fmt.Sprintf("%s %d", s.says, s.pages)
	var out []domain.OCRBlock
	if s.heads != "" {
		out = append(out, domain.OCRBlock{Label: "doc_title", Text: fmt.Sprintf("%s %d", s.heads, s.pages), IsHeading: true, Depth: 1})
	}
	return append(out, domain.OCRBlock{
		Label: "text",
		Text:  response,
		Boxes: []domain.OCRBox{{Rect: image.Rect(10, 20, 30, 40), Span: domain.ByteSpan{To: len(response)}}},
	}), nil
}

func (s *speaker) Close() error { return nil }

const documentPath = "library/scan.pdf"

// names are what one recognition of the fixture is kept under.
type names struct {
	hash        string
	artifact    domain.ArtifactName
	corrections domain.ArtifactName
	far         domain.ArtifactName
}

// textNames finds the one recognition on the shelf and says what its files are
// called.
func textNames(t *testing.T, shelved *fakeStorage) names {
	t.Helper()
	for _, name := range shelved.artifactNames() {
		if name.File != domain.ArtifactText || name.Producer != domain.OCR {
			continue
		}
		hash := name.Hash
		return names{
			hash:        hash,
			artifact:    domain.NewArtifactName(domain.ArtifactText, domain.OCR, hash),
			corrections: domain.NewArtifactName(domain.ArtifactCorrections, domain.OCR, hash),
			far:         domain.NewArtifactName(domain.ArtifactProofread, domain.OCR, hash),
		}
	}
	t.Fatal("nothing was read")
	return names{}
}

const recordingPath = "library/talk.mp3"

// newRecording is the bytes of the file the words were spoken into. Nothing
// here reads them: what the recording says comes from the model, and the model
// is a fake.
func newRecording(words []string) []byte {
	return []byte("a recording of " + strings.Join(words, "|"))
}

// getWords is the words of an artifact, in the order they were said.
func getWords(t *testing.T, raw []byte) []string {
	t.Helper()
	_, cues := transcript.Parse(raw)
	out := make([]string, 0, len(cues))
	for _, cue := range cues {
		out = append(out, cue.Text)
	}
	return out
}
