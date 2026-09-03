package proofread_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A sentence a recording broke across three stretches of speech.
func broken() proofread.Batch {
	return proofread.Batch{Number: 1, Joinable: true, Lines: []proofread.Line{
		{Number: 4, Last: 4, Text: "Krishna is Raj. Krishna is"},
		{Number: 5, Last: 5, Text: "connected with Raj Dila."},
		{Number: 6, Last: 6, Text: "Sure."},
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
	if put[0].Number != 4 || put[0].Last != 5 || !put[0].Joins() {
		t.Errorf("the run stands at %d-%d", put[0].Number, put[0].Last)
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
	if len(put) != 1 || put[0].Number != 6 || put[0].Text != "Sure of it." {
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
	page := proofread.Scanned("one two three ", []highlight.Box{
		box(4, 0, 4), box(4, 4, 4), box(4, 8, 6),
	})[0]
	if page.Joinable {
		t.Fatal("a page of a scan puts its lines together")
	}
	if _, _, ok := proofread.Fixed(page, "0-1|one two", 0.30); ok {
		t.Error("the batch was taken")
	}
	// A printed line is answered for once, as a stretch of speech is.
	if _, _, ok := proofread.Fixed(page, "0|won\n0|wan", 0.30); ok {
		t.Error("the batch was taken with a line answered for twice")
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
	if len(batches) != 1 || !batches[0].Joinable {
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
	first.Number, second.Number = 1, 2
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

// A line where one sentence ends and the next begins stands in the run of
// both, and the run is answered with every sentence it covers.
func TestARunHoldsEverySentenceItsLinesCarry(t *testing.T) {
	batch := proofread.Batch{Number: 1, Joinable: true, Lines: []proofread.Line{
		{Number: 4, Last: 4, Text: "we should go. And then"},
		{Number: 5, Last: 5, Text: "the next day he left."},
	}}
	said := "We should go. And then the next day he left."

	put, _, ok := proofread.Fixed(batch, "4-5 | "+said, 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 || put[0].Number != 4 || put[0].Last != 5 || put[0].Text != said {
		t.Errorf("%v put right", put)
	}
}

// Two runs sharing a line leave the words of that line in one answer or the
// other, and the batch is refused.
func TestTwoRunsSharingALineRefuseTheBatch(t *testing.T) {
	for _, reply := range []string{
		"4-5 | Krishna is Radha.\n5-6 | Connected with Radhika, sure.",
		"4-5 | Krishna is Radha.\n5 | Connected with Radhika.",
		"4 | Krishna is Radha.\n4 | Krishna is Radhika.",
	} {
		if _, _, ok := proofread.Fixed(broken(), reply, 0); ok {
			t.Errorf("the batch was taken with %q", reply)
		}
	}
}
