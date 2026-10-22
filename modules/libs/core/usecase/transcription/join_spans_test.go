package transcription

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// newWholeRun is a run over a transcript asked about in one window.
func newWholeRun(
	t *testing.T, told map[int]domain.ProofreadCorrection, words ...string,
) (Proofread, domain.Vault, *fakeStorage, string) {
	t.Helper()
	u, v, retained, delta, hash := newProofreadTranscript(t, told, words...)
	delta.limit = len(words)
	return u, v, retained, hash
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
func TestJoiningPreservesWhatWasHeardWhereItIs(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila."}
	u, v, shelved, hash := newWholeRun(t,
		says(joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")),
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	received := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash))
	if len(received) != 2 || received[0].Text != words[0] || received[1].Text != words[1] {
		t.Errorf("what was heard now says %+v", received)
	}
}

// newOverlappingRun is a run over four spans, with as many lines to a call as
// asked for.
func newOverlappingRun(
	t *testing.T, lines int, told map[int]domain.ProofreadCorrection,
) (Proofread, domain.Vault, *fakeStorage, string) {
	t.Helper()
	u, v, retained, delta, hash := newProofreadTranscript(t,
		told,
		"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all.")
	delta.limit = lines
	return u, v, retained, hash
}

const (
	putTogether = "Krishna is Radha, Krishna is connected with Radhika."
	onItsOwn    = "connected with Radhika."
)

// sequenced is a proofreader answering each call with the next of its replies.
type sequenced struct {
	noScan
	replies []domain.SpeechProofreading
	calls   int
}

func (s *sequenced) GetName() string { return "a proofreader" }

func (s *sequenced) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	if s.calls >= len(s.replies) {
		return domain.SpeechProofreading{}, fmt.Errorf("call %d past the %d replies", s.calls, len(s.replies))
	}
	reply := s.replies[s.calls]
	s.calls++
	return reply, nil
}

// A correction of a line another correction put into a run is answered too
// late: the words no longer stand on their own, and the correction is dropped.
//
// It holds whether the two corrections arrive in one call or in two. The second
// call takes up again at the last line of the run, as a sentence reaching past
// a cut is taken up.
func TestALineAlreadyInsertIntoARunIsLeftInIt(t *testing.T) {
	for name, one := range map[string]struct {
		lines int
		by    port.Proofreader
	}{
		"in one call": {4, &corrector{says: says(joins(0, 1, putTogether), fixes(1, onItsOwn))}},
		"in two calls": {2, &sequenced{replies: []domain.SpeechProofreading{
			{Corrections: []domain.ProofreadCorrection{joins(0, 1, putTogether)}, Next: 1},
			{Corrections: []domain.ProofreadCorrection{fixes(1, onItsOwn)}, Next: 4, IsDone: true},
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			u, v, shelved, hash := newOverlappingRun(t, one.lines, nil)
			u.by = one.by

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
func TestARunOverLinesAlreadyInsertTogetherIsDropped(t *testing.T) {
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

// runJoinUntilStopped is a run over four spans, two lines to a call, that puts
// the first two together and is stopped in its second call. It returns a run
// taking the transcript up again with the proofreader given.
func runJoinUntilStopped(t *testing.T, again port.Proofreader) (Proofread, domain.Vault, *fakeStorage, string) {
	t.Helper()
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure.", "That is all."}
	u, v, shelved, delta, hash := newProofreadTranscript(t, says(joins(0, 1, putTogether)), words...)
	delta.limit = 2

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	delta.stop = func(calls int) {
		if calls == 2 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	resumed := newTranscriptProofreading(t, u.readers, u.derived, again)
	return resumed, v, shelved, hash
}

// A transcript whose lines were put together is taken up again at the moment
// the lines are known by are not the same ones the run before it saw.
func TestARunConsumesUpATranscriptWhoseLinesWereInsertTogether(t *testing.T) {
	again, v, shelved, hash := runJoinUntilStopped(t, &corrector{})

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
	again, v, shelved, hash := runJoinUntilStopped(t, scripted{reply: domain.SpeechProofreading{
		Corrections: []domain.ProofreadCorrection{fixes(0, onItsOwn)},
		IsDone:      true,
	}})
	name := domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)
	remaining := readCues(t, shelved, name)

	if _, err := again.Execute(t.Context(), v, recordingPath); !errors.Is(err, errCorrectionUnasked) {
		t.Fatalf("got %v, want %v", err, errCorrectionUnasked)
	}

	cues := readCues(t, shelved, name)
	if len(cues) != len(remaining) || cues[0].Text != putTogether {
		t.Fatalf("the transcript says %+v", cues)
	}
	for item := range cues {
		if cues[item].Text != remaining[item].Text {
			t.Errorf("line %d says %q, the run before left %q", item, cues[item].Text, remaining[item].Text)
		}
	}
}
