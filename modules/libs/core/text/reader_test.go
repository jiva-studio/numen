package text_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// The talk is what a model heard, an hour and a half into a recording.
const (
	opening = "Today we shall speak of the holy name."
	middle  = "The name and the named are not two."
	closing = "That is what the acharyas have said."
)

// heard is the artifact a transcription of the talk leaves.
func heard() []byte {
	return transcript.Marshal([]transcript.Cue{
		{Text: opening, From: 1500, To: 4200},
		{Text: middle, From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
}

func TestATranscriptLocatesAPassageByWhenItWasSaid(t *testing.T) {
	doc := text.Transcribed(heard())

	located(t, doc, opening, "0:01")
	located(t, doc, middle, "1:23:45")
	located(t, doc, closing, "1:30:00")
}

// A transcription names no parts: the cues are where the speech was.
func TestATranscriptNamesNoParts(t *testing.T) {
	doc := text.Transcribed(heard())

	if len(doc.Parts) != 0 {
		t.Errorf("the transcript names %+v", doc.Parts)
	}
	if doc.Text == "" {
		t.Error("the transcript says nothing")
	}
}

// A transcript nothing put right is composed from its own bytes.
func TestATranscriptIsComposedFromItsOwnBytes(t *testing.T) {
	store := beside{text.Artifact(text.ASR, "abc123"): heard()}

	doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
	if err != nil {
		t.Fatal(err)
	}
	located(t, doc, middle, "1:23:45")
}

func TestATranscriptIsKeptUnderTheNameAPlayerKnowsItBy(t *testing.T) {
	if name := text.Artifact(text.ASR, "abc123"); name != "asr/abc123.vtt" {
		t.Errorf("a transcript is kept under %q", name)
	}
	if name := text.Partial(text.ASR, "abc123"); name != "asr/abc123.partial.vtt" {
		t.Errorf("a transcription still running is kept under %q", name)
	}
}

// A sweep works through the names a producer writes, and a transcription writes
// no coordinates, parts or corrections.
func TestASweepOfATranscriptNamesWhatItWrote(t *testing.T) {
	want := []string{
		"asr/abc123.vtt",
		"asr/abc123.partial.vtt",
		"asr/abc123.corrected.vtt",
		"asr/abc123.proofread",
		"asr/abc123.answer",
		"asr/abc123.json",
	}
	if got := text.Names(text.ASR, "abc123"); !slices.Equal(got, want) {
		t.Errorf("a sweep takes %v, want %v", got, want)
	}
}

// A vault already read is not read again, so the names a reading is kept under
// are the ones it was written under.
func TestAReadingIsKeptUnderTheNamesItAlwaysWas(t *testing.T) {
	want := []string{
		"ocr/abc123.txt",
		"ocr/abc123.partial",
		"ocr/abc123.boxes",
		"ocr/abc123.parts",
		"ocr/abc123.fixes",
		"ocr/abc123.proofread",
		"ocr/abc123.json",
	}
	if got := text.Names("ocr", "abc123"); !slices.Equal(got, want) {
		t.Errorf("a reading is kept under %v, want %v", got, want)
	}
}

// A transcript is composed from what it was put right to, and what was heard
// stands under its own name.
func TestATranscriptIsComposedFromWhatItWasPutRightTo(t *testing.T) {
	put := transcript.Marshal([]transcript.Cue{
		{Text: opening, From: 1500, To: 4200},
		{Text: "The name and the Named are not two.", From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
	store := beside{
		text.Artifact(text.ASR, "abc123"):    heard(),
		text.Corrections(text.ASR, "abc123"): append(put, transcript.Hand()...),
	}

	doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
	if err != nil {
		t.Fatal(err)
	}
	located(t, doc, "The name and the Named are not two.", "1:23:45")
	if strings.Contains(doc.Text, middle) {
		t.Errorf("what was heard is still the text:\n%s", doc.Text)
	}
	if strings.Contains(doc.Text, transcript.ByHand) {
		t.Errorf("the mark of a person's own words is in the text:\n%s", doc.Text)
	}
}

// A file beside the artifact holding nothing is nothing put right, and what was
// heard is what the source says.
func TestATranscriptPutRightToNothingIsWhatWasHeard(t *testing.T) {
	for _, one := range []struct {
		what string
		put  []byte
	}{
		{"a file holding no bytes", nil},
		{"a file holding no cues", []byte(transcript.Head + "\n")},
	} {
		t.Run(one.what, func(t *testing.T) {
			store := beside{
				text.Artifact(text.ASR, "abc123"):    heard(),
				text.Corrections(text.ASR, "abc123"): one.put,
			}
			doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(doc.Text, middle) {
				t.Errorf("the source says %q", doc.Text)
			}
		})
	}
}

// A run stopped part way through a cue leaves a block that is not one. It is
// passed over, and the cues before it stand where they were said.
func TestATranscriptTornMidCueIsReadAsFarAsItGoes(t *testing.T) {
	torn := append(heard(), "\n00:1"...)
	store := beside{
		text.Artifact(text.ASR, "abc123"):    heard(),
		text.Corrections(text.ASR, "abc123"): torn,
	}

	doc, err := text.Composed(t.Context(), store, text.ASR, "abc123", heard())
	if err != nil {
		t.Fatal(err)
	}
	located(t, doc, closing, "1:30:00")
	if strings.Contains(doc.Text, "00:1") {
		t.Errorf("the torn block was read as speech:\n%s", doc.Text)
	}
}

// A link note is its prose and what was fetched for it, as one text. What this
// holds to is that a place in the fetched half is still the place it was: a
// passage found there is located by when it was said, and the prose standing
// before it moves every offset by exactly its own length.
func TestALinkNoteIsItsProseAndThenWhatWasFetched(t *testing.T) {
	prose := strings.Repeat("What I made of it. ", 20)
	doc := text.Joined(prose, text.Transcribed(heard()))

	if !strings.HasPrefix(doc.Text, prose+text.Separator) {
		t.Errorf("the note reads %q, want its prose first", doc.Text)
	}
	if at := strings.Index(doc.Text, opening); at != len(prose)+len(text.Separator) {
		t.Errorf("what was fetched begins at %d, want %d", at, len(prose)+len(text.Separator))
	}
	located(t, doc, opening, "0:01")
	located(t, doc, middle, "1:23:45")
	located(t, doc, closing, "1:30:00")
}
