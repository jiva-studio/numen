package proofread_test

import (
	"reflect"
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
	put, _, ok := proofread.Fixed(broken(), "4-5 | Krishna is Radha, Krishna is connected with Radhika.", 0)
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
	put, _, ok := proofread.Fixed(broken(), "4-5 | Krishna is Raj. Krishna is connected with Raj Dila.", 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 || !put[0].Joins() {
		t.Errorf("%v put right", put)
	}
}

// A sentence running on past the last line a batch was given is named whole by
// a model reading it. That run is dropped, the rest of the batch stands, and
// the reply says a sentence ran past the end.
func TestARunReachingPastTheBatchIsDroppedAndTheRestStands(t *testing.T) {
	put, past, ok := proofread.Fixed(broken(), "5-9 | whatever it says\n6 | Sure of it.", 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 || put[0].At != 6 || put[0].Text != "Sure of it." {
		t.Errorf("%v put right", put)
	}
	if !past {
		t.Error("the reply does not say a sentence ran past the end")
	}
}

// A reply whose runs all stand inside the batch says nothing ran past the end.
func TestARunInsideTheBatchSaysNothingRanPastIt(t *testing.T) {
	for _, reply := range []string{"4-5 | Krishna is Radha, connected with Radhika.", "6 | Sure of it.", ""} {
		if _, past, ok := proofread.Fixed(broken(), reply, 0); !ok || past {
			t.Errorf("%q answered %v and ran past %v", reply, ok, past)
		}
	}
}

// A batch the gates refuse says nothing about what ran past its end.
func TestARefusedBatchSaysNothingRanPastIt(t *testing.T) {
	if _, past, ok := proofread.Fixed(broken(), "5-9 | whatever it says\n9 | no such line", 0); ok || past {
		t.Errorf("the batch answered %v and ran past %v", ok, past)
	}
}

// A run skipping a line the batch carries is an answer against the wrong
// numbers.
func TestARunSkippingALineRefusesTheBatch(t *testing.T) {
	batch := broken()
	batch.Lines = append(batch.Lines[:1], batch.Lines[2:]...)
	if _, _, ok := proofread.Fixed(batch, "4-6 | whatever it says", 0); ok {
		t.Error("the batch was taken")
	}
}

// A run written backwards is not a run.
func TestARunWrittenBackwardsRefusesTheBatch(t *testing.T) {
	if _, _, ok := proofread.Fixed(broken(), "6-4 | whatever it says", 0); ok {
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
	if _, _, ok := proofread.Fixed(page, "0-1|one two", 0.30); ok {
		t.Error("the batch was taken")
	}
	// The lines of the page are still answered for one at a time.
	if _, _, ok := proofread.Fixed(page, "0|won", 0.30); !ok {
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
		if _, _, ok := proofread.Fixed(broken(), reply, 0); ok {
			t.Errorf("the batch was taken with %q", reply)
		}
	}
}

// Gathered names the batches whose reply ran on past the last line they carry,
// in the order they were asked. A batch it refuses names none.
func TestGatheredNamesTheBatchesARunRanPast(t *testing.T) {
	first, second := broken(), broken()
	first.At, second.At = 1, 2
	replies := map[int]string{
		1: "5-9 | whatever it says",
		2: "4-5 | Krishna is Radha, Krishna is connected with Radhika.",
	}

	put, past := proofread.Gathered([]proofread.Batch{first, second}, replies, 0)
	if len(put) != 1 {
		t.Fatalf("%v put right", put)
	}
	if !reflect.DeepEqual(past, []int{1}) {
		t.Errorf("the batches a run went past are %v, want the first alone", past)
	}
}
