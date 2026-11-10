package transcribe

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func TestTranscriptCreateOfAChangedRecordingStartsARun(t *testing.T) {
	s := newRequestFixture(t)
	s.putTranscript(t, true)
	changed := lectureSound + " and a second half"
	s.vault.File(lecture, changed)
	s.texts.File(lecture, []byte(changed))

	out := s.createTranscript(t, "Lectures", lecture)

	if out.State != "started" || out.Version != "" {
		t.Errorf("transcript_create of a changed recording returned %+v, want started", out)
	}
	close(s.blocking.release)
	s.waitFinished(t, out.TaskID)
}

func TestTranscriptCreateOverAnotherPluginsWholeVersionStartsARun(t *testing.T) {
	s := newRequestFixture(t)
	other := s.texts.As("md.numen.other")
	if err := host.WriteText(t.Context(), other.GetStore("md.numen.other"), "k.jsonl",
		[]domain.TextSegment{{Text: "other words\n"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := other.PutVersion(t.Context(), host.TextPut{
		Path: lecture, Key: "k", Kind: "transcript", InputHash: domain.HashContent([]byte(lectureSound)), File: "k.jsonl",
	}); err != nil {
		t.Fatal(err)
	}

	out := s.createTranscript(t, "Lectures", lecture)

	if out.State != "started" {
		t.Errorf("transcript_create over another plugin's live version returned %+v, want started", out)
	}
	close(s.blocking.release)
	s.waitFinished(t, out.TaskID)
}

// The host has the version recorded on the partial file while the plugin's folder has the whole
// transcript: a run renamed the partial file and stopped before PutVersion (Close, or a failed call).
func TestTranscriptCreateOverAStalePartialRecordReturnsTheVersion(t *testing.T) {
	s := newRequestFixture(t)
	close(s.blocking.release)
	s.putTranscript(t, false)
	key := getTranscriptionKey(domain.HashContent([]byte(lectureSound)))
	if err := (transcripts{store: s.store}).finish(t.Context(), key); err != nil {
		t.Fatal(err)
	}

	first := s.createTranscript(t, "Lectures", lecture)
	s.waitFinished(t, first.TaskID)
	second := s.createTranscript(t, "Lectures", lecture)

	if second.State != "done" {
		listing, err := s.texts.ListVersions(t.Context(), lecture)
		t.Errorf("after the run, transcript_create returned %+v, want done; the lecture lists %+v (%v)", second, listing, err)
	}
}
