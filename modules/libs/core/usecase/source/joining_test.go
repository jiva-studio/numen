package source

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// joins is a reply putting a run of lines together as one. A run stands within
// one batch, so a transcript put right this way is asked in whole batches.
func joins(at, through int, said string) string {
	return fmt.Sprintf("%d-%d|%s", at, through, said)
}

// wholly is a run over a transcript asked about in one batch.
func wholly(t *testing.T, says map[int]string, words ...string) (PutRight, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := hearing(t, says, words...)
	u.Lines = len(words)
	return u, v, kept, hash
}

// A sentence the recording broke across two stretches comes back as one line,
// running from the first moment to the last.
func TestASentenceBrokenAcrossStretchesBecomesOneLine(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure."}
	u, v, shelved, hash := wholly(t,
		map[int]string{0: joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := cued(t, shelved, text.Said(text.ASR, hash))
	if len(cues) != 2 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[0].Text != "Krishna is Radha, Krishna is connected with Radhika." {
		t.Errorf("the first line says %q", cues[0].Text)
	}
	// The sentence runs from where the first stretch began to where the last
	// one ended.
	if cues[0].From != stretch(0).From || cues[0].To != stretch(1).To {
		t.Errorf("the sentence runs %d-%d", cues[0].From, cues[0].To)
	}
	if cues[1].Text != "Sure." || cues[1].From != stretch(2).From {
		t.Errorf("the line after it says %q at %d", cues[1].Text, cues[1].From)
	}
}

// What was heard is not written over by putting its lines together.
func TestJoiningLeavesWhatWasHeardWhereItIs(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila."}
	u, v, shelved, hash := wholly(t,
		map[int]string{0: joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	heard := cued(t, shelved, text.Artifact(text.ASR, hash))
	if len(heard) != 2 || heard[0].Text != words[0] || heard[1].Text != words[1] {
		t.Errorf("what was heard now says %+v", heard)
	}
}

// overlapping is a run over four stretches cut into two batches sharing three
// lines, with as many batches to a request as asked for.
func overlapping(t *testing.T, batches int, says map[int]string) (PutRight, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := hearing(t,
		says,
		"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all.")
	u.Lines, u.Overlap, u.Batches = 3, 2, batches
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
			u, v, shelved, hash := overlapping(t, batches, map[int]string{
				0: joins(0, 1, putTogether),
				1: corrects(1, onItsOwn),
			})

			if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
				t.Fatal(err)
			}

			cues := cued(t, shelved, text.Said(text.ASR, hash))
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
	u, v, shelved, hash := overlapping(t, 1, map[int]string{
		0: joins(0, 1, putTogether),
		1: joins(1, 2, "Connected with Radhika. Sure."),
	})

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := cued(t, shelved, text.Said(text.ASR, hash))
	if len(cues) != 3 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[0].Text != putTogether || cues[1].Text != "Sure." {
		t.Errorf("the transcript says %+v", cues)
	}
	// The run that stands reaches to the end of the second stretch, and the one
	// dropped moved no moment.
	if cues[0].To != stretch(1).To || cues[1].To != stretch(2).To {
		t.Errorf("the lines run to %d and %d", cues[0].To, cues[1].To)
	}
}

// A transcript whose lines were put together is taken up again at the moment
// the count stands at, which putting them together did not move. The numbers
// the lines are known by are not the same ones the run before it saw.
func TestARunTakesUpATranscriptWhoseLinesWerePutTogether(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all."}
	u, v, shelved, by, hash := hearing(t, map[int]string{0: joins(0, 1, putTogether)}, words...)
	u.Lines, u.Overlap, u.Batches = 2, 0, 1

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

	cues := cued(t, shelved, text.Said(text.ASR, hash))
	if len(cues) != 3 || cues[0].Text != putTogether {
		t.Errorf("the transcript says %+v", cues)
	}
}
