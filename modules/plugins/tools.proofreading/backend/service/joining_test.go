package service

import "testing"

// A sentence a recording broke across three stretches of speech.
func makeBrokenBatch() Batch {
	return Batch{Number: 1, IsJoinable: true, Lines: []Line{
		{Number: 4, Last: 4, Text: "Krishna is Raj. Krishna is"},
		{Number: 5, Last: 5, Text: "connected with Raj Dila."},
		{Number: 6, Last: 6, Text: "Sure."},
	}}
}

// A run of lines is answered for as one, and the answer says which lines it
// covers.
func TestASentenceBrokenAcrossLinesIsPutBackTogether(t *testing.T) {
	placed, _, ok := GetFixedLines(makeBrokenBatch(), "4-5 | Krishna is Radha, Krishna is connected with Radhika.", 0)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 1 {
		t.Fatalf("%v put right, want one line", placed)
	}
	if placed[0].Number != 4 || placed[0].Last != 5 || !placed[0].IsJoined() {
		t.Errorf("the run stands at %d-%d", placed[0].Number, placed[0].Last)
	}
}

// A run answered with exactly what its lines already say still puts them
// together.
func TestARunSayingWhatItsLinesSayStillJoinsThem(t *testing.T) {
	placed, _, ok := GetFixedLines(makeBrokenBatch(), "4-5 | Krishna is Raj. Krishna is connected with Raj Dila.", 0)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 1 || !placed[0].IsJoined() {
		t.Errorf("%v put right", placed)
	}
}

// A sentence running on past the last line a batch was given is named whole by
// a model reading it. That run is dropped, the rest of the batch stands, and
// the reply says a sentence ran past the end.
func TestARunReachingPastTheBatchIsDroppedAndTheRestStands(t *testing.T) {
	placed, past, ok := GetFixedLines(makeBrokenBatch(), "5-9 | whatever it says\n6 | Sure of it.", 0)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 1 || placed[0].Number != 6 || placed[0].Text != "Sure of it." {
		t.Errorf("%v put right", placed)
	}
	if !past {
		t.Error("the reply does not say a sentence ran past the end")
	}
}

// A reply whose runs all stand inside the batch says nothing ran past the end.
func TestARunInsideTheBatchSaysNothingRanPastIt(t *testing.T) {
	for _, reply := range []string{"4-5 | Krishna is Radha, connected with Radhika.", "6 | Sure of it.", ""} {
		if _, past, ok := GetFixedLines(makeBrokenBatch(), reply, 0); !ok || past {
			t.Errorf("%q answered %v and ran past %v", reply, ok, past)
		}
	}
}

// A batch the gates refuse says nothing about what ran past its end.
func TestARefusedBatchSaysNothingRanPastIt(t *testing.T) {
	if _, past, ok := GetFixedLines(makeBrokenBatch(), "5-9 | whatever it says\n9 | no such line", 0); ok || past {
		t.Errorf("the batch answered %v and ran past %v", ok, past)
	}
}

// A run skipping a line the batch carries is an answer against the wrong
// numbers.
func TestARunSkippingALineRefusesTheBatch(t *testing.T) {
	batch := makeBrokenBatch()
	batch.Lines = append(batch.Lines[:1], batch.Lines[2:]...)
	if _, _, ok := GetFixedLines(batch, "4-6 | whatever it says", 0); ok {
		t.Error("the batch was taken")
	}
}

// A run written backwards is not a run.
func TestARunWrittenBackwardsRefusesTheBatch(t *testing.T) {
	if _, _, ok := GetFixedLines(makeBrokenBatch(), "6-4 | whatever it says", 0); ok {
		t.Error("the batch was taken")
	}
}

// An answer of marks that are not words empties a line as surely as an answer
// of nothing does.
func TestAnAnswerOfNothingButMarksRefusesTheBatch(t *testing.T) {
	for _, reply := range []string{"4 | ...", "4 | —", "4 | "} {
		if _, _, ok := GetFixedLines(makeBrokenBatch(), reply, 0); ok {
			t.Errorf("the batch was taken with %q", reply)
		}
	}
}

func TestARunHoldsEverySentenceItsLinesCarry(t *testing.T) {
	batch := Batch{Number: 1, IsJoinable: true, Lines: []Line{
		{Number: 4, Last: 4, Text: "we should go. And then"},
		{Number: 5, Last: 5, Text: "the next day he left."},
	}}
	response := "We should go. And then the next day he left."

	placed, _, ok := GetFixedLines(batch, "4-5 | "+response, 0)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 1 || placed[0].Number != 4 || placed[0].Last != 5 || placed[0].Text != response {
		t.Errorf("%v put right", placed)
	}
}

// Two runs sharing a line leave the words of that line in one answer or the
// other, and the batch fails.
func TestTwoRunsSharingALineRefuseTheBatch(t *testing.T) {
	for _, reply := range []string{
		"4-5 | Krishna is Radha.\n5-6 | Connected with Radhika, sure.",
		"4-5 | Krishna is Radha.\n5 | Connected with Radhika.",
		"4 | Krishna is Radha.\n4 | Krishna is Radhika.",
	} {
		if _, _, ok := GetFixedLines(makeBrokenBatch(), reply, 0); ok {
			t.Errorf("the batch was taken with %q", reply)
		}
	}
}
