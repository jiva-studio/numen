package transcript_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// What is written is read back as the same cues, and the words come back
// without a timing in them.
func TestWhatIsWrittenIsReadBack(t *testing.T) {
	written := transcript.Marshal([]transcript.Cue{
		{Text: "первая реплика", From: 1500, To: 4200},
		{Text: "вторая реплика", From: 4200, To: 9100},
	})

	if !strings.HasPrefix(string(written), transcript.Head) {
		t.Fatalf("the file does not begin with %s:\n%s", transcript.Head, written)
	}

	said, cues := transcript.Parse(written)
	if said != "первая реплика\nвторая реплика" {
		t.Errorf("the words read back as %q", said)
	}
	if len(cues) != 2 {
		t.Fatalf("read back %d cues, want 2", len(cues))
	}
	if cues[0].From != 1500 || cues[0].To != 4200 {
		t.Errorf("the first cue spans %d-%d", cues[0].From, cues[0].To)
	}
	if said[cues[1].At:cues[1].At+len(cues[1].Text)] != "вторая реплика" {
		t.Errorf("the second cue does not stand where it says it does")
	}
}

// A cue carrying no words is not a moment the recording had.
func TestSilenceIsNotWritten(t *testing.T) {
	written := transcript.Marshal([]transcript.Cue{
		{Text: "  ", From: 0, To: 1000},
		{Text: "said", From: 1000, To: 2000},
	})
	said, cues := transcript.Parse(written)
	if said != "said" || len(cues) != 1 {
		t.Errorf("silence was written down: %q, %d cues", said, len(cues))
	}
}

// A note says how far a run got, and is not part of what was said.
func TestANoteIsNotSpeech(t *testing.T) {
	written := transcript.Marshal([]transcript.Cue{{Text: "said", From: 0, To: 2000}})
	written = append(written, transcript.Heard(2000)...)

	said, cues := transcript.Parse(written)
	if said != "said" || len(cues) != 1 {
		t.Errorf("the note was read as speech: %q", said)
	}
	if ms, end := transcript.Reached(written); ms != 2000 || end != len(written) {
		t.Errorf("the note says %d ms and ends at %d of %d", ms, end, len(written))
	}
}

// A batch that did not land whole is one no note claims, and the run before it
// is what the next run takes up.
func TestABatchNoNoteClaims(t *testing.T) {
	whole := transcript.Marshal([]transcript.Cue{{Text: "said", From: 0, To: 2000}})
	whole = append(whole, transcript.Heard(2000)...)
	torn := append(whole, []byte("\n00:00:02.000 --> 00:00:0")...)

	ms, end := transcript.Reached(torn)
	if ms != 2000 {
		t.Errorf("the last note says %d ms", ms)
	}
	if string(torn[:end]) != string(whole) {
		t.Errorf("cutting back to the note leaves %q", torn[:end])
	}
}

// The hours are optional in the format, and other tools write them out.
func TestTimingsOtherToolsWrite(t *testing.T) {
	said, cues := transcript.Parse([]byte("WEBVTT\n\n01:02.500 --> 00:01:03.000\nsaid\n"))
	if said != "said" || len(cues) != 1 {
		t.Fatalf("read %q and %d cues", said, len(cues))
	}
	if cues[0].From != 62500 || cues[0].To != 63000 {
		t.Errorf("the cue spans %d-%d, want 62500-63000", cues[0].From, cues[0].To)
	}
}

// A run of the words is played from the cue it begins in.
func TestARunIsPlayedFromItsCue(t *testing.T) {
	_, cues := transcript.Parse(transcript.Marshal([]transcript.Cue{
		{Text: "first", From: 0, To: 1000},
		{Text: "second", From: 1000, To: 2000},
		{Text: "third", From: 2000, To: 3000},
	}))

	// "second" begins after "first\n".
	ms, ok := transcript.Plays(cues, 6, 6)
	if !ok || ms != 1000 {
		t.Errorf("the run plays from %d ms (found %v)", ms, ok)
	}
	if got := transcript.At(cues, 6, 8); len(got) != 2 {
		t.Errorf("a run crossing into the third cue is in %d cues, want 2", len(got))
	}
	if _, ok := transcript.Plays(cues, 900, 5); ok {
		t.Errorf("a run past the words was placed in the recording")
	}
}

func TestStampIsWhatTheFormatWrites(t *testing.T) {
	for ms, want := range map[int]string{
		0:        "00:00:00.000",
		9100:     "00:00:09.100",
		3723456:  "01:02:03.456",
		86399999: "23:59:59.999",
	} {
		if got := transcript.Stamp(ms); got != want {
			t.Errorf("%d ms is written %q, want %q", ms, got, want)
		}
	}
}
