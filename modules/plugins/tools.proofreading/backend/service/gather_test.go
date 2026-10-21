package service

import (
	"reflect"
	"testing"
)

// gatherReplies is what the replies put right, as gatherAnswers gathers them.
func gatherReplies(asked []Batch, replies map[int]string, maxDistance float64) (map[int]Line, []int) {
	var answers []answer
	for _, batch := range asked {
		reply, replyVal := replies[batch.Number]
		if !replyVal {
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

	for _, request := range [][]Batch{
		{wide(), narrow()},
		{narrow(), wide()},
	} {
		placed, _ := gatherReplies(request, replies, MaxEditDistance)
		if len(placed) != 1 {
			t.Fatalf("%v put right, want line 3 alone", placed)
		}
		if placed[3].Text != "we spoke of the harbour lights" {
			t.Errorf("line 3 says %q, want the batch that saw the four lines after it", placed[3].Text)
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

	placed, _ := gatherReplies([]Batch{early, late}, replies, MaxEditDistance)
	if placed[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the later batch", placed[3].Text)
	}
}

func TestARefusedReplyLeavesItsLinesToTheOtherBatch(t *testing.T) {
	replies := map[int]string{
		0: "3|the " + Opens + "3" + Closes + " harbour lights",
		1: "3|we spoke of the harbor lite\n4|until the fogg came in",
	}

	placed, _ := gatherReplies([]Batch{wide(), narrow()}, replies, MaxEditDistance)
	if placed[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the batch whose reply was an answer", placed[3].Text)
	}
	if placed[4].Text != "until the fogg came in" {
		t.Errorf("line 4 says %q, want the batch whose reply was an answer", placed[4].Text)
	}
}

func TestALineNoAnsweredBatchCoversIsLeftAsItWasHeard(t *testing.T) {
	replies := map[int]string{0: "3|we spoke of the harbour lights"}

	placed, _ := gatherReplies([]Batch{wide(), narrow()}, replies, MaxEditDistance)
	for _, at := range []int{1, 2, 4, 5} {
		if text, named := placed[at]; named {
			t.Errorf("line %d was put right to %q with nothing answered about it", at, text.Text)
		}
	}
}

func TestACorrectionThatMovedTooFarIsInNoBatchesGathering(t *testing.T) {
	replies := map[int]string{1: "3|we spoke of the harbour lights\n4|fog"}

	placed, _ := gatherReplies([]Batch{narrow()}, replies, MaxEditDistance)
	if _, named := placed[4]; named {
		t.Errorf("line 4 was put right to %q, whose letters moved too far", placed[4].Text)
	}
	if placed[3].Text != "we spoke of the harbour lights" {
		t.Errorf("line 3 says %q, want the correction that stayed put", placed[3].Text)
	}
}

func TestABatchNothingCameBackAboutPutsNothingRight(t *testing.T) {
	if placed, _ := gatherReplies([]Batch{wide()}, nil, MaxEditDistance); len(placed) != 0 {
		t.Errorf("a batch with no reply put %v right", placed)
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

	placed, past := gatherReplies([]Batch{first, second}, replies, 0)
	if len(placed) != 1 {
		t.Fatalf("%v put right", placed)
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

// joinable is a batch of the speech lines from first to last that may be
// answered in runs.
func joinable(number, first, last int) Batch {
	batch := Batch{Number: number, IsJoinable: true}
	for at := first; at <= last; at++ {
		batch.Lines = append(batch.Lines, makeLine(at))
	}
	return batch
}

// coveredOnce fails for a line two corrections cover.
func coveredOnce(t *testing.T, put map[int]Line) {
	t.Helper()
	observed := map[int]int{}
	for first, line := range put {
		for at := first; at <= max(line.Last, line.Number); at++ {
			if other, twice := observed[at]; twice {
				t.Errorf("line %d is covered by the corrections of %d and %d", at, other, first)
			}
			observed[at] = first
		}
	}
}

func TestARunSeenFromFurtherBackTakesTheLineOfASingleCorrection(t *testing.T) {
	replies := map[int]string{
		0: "4|until the fog came in.",
		1: "3-4|We spoke of the harbour lights until the fog came in.",
	}
	single, run := joinable(0, 3, 5), joinable(1, 2, 5)

	placed, _ := gatherReplies([]Batch{single, run}, replies, 0)
	coveredOnce(t, placed)
	if len(placed) != 1 || placed[3].Last != 4 {
		t.Errorf("gathered %v, want the run of lines 3 to 4 alone", placed)
	}

	placed, _ = gatherReplies([]Batch{run, single}, replies, 0)
	coveredOnce(t, placed)
}

func TestTwoRunsSharingALineAreNotBothKept(t *testing.T) {
	replies := map[int]string{
		0: "2-3|And the sea was calm, we spoke of the harbour lights.",
		1: "3-4|We spoke of the harbour lights until the fog came in.",
	}
	earlier, later := joinable(0, 1, 4), joinable(1, 3, 5)

	for _, request := range [][]Batch{{earlier, later}, {later, earlier}} {
		placed, _ := gatherReplies(request, replies, 0)
		coveredOnce(t, placed)
		if len(placed) != 1 {
			t.Errorf("gathered %v, want one run", placed)
		}
	}
}
