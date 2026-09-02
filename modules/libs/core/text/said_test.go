package text_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// The talk as a person left it, with the name spelled the way the speaker
// spelled it.
const putRight = "The name and the named are not two, said Rupa."

// beside is a store holding what a run left, by the name it left it under.
type beside map[string][]byte

func (b beside) Read(_ context.Context, name string) ([]byte, error) {
	raw, held := b[name]
	if !held {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (b beside) Write(_ context.Context, name string, content []byte) error {
	b[name] = content
	return nil
}

func (b beside) Append(_ context.Context, name string, content []byte) error {
	b[name] = append(b[name], content...)
	return nil
}

func (b beside) Remove(_ context.Context, name string) error {
	delete(b, name)
	return nil
}

func (b beside) List(context.Context, string) ([]port.Stored, error) { return nil, nil }

func (b beside) Claim(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}

// said is the transcript of the talk as somebody put it right.
func said() []byte {
	return transcript.Marshal([]transcript.Cue{
		{Text: opening, From: 1500, To: 4200},
		{Text: putRight, From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
}

// The words a transcript was put right to are the words it reads as, and the
// moments they were said at stand where they were.
func TestATranscriptPutRightReadsAsTheWordsItWasPutRightTo(t *testing.T) {
	store := beside{
		text.Artifact(text.ASR, "abc123"): heard(),
		text.Said(text.ASR, "abc123"):     said(),
	}

	doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
	if err != nil {
		t.Fatal(err)
	}
	located(t, doc, putRight, "1:23:45")
	if strings.Contains(doc.Text, middle) {
		t.Errorf("the transcript still reads as what was heard:\n%s", doc.Text)
	}
}

// Taking away what a transcript was put right to gives back what was heard.
func TestATranscriptNothingPutRightReadsAsWhatWasHeard(t *testing.T) {
	store := beside{text.Artifact(text.ASR, "abc123"): heard()}

	doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
	if err != nil {
		t.Fatal(err)
	}
	located(t, doc, middle, "1:23:45")
}

// A sweep works through the names a producer writes, and what a transcript was
// put right to goes with the recording it belongs to.
func TestWhatATranscriptWasPutRightToIsSweptWithIt(t *testing.T) {
	name := text.Said(text.ASR, "abc123")
	if name != "asr/abc123.said" {
		t.Errorf("a transcript put right is kept under %q", name)
	}

	store := beside{
		text.Artifact(text.ASR, "abc123"): heard(),
		name:                              said(),
	}
	for _, held := range text.Names(text.ASR, "abc123") {
		if err := store.Remove(t.Context(), held); err != nil {
			t.Fatal(err)
		}
	}
	if len(store) != 0 {
		t.Errorf("a sweep left %v behind", store)
	}
}
