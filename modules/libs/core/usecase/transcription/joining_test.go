package transcription

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// newWholeRun is a run over a transcript asked about in one window.
func newWholeRun(
	t *testing.T, told map[int]domain.ProofreadCorrection, words ...string,
) (Proofread, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := newProofreadTranscript(t, told, words...)
	u.BatchSize = len(words)
	return u, v, kept, hash
}

// A sentence the recording broke across two spans comes back as one line,
// running from the first moment to the last.
func TestASentenceBrokenAcrossSpansBecomesOneLine(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure."}
	u, v, shelved, hash := newWholeRun(t,
		says(joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")),
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
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
		says(joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")),
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	heard := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash))
	if len(heard) != 2 || heard[0].Text != words[0] || heard[1].Text != words[1] {
		t.Errorf("what was heard now says %+v", heard)
	}
}

// newOverlappingRun is a run over four spans, with as many lines to a call as
// asked for.
func newOverlappingRun(
	t *testing.T, lines int, told map[int]domain.ProofreadCorrection,
) (Proofread, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := newProofreadTranscript(t,
		told,
		"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all.")
	u.BatchSize, u.Overlap, u.InFlight = lines, 0, 1
	return u, v, kept, hash
}

// The joined sentence, and the transcript it leaves behind it.
const (
	putTogether = "Krishna is Radha, Krishna is connected with Radhika."
	onItsOwn    = "connected with Radhika."
)

// A correction of a line another correction put into a run is answered too
// late: the words no longer stand on their own, and the correction is dropped.
//
// It holds whether the two corrections arrive in one call or in two.
func TestALineAlreadyPutIntoARunIsLeftInIt(t *testing.T) {
	for _, lines := range []int{1, 4} {
		t.Run(fmt.Sprintf("%d lines to a call", lines), func(t *testing.T) {
			u, v, shelved, hash := newOverlappingRun(t, lines, says(
				joins(0, 1, putTogether),
				fixes(1, onItsOwn),
			))

			if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
				t.Fatal(err)
			}

			cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
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

// Two corrections of runs over the same lines: the run standing first puts them
// together, and the one reaching into it is dropped.
func TestARunOverLinesAlreadyPutTogetherIsDropped(t *testing.T) {
	u, v, shelved, hash := newOverlappingRun(t, 4, says(
		joins(0, 1, putTogether),
		joins(1, 2, "Connected with Radhika. Sure."),
	))

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
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
	u, v, shelved, by, hash := newProofreadTranscript(t, says(joins(0, 1, putTogether)), words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 0, 1

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	by.stop = func(calls int) {
		if calls == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	again := newTranscriptProofreading(t, u.readers, u.derived, &corrector{})
	again.BatchSize, again.Overlap, again.InFlight = 2, 0, 1
	res, err := again.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	// Three lines stand where four were heard, and the run before this one had
	// asked about the first of them.
	if res.Lines != 3 || res.Resumed != 1 || res.Read != 3 {
		t.Errorf("got %+v", res)
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
	if len(cues) != 3 || cues[0].Text != putTogether {
		t.Errorf("the transcript says %+v", cues)
	}
}

// A run taking up a transcript asks about the lines from where the run before
// stopped. A correction of a line before them is no answer to what was asked,
// and the transcript stands as the run before left it.
func TestARunTakingUpATranscriptRefusesACorrectionOfALineItDidNotAsk(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all."}
	u, v, shelved, by, hash := newProofreadTranscript(t, says(joins(0, 1, putTogether)), words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 0, 1

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	by.stop = func(calls int) {
		if calls == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}
	name := domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)
	left := readCues(t, shelved, name)

	again := newTranscriptProofreading(t, u.readers, u.derived, scripted{reply: domain.SpeechProofreading{
		Corrections: []domain.ProofreadCorrection{fixes(0, onItsOwn)},
		IsDone:      true,
	}})
	again.BatchSize, again.Overlap, again.InFlight = 2, 0, 1
	if _, err := again.Execute(t.Context(), v, recordingPath); !errors.Is(err, errCorrectionUnasked) {
		t.Fatalf("got %v, want %v", err, errCorrectionUnasked)
	}

	cues := readCues(t, shelved, name)
	if len(cues) != len(left) || cues[0].Text != putTogether {
		t.Fatalf("the transcript says %+v", cues)
	}
	for at := range cues {
		if cues[at].Text != left[at].Text {
			t.Errorf("line %d says %q, the run before left %q", at, cues[at].Text, left[at].Text)
		}
	}
}
