package source

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// newJoinReply is a reply putting a run of lines together as one. A run stands
// within one batch, so a transcript put right this way is asked in whole
// batches.
func newJoinReply(at, through int, said string) string {
	return fmt.Sprintf("%d-%d|%s", at, through, said)
}

// newWholeRun is a run over a transcript asked about in one batch.
func newWholeRun(t *testing.T, says map[int]string, words ...string) (ProofreadTranscript, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := newProofreadTranscript(t, says, words...)
	u.BatchSize = len(words)
	return u, v, kept, hash
}

// A sentence the recording broke across two spans comes back as one line,
// running from the first moment to the last.
func TestASentenceBrokenAcrossSpansBecomesOneLine(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure."}
	u, v, shelved, hash := newWholeRun(t,
		map[int]string{0: newJoinReply(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 2 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[0].Text != "Krishna is Radha, Krishna is connected with Radhika." {
		t.Errorf("the first line says %q", cues[0].Text)
	}
	// The sentence runs from where the first span began to where the last
	// one ended.
	if cues[0].From != getSpan(0).From || cues[0].To != getSpan(1).To {
		t.Errorf("the sentence runs %d-%d", cues[0].From, cues[0].To)
	}
	if cues[1].Text != "Sure." || cues[1].From != getSpan(2).From {
		t.Errorf("the line after it says %q at %d", cues[1].Text, cues[1].From)
	}
}

// What was heard is not written over by putting its lines together.
func TestJoiningLeavesWhatWasHeardWhereItIs(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila."}
	u, v, shelved, hash := newWholeRun(t,
		map[int]string{0: newJoinReply(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	heard := readCues(t, shelved, text.Artifact(text.ASR, hash))
	if len(heard) != 2 || heard[0].Text != words[0] || heard[1].Text != words[1] {
		t.Errorf("what was heard now says %+v", heard)
	}
}

// newOverlappingRun is a run over four spans cut into two batches sharing three
// lines, with as many batches to a request as asked for.
func newOverlappingRun(t *testing.T, batches int, says map[int]string) (ProofreadTranscript, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := newProofreadTranscript(t,
		says,
		"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all.")
	u.BatchSize, u.Overlap, u.InFlight = 3, 2, batches
	return u, v, kept, hash
}

// The joined sentence, and the transcript it leaves behind it.
const (
	putTogether = "Krishna is Radha, Krishna is connected with Radhika."
	onItsOwn    = "connected with Radhika."
)

// A batch answering for a line another batch put into a run is answered too
// late: the words no longer stand on their own, and the answer is dropped.
//
// It holds whether the two batches are asked about in one request or in two,
// and it does not turn on the order the corrections come back in.
func TestALineAlreadyPutIntoARunIsLeftInIt(t *testing.T) {
	for _, batches := range []int{1, 2} {
		t.Run(fmt.Sprintf("%d to a request", batches), func(t *testing.T) {
			u, v, shelved, hash := newOverlappingRun(t, batches, map[int]string{
				0: newJoinReply(0, 1, putTogether),
				1: corrects(1, onItsOwn),
			})

			if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
				t.Fatal(err)
			}

			cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
			if len(cues) != 3 {
				t.Fatalf("the transcript says %+v", cues)
			}
			if cues[0].Text != putTogether {
				t.Errorf("the first line says %q", cues[0].Text)
			}
			if cues[1].Text != "Sure." {
				t.Errorf("the line after the run says %q", cues[1].Text)
			}
		})
	}
}

// Two batches answering for runs over the same lines: the run standing first
// puts them together, and the one reaching into it is dropped.
func TestARunOverLinesAlreadyPutTogetherIsDropped(t *testing.T) {
	u, v, shelved, hash := newOverlappingRun(t, 1, map[int]string{
		0: newJoinReply(0, 1, putTogether),
		1: newJoinReply(1, 2, "Connected with Radhika. Sure."),
	})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 3 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[0].Text != putTogether || cues[1].Text != "Sure." {
		t.Errorf("the transcript says %+v", cues)
	}
	// The run that stands reaches to the end of the second span, and the one
	// dropped moved no moment.
	if cues[0].To != getSpan(1).To || cues[1].To != getSpan(2).To {
		t.Errorf("the lines run to %d and %d", cues[0].To, cues[1].To)
	}
}

// A transcript whose lines were put together is taken up again at the moment
// the count stands at, which putting them together did not move. The numbers
// the lines are known by are not the same ones the run before it saw.
func TestARunTakesUpATranscriptWhoseLinesWerePutTogether(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all."}
	u, v, shelved, by, hash := newProofreadTranscript(t, map[int]string{0: newJoinReply(0, 1, putTogether)}, words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 0, 1

	ctx, stop := context.WithCancel(t.Context())
	by.stop = func(requests int) {
		if requests == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	again := &corrector{says: map[int]string{}}
	u.By = again
	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	// Three lines stand where four were heard, and the run before this one had
	// asked about the first of them.
	if res.Lines != 3 || res.Resumed != 1 || res.Read != 3 {
		t.Errorf("got %+v", res)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 3 || cues[0].Text != putTogether {
		t.Errorf("the transcript says %+v", cues)
	}
}

// The spans of speech one recording was heard as. The third, fourth and
// fifth of them are one sentence.
var speech = []string{
	"Welcome, everyone.",
	"Today we will read",
	"a verse that the teacher",
	"explained at some length",
	"in the morning class.",
	"The point of it",
	"is very simple.",
	"Let us begin.",
}

// What those three spans say, as one line.
const crossed = "A verse that the teacher explained at some length in the morning class."

// newCrossingRun is a run over those spans, cut into batches of four sharing
// one line, one batch to a request. The batches are numbered 0 to 2 and the
// seams over the two cuts between them 3 and 4.
func newCrossingRun(t *testing.T, says map[int]string) (ProofreadTranscript, domain.Vault, *shelf, *corrector, string) {
	t.Helper()
	u, v, kept, by, hash := newProofreadTranscript(t, says, speech...)
	u.BatchSize, u.Overlap, u.InFlight = 4, 1, 1
	return u, v, kept, by, hash
}

// A sentence beginning further before a cut than the shared lines reach, and
// ending after it, is in no batch of the first pass. The seam over that cut
// holds it whole, and it is put back together there.
func TestASentenceCrossingACutIsPutBackTogether(t *testing.T) {
	u, v, shelved, by, hash := newCrossingRun(t, map[int]string{
		0: newJoinReply(2, 4, crossed),
		3: newJoinReply(2, 4, crossed),
	})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	// The batch the sentence begins in reaches only to the cut, and the run
	// naming the whole sentence there is dropped.
	if len(by.asked) < 4 {
		t.Fatalf("it asked %v", by.asked)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 6 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[2].Text != crossed {
		t.Errorf("the sentence says %q", cues[2].Text)
	}
	if cues[2].From != getSpan(2).From || cues[2].To != getSpan(4).To {
		t.Errorf("the sentence runs %d-%d", cues[2].From, cues[2].To)
	}
	if cues[3].Text != speech[5] {
		t.Errorf("the line after it says %q", cues[3].Text)
	}
}

// Where no reply ran on past the end of its batch, no sentence crossed a cut,
// and the transcript costs what its own batches cost.
func TestATranscriptNothingRanPastAsksAboutItsBatchesOnly(t *testing.T) {
	u, v, _, by, _ := newCrossingRun(t, map[int]string{
		0: corrects(0, "Welcome, everybody."),
		1: newJoinReply(3, 4, "Today we will read a verse that the teacher"),
	})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(by.asked, [][]int{{0}, {1}, {2}}) {
		t.Errorf("it asked %v, want the three batches of the transcript", by.asked)
	}

	// The transcript is answered whole, seams and all, and a run over it again
	// asks nothing.
	again := &corrector{}
	u.By = again
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if len(again.asked) != 0 {
		t.Errorf("it asked %v again", again.asked)
	}
}

// One batch answered for a sentence running on past its end. The seam over that
// cut is asked about, and the other cut costs nothing.
func TestOnlyTheCutASentenceRanPastIsAskedAbout(t *testing.T) {
	u, v, _, by, _ := newCrossingRun(t, map[int]string{0: newJoinReply(2, 4, crossed)})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(by.asked, [][]int{{0}, {1}, {2}, {3}}) {
		t.Errorf("it asked %v, want the three batches and the seam over the first cut", by.asked)
	}
}

// A transcript of one batch has no cut, and nothing is asked about twice.
func TestATranscriptOfOneBatchAsksNothingMore(t *testing.T) {
	u, v, _, by, _ := newCrossingRun(t, nil)
	u.BatchSize = len(speech)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(by.asked, [][]int{{0}}) {
		t.Errorf("it asked %v, want the one batch", by.asked)
	}
}

// A run stopped in the seam pass takes up at the seam it stopped on. What the
// pass before it asked about is not asked about again.
func TestARunStoppedInTheSeamPassTakesUpWhereItStopped(t *testing.T) {
	u, v, shelved, by, hash := newCrossingRun(t, map[int]string{
		0: newJoinReply(2, 4, crossed),
		1: newJoinReply(5, 7, "The point of it is very simple. Let us begin."),
		2: corrects(7, "Let us begin!"),
	})

	ctx, stop := context.WithCancel(t.Context())
	by.stop = func(requests int) {
		if requests == 5 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	again := &corrector{says: map[int]string{4: newJoinReply(5, 6, "The point of it is very simple.")}}
	u.By = again
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again.asked, [][]int{{4}}) {
		t.Errorf("it asked %v, want the seam it stopped on", again.asked)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 7 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[6].Text != "Let us begin!" {
		t.Errorf("what the pass before put right says %q", cues[6].Text)
	}
	if cues[5].Text != "The point of it is very simple." || cues[5].To != getSpan(6).To {
		t.Errorf("the sentence says %q and runs to %d", cues[5].Text, cues[5].To)
	}
}

// A line the first pass put into a run stands in it. A seam answering for that
// line again is answered too late.
func TestALineTheFirstPassJoinedIsNotJoinedAgain(t *testing.T) {
	u, v, shelved, _, hash := newCrossingRun(t, map[int]string{
		0: newJoinReply(1, 2, "Today we will read a verse that the teacher") + "\n" +
			newJoinReply(3, 5, "Explained at some length in the morning class."),
		3: newJoinReply(2, 3, "A verse that the teacher explained at some length"),
	})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := readCues(t, shelved, text.Corrections(text.ASR, hash))
	if len(cues) != 7 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[1].Text != "Today we will read a verse that the teacher" || cues[1].To != getSpan(2).To {
		t.Errorf("the run says %q and runs to %d", cues[1].Text, cues[1].To)
	}
	if cues[2].Text != speech[3] {
		t.Errorf("the line after the run says %q", cues[2].Text)
	}
}

// A seam carries what the recording holds, as a batch of the first pass does.
func TestASeamCarriesWhatTheRecordingHolds(t *testing.T) {
	u, v, _, by, _ := newCrossingRun(t, map[int]string{0: newJoinReply(2, 4, crossed)})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if len(by.about) < 4 {
		t.Fatalf("it was asked about %d batches", len(by.about))
	}
	for at, about := range by.about {
		if !strings.Contains(about, "The speech opens: Welcome, everyone. Today we will read") {
			t.Errorf("batch %d says the recording holds %q", at, about)
		}
	}
}
