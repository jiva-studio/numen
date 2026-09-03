package proofread_test

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// box is one printed line on a page, over a run of the prose.
func box(page, start, length int) highlight.Box {
	return highlight.Box{Page: page, Run: highlight.Run{Start: start, Length: length}}
}

func TestALineIsKnownByItsPlaceInTheWholeReading(t *testing.T) {
	prose := "one two three four "
	boxes := []highlight.Box{
		box(4, 0, 4), box(4, 4, 4),
		box(5, 8, 6), box(5, 14, 5),
	}

	batches := proofread.Scanned(prose, boxes)
	if len(batches) != 2 {
		t.Fatalf("%d batches, want the two pages the boxes were read from", len(batches))
	}
	if batches[0].Number != 4 || batches[1].Number != 5 {
		t.Errorf("pages %d and %d", batches[0].Number, batches[1].Number)
	}

	var numbers []int
	var text []string
	for _, batch := range batches {
		for _, line := range batch.Lines {
			numbers = append(numbers, line.Number)
			text = append(text, line.Text)
		}
	}
	for i, want := range []int{0, 1, 2, 3} {
		if numbers[i] != want {
			t.Errorf("line %d is numbered %d, want its place in the reading %d", i, numbers[i], want)
		}
	}
	for i, want := range []string{"one ", "two ", "three ", "four "} {
		if text[i] != want {
			t.Errorf("line %d says %q, want %q", i, text[i], want)
		}
	}
}

func TestBoxesWrittenForOtherBytesGiveNothing(t *testing.T) {
	prose := "one two"
	boxes := []highlight.Box{box(1, 0, 4), box(1, 4, 90)}

	if batches := proofread.Scanned(prose, boxes); batches != nil {
		t.Errorf("a reading of other bytes came back as %v", batches)
	}
}

func TestABoxWithNoLengthCarriesNoLine(t *testing.T) {
	prose := "one two "
	boxes := []highlight.Box{box(1, 0, 4), box(1, 4, 0), box(1, 4, 4)}

	batches := proofread.Scanned(prose, boxes)
	if len(batches) != 1 {
		t.Fatalf("%d batches, want one", len(batches))
	}
	if len(batches[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(batches[0].Lines))
	}
	if batches[0].Lines[1].Number != 2 {
		t.Errorf("line numbered %d, want its place in the reading 2", batches[0].Lines[1].Number)
	}
}

// cue is one stretch of speech, at the place in the transcript it was heard.
func cue(at int, text string) transcript.Cue {
	return transcript.Cue{Text: text, Offset: at}
}

func TestSpeechIsCutIntoBatchesOfSize(t *testing.T) {
	cues := []transcript.Cue{
		cue(0, "one"), cue(4, "two"), cue(8, "three"),
		cue(14, "four"), cue(19, "five"),
	}

	batches := proofread.Spoken(cues, 2, 0)
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
func numbers(batches []proofread.Batch) [][]int {
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

// spoken is a transcript of count cues, each saying something.
func spoken(count int) []transcript.Cue {
	var cues []transcript.Cue
	for i := 0; i < count; i++ {
		cues = append(cues, cue(i*4, strconv.Itoa(i)))
	}
	return cues
}

func TestSharingNothingCutsSpeechAsItAlwaysWas(t *testing.T) {
	cues := spoken(7)

	for _, overlap := range []int{0, -1, -8} {
		for size := 1; size <= 4; size++ {
			var want [][]int
			for at := 0; at < len(cues); at++ {
				if at%size == 0 {
					want = append(want, nil)
				}
				want[len(want)-1] = append(want[len(want)-1], at)
			}
			got := numbers(proofread.Spoken(cues, size, overlap))
			if !reflect.DeepEqual(got, want) {
				t.Errorf("size %d sharing %d gave %v, want %v", size, overlap, got, want)
			}
		}
	}
}

func TestABatchOpensOnTheLastLinesOfTheOneBefore(t *testing.T) {
	got := numbers(proofread.Spoken(spoken(7), 4, 2))
	want := [][]int{{0, 1, 2, 3}, {2, 3, 4, 5}, {4, 5, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("batches %v, want %v", got, want)
	}
}

func TestSharingAsMuchAsABatchHoldsStillWalksTheTranscript(t *testing.T) {
	for _, overlap := range []int{3, 4, 9} {
		got := numbers(proofread.Spoken(spoken(5), 3, overlap))
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
				cues := spoken(count)
				batches := proofread.Spoken(cues, size, overlap)

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
		got := numbers(proofread.Spoken(spoken(1), 3, overlap))
		if !reflect.DeepEqual(got, [][]int{{0}}) {
			t.Errorf("sharing %d gave %v, want the one line", overlap, got)
		}
	}
}

func TestSpeechOfNoLinesIsNoBatches(t *testing.T) {
	for _, overlap := range []int{0, 2, 7} {
		if batches := proofread.Spoken(nil, 3, overlap); batches != nil {
			t.Errorf("sharing %d gave back %v", overlap, batches)
		}
		if batches := proofread.Spoken([]transcript.Cue{cue(0, "")}, 3, overlap); batches != nil {
			t.Errorf("sharing %d over silence gave back %v", overlap, batches)
		}
	}
}

func TestACueSayingNothingCarriesNoLine(t *testing.T) {
	batches := proofread.Spoken([]transcript.Cue{cue(0, "one"), cue(4, ""), cue(4, "two")}, 8, 0)
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
	if batches := proofread.Spoken([]transcript.Cue{cue(0, "one")}, 0, 0); batches != nil {
		t.Errorf("a size of nothing gave back %v", batches)
	}
}

// The two instructions ask for different corrections: one is about what a
// machine read, the other about what it heard.
func TestSpeechIsProofreadForDifferentThingsThanAScan(t *testing.T) {
	if proofread.ScanInstruction == proofread.SpeechInstruction {
		t.Fatal("a scan and speech are proofread with the same instruction")
	}
	for _, word := range []string{"read off the pages", "diacritics"} {
		if !strings.Contains(proofread.ScanInstruction, word) {
			t.Errorf("the scan instruction does not mention %q", word)
		}
		if strings.Contains(proofread.SpeechInstruction, word) {
			t.Errorf("the speech instruction mentions %q", word)
		}
	}
	for _, word := range []string{"heard in a recording", "sounds like it", "capitalisation", "hesitation", "summarise"} {
		if !strings.Contains(proofread.SpeechInstruction, word) {
			t.Errorf("the speech instruction does not mention %q", word)
		}
	}
	// Both are answered the same way.
	for _, one := range []string{proofread.ScanInstruction, proofread.SpeechInstruction} {
		for _, word := range []string{
			"12|the line, put right",
			"A line you would leave alone is a line you do not answer with.",
			proofread.Opens,
			proofread.Closes,
		} {
			if !strings.Contains(one, word) {
				t.Errorf("an instruction does not say %q", word)
			}
		}
	}

	// A page's printed lines stay where they were printed. A sentence broken
	// across lines of speech is answered for as one line.
	if !strings.Contains(proofread.ScanInstruction, "Nothing moves from one line to\n  another") {
		t.Error("the scan instruction lets a word move between lines")
	}
	if !strings.Contains(proofread.SpeechInstruction, "12-14|the whole sentence, put right") {
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

// A seam batch holds the size lines around a cut, half of them before it, and
// is numbered on from the batches the transcript was cut into.
func TestASeamHoldsTheLinesOnBothSidesOfACut(t *testing.T) {
	cues := spoken(8)
	batches := proofread.Spoken(cues, 4, 1)
	if got := numbers(batches); !reflect.DeepEqual(got, [][]int{{0, 1, 2, 3}, {3, 4, 5, 6}, {6, 7}}) {
		t.Fatalf("batches %v", got)
	}

	seams := proofread.Seams(cues, 4, 1, cuts(len(batches)))
	if got := numbers(seams); !reflect.DeepEqual(got, [][]int{{1, 2, 3, 4}, {4, 5, 6, 7}}) {
		t.Errorf("seams %v, want a window over each of the two cuts", got)
	}
	for i, seam := range seams {
		if seam.Number != len(batches)+i {
			t.Errorf("seam %d is numbered %d, want its place after the batches", i, seam.Number)
		}
		if !seam.Joinable {
			t.Errorf("seam %d does not put lines together", i)
		}
	}
}

// A seam is built for the cuts it is asked for and no others. The cut after
// the last batch is no cut at all.
func TestOnlyTheCutsAskedForAreBuilt(t *testing.T) {
	cues := spoken(8)
	if got := numbers(proofread.Seams(cues, 4, 1, []int{1})); !reflect.DeepEqual(got, [][]int{{4, 5, 6, 7}}) {
		t.Errorf("seams %v, want the window over the second cut", got)
	}
	if seams := proofread.Seams(cues, 4, 1, nil); seams != nil {
		t.Errorf("no cuts gave back %v", seams)
	}
	if seams := proofread.Seams(cues, 4, 1, []int{2, 9, -1}); seams != nil {
		t.Errorf("cuts after the last batch gave back %v", seams)
	}
}

func TestATranscriptOfOneBatchHasNoSeams(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 4} {
		if seams := proofread.Seams(spoken(count), 4, 1, cuts(4)); seams != nil {
			t.Errorf("%d lines gave back %v", count, seams)
		}
	}
}

// A batch of one line has no room for a line on either side of a cut.
func TestABatchOfOneLineHasNoSeams(t *testing.T) {
	if seams := proofread.Seams(spoken(6), 1, 0, cuts(6)); seams != nil {
		t.Errorf("gave back %v", seams)
	}
}

// carries is whether a batch holds every line of a run, first to last.
func carries(batch proofread.Batch, at, through int) bool {
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
// fewer that no batch of the transcript holds whole is held whole by a seam.
func TestASentenceCrossingACutStandsWholeInASeam(t *testing.T) {
	for _, count := range []int{2, 3, 5, 8, 13, 21} {
		for size := 2; size <= 8; size++ {
			for overlap := 0; overlap < size; overlap++ {
				cues := spoken(count)
				batches := proofread.Spoken(cues, size, overlap)
				seams := proofread.Seams(cues, size, overlap, cuts(len(batches)))

				for at := 0; at < count; at++ {
					for through := at; through <= min(at+size/2-1, count-1); through++ {
						if slices.ContainsFunc(batches, func(b proofread.Batch) bool {
							return carries(b, at, through)
						}) {
							continue
						}
						if !slices.ContainsFunc(seams, func(b proofread.Batch) bool {
							return carries(b, at, through)
						}) {
							t.Errorf("%d lines by %d sharing %d: the run %d-%d is in no batch and no seam",
								count, size, overlap, at, through)
						}
					}
				}
			}
		}
	}
}

// Each seam reaches further into the transcript than the one before it, so a
// run taking up in the seam pass never stalls and never skips one.
func TestEachSeamReachesFurtherThanTheOneBefore(t *testing.T) {
	for _, count := range []int{2, 3, 5, 8, 13, 21} {
		for size := 2; size <= 8; size++ {
			for overlap := 0; overlap < size; overlap++ {
				cues := spoken(count)
				seams := proofread.Seams(cues, size, overlap, cuts(len(proofread.Spoken(cues, size, overlap))))
				for i, seam := range seams {
					if len(seam.Lines) == 0 {
						t.Fatalf("%d lines by %d sharing %d: seam %d holds nothing", count, size, overlap, i)
					}
					if len(seam.Lines) > size {
						t.Errorf("%d lines by %d sharing %d: seam %d holds %d lines",
							count, size, overlap, i, len(seam.Lines))
					}
					if i > 0 && seam.Last() <= seams[i-1].Last() {
						t.Errorf("%d lines by %d sharing %d: seam %d reaches no further than %d",
							count, size, overlap, i, i-1)
					}
				}
			}
		}
	}
}
