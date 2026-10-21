package text_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// The talk as a person left it, with the name spelled the way the speaker
// spelled it.
const putRight = "The name and the named are not two, said Rupa."

// shelf is a store holding what a run left, by the name it left it under.
type shelf map[domain.ArtifactName][]byte

func (b shelf) ReadArtifact(_ context.Context, name domain.ArtifactName) ([]byte, error) {
	raw, target := b[name]
	if !target {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

// Open is one file of the store, which a copy of a video is played from.
func (b shelf) OpenArtifact(_ context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error) {
	raw, target := b[name]
	if !target {
		return nil, 0, fs.ErrNotExist
	}
	return readingBytes{bytes.NewReader(raw)}, int64(len(raw)), nil
}

// WriteStream puts what a reader gives under a name.
func (b shelf) WriteArtifactStream(_ context.Context, name domain.ArtifactName, from io.Reader) (int64, error) {
	raw, err := io.ReadAll(from)
	if err != nil {
		return 0, err
	}
	b[name] = raw
	return int64(len(raw)), nil
}

// readingBytes is a reader of bytes already in memory, closed by nobody.
type readingBytes struct{ *bytes.Reader }

func (readingBytes) Close() error { return nil }

func (b shelf) WriteArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	b[name] = content
	return nil
}

func (b shelf) AppendArtifact(_ context.Context, name domain.ArtifactName, content []byte) error {
	b[name] = append(b[name], content...)
	return nil
}

func (b shelf) RemoveArtifact(_ context.Context, name domain.ArtifactName) error {
	delete(b, name)
	return nil
}

func (b shelf) ClaimArtifact(context.Context, domain.ArtifactName) (func() error, error) {
	return func() error { return nil }, nil
}

// writeCorrections is the transcript of the talk as somebody put it right.
func writeCorrections() []byte {
	return transcript.Marshal([]domain.TranscriptCue{
		{Text: opening, From: 1500, To: 4200},
		{Text: putRight, From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
}

// The words a transcript was put right to are the words it reads as, and the
// moments they were said at stand where they were.
func TestATranscriptPutRightReadsAsTheWordsItWasPutRightTo(t *testing.T) {
	store := shelf{
		domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"):        writeTranscript(),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"): writeCorrections(),
	}

	doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
	if err != nil {
		t.Fatal(err)
	}
	checkLocation(t, doc, putRight, "1:23:45")
	if strings.Contains(doc.Text, middle) {
		t.Errorf("the transcript still reads as what was heard:\n%s", doc.Text)
	}
}

// Taking away what a transcript was put right to gives back what was heard.
func TestATranscriptNothingPutRightReadsAsWhatWasHeard(t *testing.T) {
	store := shelf{domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"): writeTranscript()}

	doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
	if err != nil {
		t.Fatal(err)
	}
	checkLocation(t, doc, middle, "1:23:45")
}

// A sweep works through the names a producer writes, and what a transcript was
// put right to goes with the recording it belongs to.
func TestWhatATranscriptWasPutRightToIsSweptWithIt(t *testing.T) {
	store := shelf{
		domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"):        writeTranscript(),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"): writeCorrections(),
	}
	for _, target := range domain.GetArtifactNames(domain.ASR, "abc123") {
		if err := store.RemoveArtifact(t.Context(), target); err != nil {
			t.Fatal(err)
		}
	}
	if len(store) != 0 {
		t.Errorf("a sweep left %v behind", store)
	}
}
