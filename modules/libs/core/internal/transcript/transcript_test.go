package transcript_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
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
	if said[cues[1].Offset:cues[1].Offset+len(cues[1].Text)] != "вторая реплика" {
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
	written = append(written, transcript.Reaches(2000)...)

	said, cues := transcript.Parse(written)
	if said != "said" || len(cues) != 1 {
		t.Errorf("the note was read as speech: %q", said)
	}
	if ms, end := transcript.ReadReached(written); ms != 2000 || end != len(written) {
		t.Errorf("the note says %d ms and ends at %d of %d", ms, end, len(written))
	}
}

// A batch that did not land whole is one no note claims, and the run before it
// is what the next run takes up.
func TestABatchNoNoteClaims(t *testing.T) {
	whole := transcript.Marshal([]transcript.Cue{{Text: "said", From: 0, To: 2000}})
	whole = append(whole, transcript.Reaches(2000)...)
	torn := slices.Concat(whole, []byte("\n00:00:02.000 --> 00:00:0"))

	ms, end := transcript.ReadReached(torn)
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

// A run of the words is in the cues it crosses, and in none where it is past
// them.
func TestARunIsInTheCuesItCrosses(t *testing.T) {
	_, cues := transcript.Parse(transcript.Marshal([]transcript.Cue{
		{Text: "first", From: 0, To: 1000},
		{Text: "second", From: 1000, To: 2000},
		{Text: "third", From: 2000, To: 3000},
	}))

	// "second" begins after "first\n".
	if got := transcript.At(cues, 6, 6); len(got) != 1 || got[0].From != 1000 {
		t.Errorf("the run is in %d cues, the first from %v", len(got), got)
	}
	if got := transcript.At(cues, 6, 8); len(got) != 2 {
		t.Errorf("a run crossing into the third cue is in %d cues, want 2", len(got))
	}
	if got := transcript.At(cues, 900, 5); len(got) != 0 {
		t.Errorf("a run past the words is in %d cues, want none", len(got))
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

// The mark a person's own words carry is a note of its own. The same words
// spoken in a cue are speech, and a proofreader is not stopped by them.
func TestTheMarkOfAPersonsWordsIsANoteAndNotSpeech(t *testing.T) {
	spoken := transcript.Marshal([]transcript.Cue{
		{Text: transcript.ByHand, From: 0, To: 2000},
		{Text: "and then he said it again", From: 2000, To: 4000},
	})
	if transcript.IsWrittenByHand(spoken) {
		t.Errorf("a cue saying %q was read as a person's own words:\n%s", transcript.ByHand, spoken)
	}

	own := slices.Concat(spoken, transcript.Hand())
	if !transcript.IsWrittenByHand(own) {
		t.Errorf("the mark was not read:\n%s", own)
	}

	// The note is passed over, and the words are the ones that were spoken.
	said, cues := transcript.Parse(own)
	if len(cues) != 2 || said != transcript.ByHand+"\nand then he said it again" {
		t.Errorf("the words read back as %q in %d cues", said, len(cues))
	}
}

// How far a run got is a note of its own. A cue saying the same words is
// speech, and nothing is cut away behind it.
func TestHowFarARunGotIsANoteAndNotSpeech(t *testing.T) {
	claimed := transcript.Marshal([]transcript.Cue{{Text: "said", From: 0, To: 2000}})
	claimed = append(claimed, transcript.Reaches(2000)...)

	// A batch that did not land whole, speaking the words a note is written in.
	spoken := transcript.Marshal([]transcript.Cue{{Text: "NOTE heard 9999", From: 2000, To: 4000}})
	whole := slices.Concat(claimed, bytes.TrimPrefix(spoken, []byte(transcript.Head+"\n")))

	ms, end := transcript.ReadReached(whole)
	if ms != 2000 {
		t.Errorf("the run is said to have reached %d ms", ms)
	}
	if string(whole[:end]) != string(claimed) {
		t.Errorf("cutting back to the note leaves %q", whole[:end])
	}
}

// A moment of a recording as a player writes one. An hour that is not there is
// not written.
func TestClockIsAMomentAsAPlayerWritesOne(t *testing.T) {
	for ms, want := range map[int]string{
		-1:      "0:00",
		0:       "0:00",
		9100:    "0:09",
		61000:   "1:01",
		3599999: "59:59",
		3723456: "1:02:03",
	} {
		if got := transcript.Clock(ms); got != want {
			t.Errorf("%d ms reads as %q, want %q", ms, got, want)
		}
	}
}

// A note stands at the top of its own block, so the same words spoken inside a
// cue say nothing about how far a run got.
func TestANoteInsideALineIsNotANote(t *testing.T) {
	whole := transcript.Marshal([]transcript.Cue{
		{Text: "he said NOTE heard 999 and sat down", From: 0, To: 2000},
	})
	whole = append(whole, transcript.Reaches(2000)...)

	if ms, end := transcript.ReadReached(whole); ms != 2000 || end != len(whole) {
		t.Errorf("the note says %d ms and ends at %d of %d", ms, end, len(whole))
	}
}

// A file carrying the words of a note inside every one of its cues is read in
// one pass over it.
func TestReachedReadsALargeTranscriptAtOnce(t *testing.T) {
	var file strings.Builder
	file.WriteString(transcript.Head + "\n")
	for i := range 200_000 {
		fmt.Fprintf(&file, "\n00:00:0%d.000 --> 00:00:0%d.000\nNOTE heard %d said\n", i%9, i%9+1, i)
	}
	raw := []byte(file.String())

	done := make(chan int, 1)
	go func() {
		ms, _ := transcript.ReadReached(raw)
		done <- ms
	}()
	select {
	case ms := <-done:
		if ms != 0 {
			t.Errorf("a note inside a cue said %d ms", ms)
		}
	case <-time.After(30 * time.Second):
		t.Error("reading a transcript of 10 MB did not finish")
	}
}
