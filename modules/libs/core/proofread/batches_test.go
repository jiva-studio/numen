package proofread_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// box is one printed line on a page, over a run of the prose.
func box(page, start, length int) lit.Box {
	return lit.Box{Page: page, Start: start, Length: length}
}

func TestALineIsKnownByItsPlaceInTheWholeReading(t *testing.T) {
	prose := "one two three four "
	boxes := []lit.Box{
		box(4, 0, 4), box(4, 4, 4),
		box(5, 8, 6), box(5, 14, 5),
	}

	batches := proofread.Scanned(prose, boxes)
	if len(batches) != 2 {
		t.Fatalf("%d batches, want the two pages the boxes were read from", len(batches))
	}
	if batches[0].At != 4 || batches[1].At != 5 {
		t.Errorf("pages %d and %d", batches[0].At, batches[1].At)
	}

	var numbers []int
	var text []string
	for _, batch := range batches {
		for _, line := range batch.Lines {
			numbers = append(numbers, line.At)
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
	boxes := []lit.Box{box(1, 0, 4), box(1, 4, 90)}

	if batches := proofread.Scanned(prose, boxes); batches != nil {
		t.Errorf("a reading of other bytes came back as %v", batches)
	}
}

func TestABoxWithNoLengthCarriesNoLine(t *testing.T) {
	prose := "one two "
	boxes := []lit.Box{box(1, 0, 4), box(1, 4, 0), box(1, 4, 4)}

	batches := proofread.Scanned(prose, boxes)
	if len(batches) != 1 {
		t.Fatalf("%d batches, want one", len(batches))
	}
	if len(batches[0].Lines) != 2 {
		t.Errorf("%d lines, want the two that say something", len(batches[0].Lines))
	}
	if batches[0].Lines[1].At != 2 {
		t.Errorf("line numbered %d, want its place in the reading 2", batches[0].Lines[1].At)
	}
}

// cue is one stretch of speech, at the place in the transcript it was heard.
func cue(at int, text string) transcript.Cue {
	return transcript.Cue{Text: text, At: at}
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
		if batches[i].At != i {
			t.Errorf("batch %d is numbered %d, want its place in the run", i, batches[i].At)
		}
	}
	if batches[2].Lines[0].At != 4 || batches[2].Lines[0].Text != "five" {
		t.Errorf("last line is %v, want the fifth cue", batches[2].Lines[0])
	}
}

// numbers is the lines a run of batches carries, batch by batch.
func numbers(batches []proofread.Batch) [][]int {
	out := make([][]int, 0, len(batches))
	for _, batch := range batches {
		var carried []int
		for _, line := range batch.Lines {
			carried = append(carried, line.At)
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
					if batch.At != i {
						t.Errorf("%d cues by %d sharing %d: batch %d is numbered %d", count, size, overlap, i, batch.At)
					}
					for _, line := range batch.Lines {
						seen[line.At] = true
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
					if last[len(last)-1].At <= before[len(before)-1].At {
						t.Errorf("%d cues by %d sharing %d: batch %d reaches no further than %d",
							count, size, overlap, i, i-1)
					}
					if last[0].At <= before[0].At {
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
	if batches[0].Lines[1].At != 2 {
		t.Errorf("line numbered %d, want its place in the transcript 2", batches[0].Lines[1].At)
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
	// Both are answered the same way, and neither moves a word.
	for _, one := range []string{proofread.ScanInstruction, proofread.SpeechInstruction} {
		for _, word := range []string{
			"12|the line, put right",
			"Nothing moves from one line to\n  another",
			"A line you would leave alone is a line you do not answer with.",
			proofread.Opens,
			proofread.Closes,
		} {
			if !strings.Contains(one, word) {
				t.Errorf("an instruction does not say %q", word)
			}
		}
	}
}
