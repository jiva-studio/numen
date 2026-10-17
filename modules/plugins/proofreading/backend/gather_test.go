package main

import (
	"reflect"
	"testing"
)

// gatherReplies is what the replies put right, as gatherAnswers gathers them.
func gatherReplies(asked []Batch, replies map[int]string, maxDistance float64) (map[int]Line, []int) {
	var answers []answer
	for _, batch := range asked {
		reply, answered := replies[batch.Number]
		if !answered {
			continue
		}
		answers = append(answers, readAnswer(batch, reply, maxDistance))
	}
	return gatherAnswers(answers)
}

// makeLine is one stretch of speech as the machine heard it.
func makeLine(at int) Line {
	return Line{Number: at, Text: []string{
		"",
		"the ferry left at noone",
		"and the sea was calm",
		"we spoke of the harbour lites",
		"until the fog came in",
		"and the bell began to ring",
	}[at]}
}

// wide and narrow both answer about lines 3 and 4. In wide the two stand
// further from the end.
func wide() Batch {
	return Batch{Number: 0, Lines: []Line{
		makeLine(1), makeLine(2), makeLine(3), makeLine(4), makeLine(5),
	}}
}

func narrow() Batch {
	return Batch{Number: 1, Lines: []Line{makeLine(3), makeLine(4)}}
}

func TestACorrectionComesFromTheBatchThatSawMoreOfWhatFollows(t *testing.T) {
	replies := map[int]string{
		0: "3|we spoke of the harbour lights",
		1: "3|we spoke of the harbor lite",
	}

	for _, asked := range [][]Batch{
		{wide(), narrow()},
		{narrow(), wide()},
	} {
		put, _ := gatherReplies(asked, replies, MaxEditDistance)
		if len(put) != 1 {
			t.Fatalf("%v put right, want line 3 alone", put)
		}
		if put[3].Text != "we spoke of the harbour lights" {
			t.Errorf("line 3 says %q, want the batch that saw the four lines after it", put[3].Text)
		}
	}
}

func TestBatchesSeeingAsMuchAsEachOtherAreSettledByTheLaterOne(t *testing.T) {
	early := Batch{Number: 0, Lines: []Line{makeLine(1), makeLine(2), makeLine(3)}}
	late := Batch{Number: 1, Lines: []Line{makeLine(2), makeLine(3)}}
	replies := map[int]string{
		0: "3|we spoke of the harbour lights",
		1: "3|we spoke of the harbor lite",
	}

	put, _ := gatherReplies([]Batch{early, late}, replies, MaxEditDistance)
	if put[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the later batch", put[3].Text)
	}
}

func TestARefusedReplyLeavesItsLinesToTheOtherBatch(t *testing.T) {
	replies := map[int]string{
		0: "3|the " + Opens + "3" + Closes + " harbour lights",
		1: "3|we spoke of the harbor lite\n4|until the fogg came in",
	}

	put, _ := gatherReplies([]Batch{wide(), narrow()}, replies, MaxEditDistance)
	if put[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the batch whose reply was an answer", put[3].Text)
	}
	if put[4].Text != "until the fogg came in" {
		t.Errorf("line 4 says %q, want the batch whose reply was an answer", put[4].Text)
	}
}

func TestALineNoAnsweredBatchCoversIsLeftAsItWasHeard(t *testing.T) {
	replies := map[int]string{0: "3|we spoke of the harbour lights"}

	put, _ := gatherReplies([]Batch{wide(), narrow()}, replies, MaxEditDistance)
	for _, at := range []int{1, 2, 4, 5} {
		if text, named := put[at]; named {
			t.Errorf("line %d was put right to %q with nothing answered about it", at, text.Text)
		}
	}
}

func TestACorrectionThatMovedTooFarIsInNoBatchesGathering(t *testing.T) {
	replies := map[int]string{1: "3|we spoke of the harbour lights\n4|fog"}

	put, _ := gatherReplies([]Batch{narrow()}, replies, MaxEditDistance)
	if _, named := put[4]; named {
		t.Errorf("line 4 was put right to %q, whose letters moved too far", put[4].Text)
	}
	if put[3].Text != "we spoke of the harbour lights" {
		t.Errorf("line 3 says %q, want the correction that stayed put", put[3].Text)
	}
}

func TestABatchNothingCameBackAboutPutsNothingRight(t *testing.T) {
	if put, _ := gatherReplies([]Batch{wide()}, nil, MaxEditDistance); len(put) != 0 {
		t.Errorf("a batch with no reply put %v right", put)
	}
}

// Gathered names the batches whose reply ran on past the last line they carry,
// in the order they were asked. A batch it refuses names none.
func TestGatheredNamesTheBatchesARunRanPast(t *testing.T) {
	first, second := makeBrokenBatch(), makeBrokenBatch()
	first.Number, second.Number = 1, 2
	replies := map[int]string{
		1: "5-9 | whatever it says",
		2: "4-5 | Krishna is Radha, Krishna is connected with Radhika.",
	}

	put, past := gatherReplies([]Batch{first, second}, replies, 0)
	if len(put) != 1 {
		t.Fatalf("%v put right", put)
	}
	if !reflect.DeepEqual(past, []int{1}) {
		t.Errorf("the batches a run went past are %v, want the first alone", past)
	}
}

func TestAReachingRunNamesTheLineItBeginsOn(t *testing.T) {
	batch := makeBrokenBatch()
	got := readAnswer(batch, "4 | Krishna is Radha.\n5-9 | whatever it says", 0)
	if !got.isAnswer || !reflect.DeepEqual(got.reaching, []int{5}) {
		t.Errorf("answer %+v, want a run reaching past the batch from line 5", got)
	}
	if len(got.lines) != 1 || got.lines[0].Number != 4 {
		t.Errorf("lines %v, want line 4 alone", got.lines)
	}
}

func TestARefusedReplyIsNoAnswer(t *testing.T) {
	if got := readAnswer(wide(), "999|invented", 0); got.isAnswer || got.lines != nil {
		t.Errorf("answer %+v", got)
	}
}
