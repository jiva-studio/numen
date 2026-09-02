package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// A sentence a recording broke across three stretches of speech.
func broken() proofread.Batch {
	return proofread.Batch{At: 1, Lines: []proofread.Line{
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
