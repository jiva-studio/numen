package service

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestSpeechIsCutIntoBatchesOfSize(t *testing.T) {
	cues := makeLines("one", "two", "three", "four", "five")

	batches := speechBatches(cues, 2, 0)
	if len(batches) != 3 {
		t.Fatalf("%d batches, want five cues two at a time", len(batches))
	}
	for i, want := range []int{2, 2, 1} {
		if len(batches[i].Lines) != want {
			t.Errorf("batch %d holds %d lines, want %d", i, len(batches[i].Lines), want)
		}
		if batches[i].Number != i {
			t.Errorf("batch %d is numbered %d, want its place in the run", i, batches[i].Number)
		}
	}
	if batches[2].Lines[0].Number != 4 || batches[2].Lines[0].Text != "five" {
		t.Errorf("last line is %v, want the fifth cue", batches[2].Lines[0])
	}
}

// numbers is the lines a run of batches carries, batch by batch.
func numbers(batches []Batch) [][]int {
	out := make([][]int, 0, len(batches))
	for _, batch := range batches {
		var carried []int
		for _, line := range batch.Lines {
			carried = append(carried, line.Number)
		}
		out = append(out, carried)
	}
	return out
}

// makeTranscript is a transcript of count cues, each saying something.
func makeTranscript(count int) []Line {
	var cues []Line
	for i := 0; i < count; i++ {
		cues = append(cues, Line{Number: i, Text: strconv.Itoa(i)})
	}
	return cues
}

func TestSharingNothingCutsSpeechAsItAlwaysWas(t *testing.T) {
	cues := makeTranscript(7)

	for _, overlap := range []int{0, -1, -8} {
		for size := 1; size <= 4; size++ {
			var want [][]int
			for at := 0; at < len(cues); at++ {
				if at%size == 0 {
					want = append(want, nil)
				}
				want[len(want)-1] = append(want[len(want)-1], at)
			}
			got := numbers(speechBatches(cues, size, overlap))
			if !reflect.DeepEqual(got, want) {
				t.Errorf("size %d sharing %d gave %v, want %v", size, overlap, got, want)
			}
		}
	}
}

func TestABatchOpensOnTheLastLinesOfTheOneBefore(t *testing.T) {
	got := numbers(speechBatches(makeTranscript(7), 4, 2))
	want := [][]int{{0, 1, 2, 3}, {2, 3, 4, 5}, {4, 5, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("batches %v, want %v", got, want)
	}
}

func TestSharingAsMuchAsABatchHoldsStillWalksTheTranscript(t *testing.T) {
	for _, overlap := range []int{3, 4, 9} {
		got := numbers(speechBatches(makeTranscript(5), 3, overlap))
		want := [][]int{{0, 1, 2}, {1, 2, 3}, {2, 3, 4}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sharing %d gave %v, want %v", overlap, got, want)
		}
	}
}

func TestEveryLineIsInABatchHoweverSpeechIsCut(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 5, 8, 13} {
		for size := 1; size <= 6; size++ {
			for overlap := -2; overlap <= 8; overlap++ {
				cues := makeTranscript(count)
				batches := speechBatches(cues, size, overlap)

				seen := make(map[int]bool, count)
				for i, batch := range batches {
					if len(batch.Lines) == 0 {
						t.Fatalf("%d cues by %d sharing %d: batch %d holds nothing", count, size, overlap, i)
					}
					if len(batch.Lines) > size {
						t.Errorf("%d cues by %d sharing %d: batch %d holds %d lines", count, size, overlap, i, len(batch.Lines))
					}
					if batch.Number != i {
						t.Errorf("%d cues by %d sharing %d: batch %d is numbered %d", count, size, overlap, i, batch.Number)
					}
					for _, line := range batch.Lines {
						seen[line.Number] = true
					}
				}
				for at := range cues {
					if !seen[at] {
						t.Errorf("%d cues by %d sharing %d: cue %d is in no batch", count, size, overlap, at)
					}
				}

				// Each batch reaches further than the one before it, so the walk
				// through the transcript never stalls.
				for i := 1; i < len(batches); i++ {
					last, before := batches[i].Lines, batches[i-1].Lines
					if last[len(last)-1].Number <= before[len(before)-1].Number {
						t.Errorf("%d cues by %d sharing %d: batch %d reaches no further than %d",
							count, size, overlap, i, i-1)
					}
					if last[0].Number <= before[0].Number {
						t.Errorf("%d cues by %d sharing %d: batch %d opens no later than %d",
							count, size, overlap, i, i-1)
					}
				}
			}
		}
	}
}

func TestOneLineOfSpeechIsOneBatch(t *testing.T) {
	for _, overlap := range []int{0, 1, 5} {
		got := numbers(speechBatches(makeTranscript(1), 3, overlap))
		if !reflect.DeepEqual(got, [][]int{{0}}) {
			t.Errorf("sharing %d gave %v, want the one line", overlap, got)
		}
	}
}

func TestSpeechOfNoLinesIsNoBatches(t *testing.T) {
	for _, overlap := range []int{0, 2, 7} {
		if batches := speechBatches(nil, 3, overlap); batches != nil {
			t.Errorf("sharing %d gave back %v", overlap, batches)
		}
		if batches := speechBatches(makeLines(""), 3, overlap); batches != nil {
			t.Errorf("sharing %d over silence gave back %v", overlap, batches)
		}
	}
}

func TestACueSayingNothingCarriesNoLine(t *testing.T) {
	batches := speechBatches(makeLines("one", "", "two"), 8, 0)
	if len(batches) != 1 {
		t.Fatalf("%d batches, want one", len(batches))
	}
	if len(batches[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(batches[0].Lines))
	}
	if batches[0].Lines[1].Number != 2 {
		t.Errorf("line numbered %d, want its place in the transcript 2", batches[0].Lines[1].Number)
	}
}

func TestABatchOfNoLinesHoldsNothing(t *testing.T) {
	if batches := speechBatches(makeLines("one"), 0, 0); batches != nil {
		t.Errorf("a size of nothing gave back %v", batches)
	}
}

// The two instructions ask for different corrections: one is about what a
// machine read, the other about what it heard.
func TestSpeechIsProofreadForDifferentThingsThanAScan(t *testing.T) {
	if scanInstruction == speechInstruction {
		t.Fatal("a scan and speech are proofread with the same instruction")
	}
	for _, word := range []string{"read off the pages", "diacritics"} {
		if !strings.Contains(scanInstruction, word) {
			t.Errorf("the scan instruction does not mention %q", word)
		}
		if strings.Contains(speechInstruction, word) {
			t.Errorf("the speech instruction mentions %q", word)
		}
	}
	for _, word := range []string{"heard in a recording", "sounds like it", "capitalisation", "hesitation", "summarise"} {
		if !strings.Contains(speechInstruction, word) {
			t.Errorf("the speech instruction does not mention %q", word)
		}
	}
	// Both are answered the same way.
	for _, one := range []string{scanInstruction, speechInstruction} {
		for _, word := range []string{
			"12|the line, put right",
			"A line you would leave alone is a line you do not answer with.",
			Opens,
			Closes,
		} {
			if !strings.Contains(one, word) {
				t.Errorf("an instruction does not say %q", word)
			}
		}
	}

	// A page's printed lines stay where they were printed. A sentence broken
	// across lines of speech is answered for as one line.
	if !strings.Contains(scanInstruction, "Nothing moves from one line to\n  another") {
		t.Error("the scan instruction lets a word move between lines")
	}
	if !strings.Contains(speechInstruction, "12-14|the whole sentence, put right") {
		t.Error("the speech instruction does not say how a broken sentence is answered for")
	}
}

// cuts is the cut after each of a run of batches but the last.
func cuts(batches int) []int {
	out := make([]int, max(batches-1, 0))
	for at := range out {
		out[at] = at
	}
	return out
}

// A batch over a cut batch holds the size lines around a cut, half of them before it, and
// is numbered on from the batches the transcript was cut into.
func TestASeamHoldsTheLinesOnBothSidesOfACut(t *testing.T) {
	cues := makeTranscript(8)
	batches := speechBatches(cues, 4, 1)
	if got := numbers(batches); !reflect.DeepEqual(got, [][]int{{0, 1, 2, 3}, {3, 4, 5, 6}, {6, 7}}) {
		t.Fatalf("batches %v", got)
	}

	batchesAcrossCuts := cutBatches(cues, 4, 1, cuts(len(batches)))
	if got := numbers(batchesAcrossCuts); !reflect.DeepEqual(got, [][]int{{1, 2, 3, 4}, {4, 5, 6, 7}}) {
		t.Errorf("batchesAcrossCuts %v, want a window over each of the two cuts", got)
	}
	for i, atCut := range batchesAcrossCuts {
		if atCut.Number != len(batches)+i {
			t.Errorf("batch over a cut %d is numbered %d, want its place after the batches", i, atCut.Number)
		}
		if !atCut.IsJoinable {
			t.Errorf("batch over a cut %d does not put lines together", i)
		}
	}
}

// A batch over a cut is built for the cuts it is asked for and no others. The cut after
// the last batch is no cut at all.
func TestOnlyTheCutsAskedForAreBuilt(t *testing.T) {
	cues := makeTranscript(8)
	if got := numbers(cutBatches(cues, 4, 1, []int{1})); !reflect.DeepEqual(got, [][]int{{4, 5, 6, 7}}) {
		t.Errorf("batchesAcrossCuts %v, want the window over the second cut", got)
	}
	if batchesAcrossCuts := cutBatches(cues, 4, 1, nil); batchesAcrossCuts != nil {
		t.Errorf("no cuts gave back %v", batchesAcrossCuts)
	}
	if batchesAcrossCuts := cutBatches(cues, 4, 1, []int{2, 9, -1}); batchesAcrossCuts != nil {
		t.Errorf("cuts after the last batch gave back %v", batchesAcrossCuts)
	}
}

func TestATranscriptOfOneBatchHasNoSeams(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 4} {
		if batchesAcrossCuts := cutBatches(makeTranscript(count), 4, 1, cuts(4)); batchesAcrossCuts != nil {
			t.Errorf("%d lines gave back %v", count, batchesAcrossCuts)
		}
	}
}

// A batch of one line has no room for a line on either side of a cut.
func TestABatchOfOneLineHasNoSeams(t *testing.T) {
	if batchesAcrossCuts := cutBatches(makeTranscript(6), 1, 0, cuts(6)); batchesAcrossCuts != nil {
		t.Errorf("gave back %v", batchesAcrossCuts)
	}
}

// hasLines reports whether a batch holds every line of a run, first to last.
func hasLines(batch Batch, at, through int) bool {
	held := make(map[int]bool, len(batch.Lines))
	for _, line := range batch.Lines {
		held[line.Number] = true
	}
	for line := at; line <= through; line++ {
		if !held[line] {
			return false
		}
	}
	return true
}

// A run of lines is answered for inside one batch. A run of half a batch or
// fewer that no batch of the transcript holds whole is held whole by a batch over a cut.
func TestASentenceCrossingACutStandsWholeInASeam(t *testing.T) {
	for _, count := range []int{2, 3, 5, 8, 13, 21} {
		for size := 2; size <= 8; size++ {
			for overlap := 0; overlap < size; overlap++ {
				cues := makeTranscript(count)
				batches := speechBatches(cues, size, overlap)
				batchesAcrossCuts := cutBatches(cues, size, overlap, cuts(len(batches)))

				for at := 0; at < count; at++ {
					for through := at; through <= min(at+size/2-1, count-1); through++ {
						if slices.ContainsFunc(batches, func(b Batch) bool {
							return hasLines(b, at, through)
						}) {
							continue
						}
						if !slices.ContainsFunc(batchesAcrossCuts, func(b Batch) bool {
							return hasLines(b, at, through)
						}) {
							t.Errorf("%d lines by %d sharing %d: the run %d-%d is in no batch and no batch over a cut",
								count, size, overlap, at, through)
						}
					}
				}
			}
		}
	}
}

// Each batch over a cut reaches further into the transcript than the one before it, so a
// run taking up in the pass over the cuts never stalls and never skips one.
func TestEachSeamReachesFurtherThanTheOneBefore(t *testing.T) {
	for _, count := range []int{2, 3, 5, 8, 13, 21} {
		for size := 2; size <= 8; size++ {
			for overlap := 0; overlap < size; overlap++ {
				cues := makeTranscript(count)
				batchesAcrossCuts := cutBatches(cues, size, overlap, cuts(len(speechBatches(cues, size, overlap))))
				for i, atCut := range batchesAcrossCuts {
					if len(atCut.Lines) == 0 {
						t.Fatalf("%d lines by %d sharing %d: batch over a cut %d holds nothing", count, size, overlap, i)
					}
					if len(atCut.Lines) > size {
						t.Errorf("%d lines by %d sharing %d: batch over a cut %d holds %d lines",
							count, size, overlap, i, len(atCut.Lines))
					}
					if i > 0 && atCut.Last() <= batchesAcrossCuts[i-1].Last() {
						t.Errorf("%d lines by %d sharing %d: batch over a cut %d reaches no further than %d",
							count, size, overlap, i, i-1)
					}
				}
			}
		}
	}
}
