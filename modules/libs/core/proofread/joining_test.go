package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A sentence a recording broke across three stretches of speech.
func broken() proofread.Batch {
	return proofread.Batch{At: 1, Joining: true, Lines: []proofread.Line{
		{At: 4, Through: 4, Text: "Krishna is Raj. Krishna is"},
		{At: 5, Through: 5, Text: "connected with Raj Dila."},
		{At: 6, Through: 6, Text: "Sure."},
	}}
}

// A run of lines is answered for as one, and the answer says which lines it
// covers.
func TestASentenceBrokenAcrossLinesIsPutBackTogether(t *testing.T) {
	put, ok := proofread.Fixed(broken(), "4-5 | Krishna is Radha, Krishna is connected with Radhika.", 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 {
		t.Fatalf("%v put right, want one line", put)
	}
	if put[0].At != 4 || put[0].Through != 5 || !put[0].Joins() {
		t.Errorf("the run stands at %d-%d", put[0].At, put[0].Through)
	}
}

// A run answered with exactly what its lines already say still puts them
// together.
func TestARunSayingWhatItsLinesSayStillJoinsThem(t *testing.T) {
	put, ok := proofread.Fixed(broken(), "4-5 | Krishna is Raj. Krishna is connected with Raj Dila.", 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 || !put[0].Joins() {
		t.Errorf("%v put right", put)
	}
}

// A run reaching past the lines the batch carries is a run answering about
// lines nobody asked about.
func TestARunReachingPastTheBatchRefusesIt(t *testing.T) {
	if _, ok := proofread.Fixed(broken(), "5-9 | whatever it says", 0); ok {
		t.Error("the batch was taken")
	}
}

// A run written backwards is not a run.
func TestARunWrittenBackwardsRefusesTheBatch(t *testing.T) {
	if _, ok := proofread.Fixed(broken(), "6-4 | whatever it says", 0); ok {
		t.Error("the batch was taken")
	}
}

// The printed lines of a page stay where they were printed, so a reply putting
// two of them together is no answer to what was asked.
func TestARunRefusesABatchThatDoesNotPutLinesTogether(t *testing.T) {
	page := proofread.Scanned("one two three ", []lit.Box{
		box(4, 0, 4), box(4, 4, 4), box(4, 8, 6),
	})[0]
	if page.Joining {
		t.Fatal("a page of a scan puts its lines together")
	}
	if _, ok := proofread.Fixed(page, "0-1|one two", 0.30); ok {
		t.Error("the batch was taken")
	}
	// The lines of the page are still answered for one at a time.
	if _, ok := proofread.Fixed(page, "0|won", 0.30); !ok {
		t.Error("the batch was refused")
	}
}

// Speech runs on past the line it was cut into, so a batch of it puts lines
// together.
func TestABatchOfSpeechPutsLinesTogether(t *testing.T) {
	batches := proofread.Spoken([]transcript.Cue{
		{Text: "Krishna is Raj. Krishna is", From: 0, To: 800},
		{Text: "connected with Raj Dila.", From: 1000, To: 1800},
	}, 2, 0)
	if len(batches) != 1 || !batches[0].Joining {
		t.Fatalf("the batches are %+v", batches)
	}
}

// An answer of marks that are not words empties a line as surely as an answer
// of nothing does.
func TestAnAnswerOfNothingButMarksRefusesTheBatch(t *testing.T) {
	for _, reply := range []string{"4 | ...", "4 | —", "4 | "} {
		if _, ok := proofread.Fixed(broken(), reply, 0); ok {
			t.Errorf("the batch was taken with %q", reply)
		}
	}
}
