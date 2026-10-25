package text_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// The talk is what a model heard, an hour and a half into a recording.
const (
	opening = "Today we shall speak of the holy name."
	middle  = "The name and the named are not two."
	closing = "That is what the acharyas have said."
)

// writeTranscript is the artifact a transcription of the talk leaves.
func writeTranscript() []byte {
	return transcript.Marshal([]domain.TranscriptCue{
		{Text: opening, From: 1500, To: 4200},
		{Text: middle, From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
}

func TestATranscriptLocatesAPassageByWhenItWasSaid(t *testing.T) {
	doc := text.ReadTranscript(writeTranscript())

	checkLocation(t, doc, opening, "0:01")
	checkLocation(t, doc, middle, "1:23:45")
	checkLocation(t, doc, closing, "1:30:00")
}

// A transcription names no parts: the cues are where the speech was.
func TestATranscriptNamesNoParts(t *testing.T) {
	doc := text.ReadTranscript(writeTranscript())

	if len(doc.Parts) != 0 {
		t.Errorf("the transcript names %+v", doc.Parts)
	}
	if doc.Text == "" {
		t.Error("the transcript says nothing")
	}
}

// A transcript nothing put right is composed from its own bytes.
func TestATranscriptIsComposedFromItsOwnBytes(t *testing.T) {
	store := artifactMap{domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"): writeTranscript()}

	doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
	if err != nil {
		t.Fatal(err)
	}
	checkLocation(t, doc, middle, "1:23:45")
}

// A transcript is composed from what it was put right to, and what was heard
// is placed under its own name.
func TestATranscriptIsComposedFromWhatItWasInsertRightTo(t *testing.T) {
	cuesData := transcript.Marshal([]domain.TranscriptCue{
		{Text: opening, From: 1500, To: 4200},
		{Text: "The name and the Named are not two.", From: 5025000, To: 5028000},
		{Text: closing, From: 5400000, To: 5403500},
	})
	store := artifactMap{
		domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"):        writeTranscript(),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"): append(cuesData, transcript.GetHandWrittenMark()...),
	}

	doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
	if err != nil {
		t.Fatal(err)
	}
	checkLocation(t, doc, "The name and the Named are not two.", "1:23:45")
	if strings.Contains(doc.Text, middle) {
		t.Errorf("what was heard is still the text:\n%s", doc.Text)
	}
	if strings.Contains(doc.Text, transcript.HandWrittenNote) {
		t.Errorf("the mark of a person's own words is in the text:\n%s", doc.Text)
	}
}

// A file beside the artifact holding nothing is nothing put right, and what was
// heard is what the source says.
func TestATranscriptInsertRightToNothingIsWhatWasHeard(t *testing.T) {
	for _, one := range []struct {
		what    string
		applied []byte
	}{
		{"a file holding no bytes", nil},
		{"a file holding no cues", []byte(transcript.Head + "\n")},
	} {
		t.Run(one.what, func(t *testing.T) {
			store := artifactMap{
				domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"):        writeTranscript(),
				domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"): one.applied,
			}
			doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
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
	divided := append(writeTranscript(), "\n00:1"...)
	store := artifactMap{
		domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"):        writeTranscript(),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"): divided,
	}

	doc, err := text.ReadComposed(t.Context(), store, domain.ASR, "abc123", writeTranscript())
	if err != nil {
		t.Fatal(err)
	}
	checkLocation(t, doc, closing, "1:30:00")
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
	doc := text.JoinDocuments(prose, text.ReadTranscript(writeTranscript()))

	if !strings.HasPrefix(doc.Text, prose+text.Separator) {
		t.Errorf("the note reads %q, want its prose first", doc.Text)
	}
	if pos := strings.Index(doc.Text, opening); pos != len(prose)+len(text.Separator) {
		t.Errorf("what was fetched begins at %d, want %d", pos, len(prose)+len(text.Separator))
	}
	checkLocation(t, doc, opening, "0:01")
	checkLocation(t, doc, middle, "1:23:45")
	checkLocation(t, doc, closing, "1:30:00")
}
