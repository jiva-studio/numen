package numbered

import (
	"reflect"
	"testing"
)

func gatherReplies(requested []Batch, replies map[int]string, maxDistance float64) (map[int]Line, []int) {
	var answers []Answer
	for _, batch := range requested {
		reply, replyVal := replies[batch.Number]
		if !replyVal {
			continue
		}
		answers = append(answers, ReadAnswer(batch, reply, maxDistance))
	}
	return GatherAnswers(answers)
}

func makeLine(pos int) Line {
	return Line{Number: pos, Text: []string{
		"",
		"the ferry left at noone",
		"and the sea was calm",
		"we spoke of the harbor lites",
		"until the fog came in",
		"and the bell began to ring",
	}[pos]}
}

func wide() Batch {
	return Batch{Number: 0, Lines: []Line{
		makeLine(1), makeLine(2), makeLine(3), makeLine(4), makeLine(5),
	}}
}

func narrow() Batch {
	return Batch{Number: 1, Lines: []Line{makeLine(3), makeLine(4)}}
}

func TestACorrectionReachesFromTheBatchThatSawMoreOfWhatFollows(t *testing.T) {
	replies := map[int]string{
		0: "3|we spoke of the harbor lights",
		1: "3|we spoke of the harbor lite",
	}

	for _, request := range [][]Batch{
		{wide(), narrow()},
		{narrow(), wide()},
	} {
		placed, _ := gatherReplies(request, replies, maxEditDistance)
		if len(placed) != 1 {
			t.Fatalf("%v put right, want line 3 alone", placed)
		}
		if placed[3].Text != "we spoke of the harbor lights" {
			t.Errorf("line 3 says %q, want the batch that saw the four lines after it", placed[3].Text)
		}
	}
}

func TestBatchesSeeingAsMuchAsEachOtherAreSettledByTheLaterOne(t *testing.T) {
	early := Batch{Number: 0, Lines: []Line{makeLine(1), makeLine(2), makeLine(3)}}
	late := Batch{Number: 1, Lines: []Line{makeLine(2), makeLine(3)}}
	replies := map[int]string{
		0: "3|we spoke of the harbor lights",
		1: "3|we spoke of the harbor lite",
	}

	placed, _ := gatherReplies([]Batch{early, late}, replies, maxEditDistance)
	if placed[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the later batch", placed[3].Text)
	}
}

func TestARefusedReplyPreservesItsLinesToTheOtherBatch(t *testing.T) {
	replies := map[int]string{
		0: "3|the " + Opens + "3" + Closes + " harbor lights",
		1: "3|we spoke of the harbor lite\n4|until the fogg came in",
	}

	placed, _ := gatherReplies([]Batch{wide(), narrow()}, replies, maxEditDistance)
	if placed[3].Text != "we spoke of the harbor lite" {
		t.Errorf("line 3 says %q, want the batch whose reply was an answer", placed[3].Text)
	}
	if placed[4].Text != "until the fogg came in" {
		t.Errorf("line 4 says %q, want the batch whose reply was an answer", placed[4].Text)
	}
}

func TestGatherReplies_KeepsUncoveredLinesUnchanged(t *testing.T) {
	replies := map[int]string{0: "3|we spoke of the harbor lights"}

	placed, _ := gatherReplies([]Batch{wide(), narrow()}, replies, maxEditDistance)
	for _, item := range []int{1, 2, 4, 5} {
		if text, named := placed[item]; named {
			t.Errorf("line %d was put right to %q with nothing answered about it", item, text.Text)
		}
	}
}

func TestACorrectionThatMovedTooFarIsInNoBatchesGathering(t *testing.T) {
	replies := map[int]string{1: "3|we spoke of the harbor lights\n4|fog"}

	placed, _ := gatherReplies([]Batch{narrow()}, replies, maxEditDistance)
	if _, named := placed[4]; named {
		t.Errorf("line 4 was put right to %q, whose letters moved too far", placed[4].Text)
	}
	if placed[3].Text != "we spoke of the harbor lights" {
		t.Errorf("line 3 says %q, want the correction that stayed put", placed[3].Text)
	}
}

func TestABatchNothingReturnedAboutInsertsNothingRight(t *testing.T) {
	if placed, _ := gatherReplies([]Batch{wide()}, nil, maxEditDistance); len(placed) != 0 {
		t.Errorf("a batch with no reply put %v right", placed)
	}
}

// Gathered lists the batches whose reply ran past the last line they carry,
// in the order they were requested. A batch it rejects lists none.
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
	got := ReadAnswer(batch, "4-6 | Krishna is Radha.\n5-9 | whatever it says", 0)
	if !got.IsAnswer || !reflect.DeepEqual(got.Reaching, []int{5}) {
		t.Errorf("answer %+v, want a run reaching past the batch from line 5", got)
	}
	if len(got.Lines) != 1 || got.Lines[0].Number != 4 || got.Lines[0].Last != 6 {
		t.Errorf("lines %v, want run 4-6 alone", got.Lines)
	}
}

func TestARefusedReplyIsNoAnswer(t *testing.T) {
	if got := ReadAnswer(wide(), "999|invented", 0); got.IsAnswer || got.Lines != nil || got.Batch.Number != wide().Number {
		t.Errorf("answer %+v", got)
	}
	placed, past := GatherAnswers([]Answer{{
		Batch:    wide(),
		Lines:    []Line{{Number: 1, Text: "ignored"}},
		Reaching: []int{1},
		IsAnswer: false,
	}})
	if len(placed) != 0 || len(past) != 0 {
		t.Errorf("GatherAnswers kept non-answer: placed=%v past=%v", placed, past)
	}
}

func joinable(number, first, last int) Batch {
	batch := Batch{Number: number, IsJoinable: true}
	for pos := first; pos <= last; pos++ {
		batch.Lines = append(batch.Lines, makeLine(pos))
	}
	return batch
}

func coveredOnce(t *testing.T, lines map[int]Line) {
	t.Helper()
	observed := map[int]int{}
	for first, line := range lines {
		for pos := first; pos <= max(line.Last, line.Number); pos++ {
			if other, twice := observed[pos]; twice {
				t.Errorf("line %d is covered by the corrections of %d and %d", pos, other, first)
			}
			observed[pos] = first
		}
	}
}

func TestGatherReplies_LongerRunReplacesSingleCorrection(t *testing.T) {
	replies := map[int]string{
		0: "4|until the fog came in.",
		1: "3-4|We spoke of the harbor lights until the fog came in.",
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

func TestTwoRunsSharingALineAreNotBothCached(t *testing.T) {
	replies := map[int]string{
		0: "2-3|And the sea was calm, we spoke of the harbor lights.",
		1: "3-4|We spoke of the harbor lights until the fog came in.",
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

func TestEvictingRunReleasesCoveredTailLinesAndClearsOwner(t *testing.T) {
	answers := []Answer{
		{
			Batch:    joinable(0, 2, 3),
			Lines:    []Line{{Number: 2, Last: 3, Text: "two and three"}},
			IsAnswer: true,
		},
		{
			Batch:    joinable(1, 1, 3),
			Lines:    []Line{{Number: 1, Last: 2, Text: "one and two"}},
			IsAnswer: true,
		},
		{
			Batch:    joinable(2, 2, 5),
			Lines:    []Line{{Number: 2, Last: 2, Text: "two alone"}},
			IsAnswer: true,
		},
		{
			Batch:    joinable(3, 3, 3),
			Lines:    []Line{{Number: 3, Last: 3, Text: "three alone"}},
			IsAnswer: true,
		},
	}
	placed, _ := GatherAnswers(answers)
	if placed[2].Text != "two alone" || placed[3].Text != "three alone" {
		t.Errorf("placed = %+v, want lines 2 and 3 both kept", placed)
	}

	remaining := countRemainingLines([]Line{{Number: 10}, {Number: 20}})
	if remaining[10] != 1 || remaining[20] != 0 {
		t.Errorf("countRemainingLines = %v", remaining)
	}
	if rows := splitRows("  10|fixed  \n   \n"); !reflect.DeepEqual(rows, []string{"10|fixed"}) {
		t.Errorf("splitRows = %v", rows)
	}
	if (Line{Number: 5, Last: 0}).IsJoined() || (Line{Number: 5, Last: 5}).IsJoined() || !(Line{Number: 5, Last: 6}).IsJoined() {
		t.Error("Line.IsJoined failed")
	}
	if (Line{Number: 5, Last: 0}).GetLastNumber() != 5 || (Line{Number: 5, Last: 6}).GetLastNumber() != 6 {
		t.Error("Line.GetLastNumber failed")
	}
	if (Batch{}).Last() != -1 || (Batch{Lines: []Line{{Number: 7}}}).Last() != 7 {
		t.Error("Batch.Last failed")
	}
}
