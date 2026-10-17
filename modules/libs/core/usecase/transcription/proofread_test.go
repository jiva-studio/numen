package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// hearing is a Proofread over one vault holding one recording whose
// transcript is written down already, and the hash that transcript is kept
// under.
//
// One line to a batch and one batch to a request, so the number a reply is
// about is the number of the line it puts right.
func newProofreadTranscript(
	t *testing.T, says map[int]string, words ...string,
) (Proofread, domain.Vault, *shelf, *corrector, string) {
	t.Helper()
	raw := newRecording(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	kept := newShelf()

	hash := text.Fingerprint(raw)
	if err := kept.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash), transcript.Marshal(newCues(words))); err != nil {
		t.Fatal(err)
	}
	by := &corrector{says: says}
	return newTranscriptProofreading(t, vaults{first.ID: shelved}, shelves{kept}, by),
		first, kept, by, hash
}

// newTranscriptProofreading is one built the only way there is, with the
// batching the tests here read a line at a time by.
func newTranscriptProofreading(
	t *testing.T, readers port.VaultReaders, derived port.DerivedStores, by port.Proofreader,
) Proofread {
	t.Helper()
	made, err := NewProofread(readers, derived, by)
	if err != nil {
		t.Fatal(err)
	}
	made.BatchSize = 1
	made.InFlight = 1
	return made
}

// newCues is the words as the cues a model wrote them down as.
func newCues(words []string) []domain.TranscriptCue {
	out := make([]domain.TranscriptCue, 0, len(words))
	for n, said := range words {
		out = append(out, domain.TranscriptCue{Text: said, From: getSpan(n).From, To: getSpan(n).To})
	}
	return out
}

// readCues is what a transcript on the shelf says, cue by cue.
func readCues(t *testing.T, shelved *shelf, name domain.ArtifactName) []domain.TranscriptCue {
	t.Helper()
	_, cues := transcript.Parse(readShelf(t, shelved, name))
	return cues
}

// A proofreading with nothing to proofread with is refused where it would be
// made, so no such thing exists to be called. An installation that named no
// profile has no proofreader at all, so this is a value the settings produce
// and not a caller's slip.
func TestNoTranscriptProofreadingIsMadeWithNothingToProofreadWith(t *testing.T) {
	u, _, _, by, _ := newProofreadTranscript(t, nil, "first thing", "secnd thing")

	for name, one := range map[string]struct {
		readers port.VaultReaders
		derived port.DerivedStores
		by      port.Proofreader
		want    error
	}{
		"nothing to proofread with": {u.readers, u.derived, nil, run.ErrNothingProofreads},
		"no vault":                  {nil, u.derived, by, run.ErrNoVaultToProofread},
		"no store":                  {u.readers, nil, by, run.ErrNoStoreToProofread},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProofread(one.readers, one.derived, one.by); !errors.Is(err, one.want) {
				t.Fatalf("made with %v", err)
			}
		})
	}

	// The control: all three, and it is made.
	if _, err := NewProofread(u.readers, u.derived, by); err != nil {
		t.Fatalf("a proofreading with everything it needs was refused: %v", err)
	}
}

func TestATranscriptIsPutRightAndEveryTimingStands(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, map[int]string{1: corrects(1, "second thing")}, words...)

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 3 || res.Read != 3 || res.Fixed != 1 || res.Left != 2 || res.UncorrectedBatches != 0 {
		t.Errorf("got %+v", res)
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
	if len(cues) != len(words) {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[1].Text != "second thing" {
		t.Errorf("the second line says %q", cues[1].Text)
	}
	for at, cue := range cues {
		if cue.From != getSpan(at).From || cue.To != getSpan(at).To {
			t.Errorf("line %d is now %d to %d", at, cue.From, cue.To)
		}
	}
	if cues[0].Text != words[0] || cues[2].Text != words[2] {
		t.Errorf("a line nothing was said about says %q and %q", cues[0].Text, cues[2].Text)
	}

	var stood putting
	if err := json.Unmarshal(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash)), &stood); err != nil {
		t.Fatal(err)
	}
	if stood.By != "a proofreader" || stood.At != getSpan(2).To {
		t.Errorf("got %+v", stood)
	}
}

func TestWhatTheModelHeardIsNotWrittenOver(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, map[int]string{1: corrects(1, "second thing")}, words...)
	was := string(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash)))

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if now := string(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash))); now != was {
		t.Errorf("the artifact now says %q", now)
	}
}

func TestAReplyThatIsNoAnswerLeavesItsLinesAsHeard(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, map[int]string{
		1: refused,
	}, words...)

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.UncorrectedBatches != 1 || res.Fixed != 0 || res.Left != 2 {
		t.Errorf("got %+v", res)
	}
	if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
		t.Error("a transcript nothing put right was written beside the artifact")
	}
}

func TestATranscriptIsTakenUpWhereTheRunBeforeStopped(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, by, hash := newProofreadTranscript(t, map[int]string{
		0: corrects(0, "the first thing"),
		1: corrects(1, "second thing"),
	}, words...)

	ctx, stop := context.WithCancel(t.Context())
	by.stop = func(requests int) {
		if requests == 3 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	said := &corrector{says: map[int]string{3: corrects(3, "fourth thing")}}
	again := newTranscriptProofreading(t, u.readers, u.derived, said)
	res, err := again.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d lines of the transcript", res.Resumed)
	}
	for _, asked := range said.asked {
		for _, batch := range asked {
			if batch < 2 {
				t.Errorf("asked about line %d again", batch)
			}
		}
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
	if len(cues) != 4 {
		t.Fatalf("the transcript says %+v", cues)
	}
	for at, said := range []string{"the first thing", "second thing", "third thing", "fourth thing"} {
		if cues[at].Text != said {
			t.Errorf("line %d says %q", at, cues[at].Text)
		}
	}
}

// A batch reaches back over the lines it shares with the one before it, and a
// run taking up in it reads those lines again. What it says it has read never
// passes what the transcript holds.
func TestAResumedRunReadsNoMoreLinesThanTheTranscriptHas(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, nil, words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 1, 1
	if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), transcript.Marshal(newCues(words))); err != nil {
		t.Fatal(err)
	}
	stood, err := json.Marshal(putting{By: "a proofreader", At: getSpan(1).To})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash), stood); err != nil {
		t.Fatal(err)
	}

	furthest := 0
	u.OnProgress = func(res ProofreadResult) {
		if res.Read > res.Lines {
			t.Errorf("read %d lines of %d", res.Read, res.Lines)
		}
		if res.Left < 0 {
			t.Errorf("%d lines stand as they were heard", res.Left)
		}
		furthest = max(furthest, res.Read)
	}

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 || res.Lines != 4 {
		t.Errorf("got %+v", res)
	}
	if furthest != 4 {
		t.Errorf("the run stopped at line %d of 4", furthest)
	}
}

func TestOneRunToARecordingBeingPutRight(t *testing.T) {
	u, v, shelved, by, hash := newProofreadTranscript(t, nil, "first thing", "secnd thing")
	shelved.hold(domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, hash))

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsBusy {
		t.Error("a recording another run holds was put right")
	}
	if len(by.asked) != 0 {
		t.Errorf("it asked about %v", by.asked)
	}
}

func TestATranscriptSomebodyElseWroteIsLeftAsTheyLeftIt(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	own := transcript.Marshal(newCues([]string{"first thing", "what a person typed"}))

	for _, one := range []struct {
		name  string
		said  []byte
		stood putting
	}{
		{"a person wrote it in the window", append(own, transcript.GetHandWrittenMark()...), putting{By: "a proofreader", At: getSpan(1).To}},
		{"nobody here wrote it", own, putting{}},
		{"another proofreader wrote it", own, putting{By: "somebody else", At: getSpan(1).To}},
	} {
		t.Run(one.name, func(t *testing.T) {
			u, v, shelved, by, hash := newProofreadTranscript(t, map[int]string{1: corrects(1, "second thing")}, words...)
			if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), one.said); err != nil {
				t.Fatal(err)
			}
			if one.stood.By != "" {
				stood, err := json.Marshal(one.stood)
				if err != nil {
					t.Fatal(err)
				}
				if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash), stood); err != nil {
					t.Fatal(err)
				}
			}

			res, err := u.Execute(t.Context(), v, recordingPath)
			if err != nil {
				t.Fatal(err)
			}
			if !res.IsEdited || res.Fixed != 0 {
				t.Errorf("got %+v", res)
			}
			if len(by.asked) != 0 {
				t.Errorf("it asked about %v", by.asked)
			}
			if now := string(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))); now != string(one.said) {
				t.Errorf("the transcript now says %q", now)
			}
		})
	}
}

func TestATranscriptThatIsNotThereIsNothingToPutRight(t *testing.T) {
	u, v, shelved, _, hash := newProofreadTranscript(t, nil, "first thing")
	if err := shelved.RemoveArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash)); err != nil {
		t.Fatal(err)
	}

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNone {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsCutAgainAsItsLinesArePutRight(t *testing.T) {
	u, v, _, _, _ := newProofreadTranscript(t, map[int]string{0: corrects(0, "the first thing")}, "first thing", "secnd thing")
	cuts := 0
	u.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	// Two lines, one to a request.
	if cuts != 2 {
		t.Errorf("cut %d times", cuts)
	}
}

// A transcript nothing is left to be asked about is not work, and nothing is
// told about it.
func TestATranscriptAtItsLastLineReportsNoProgress(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, _, _, _ := newProofreadTranscript(t, map[int]string{1: corrects(1, "second thing")}, words...)
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	told := 0
	again := newTranscriptProofreading(t, u.readers, u.derived, &corrector{})
	again.OnProgress = func(ProofreadResult) { told++ }
	res, err := again.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Lines {
		t.Fatalf("got %+v", res)
	}
	if told != 0 {
		t.Errorf("a transcript with nothing left to put right was told about %d times", told)
	}
}

// A transcript proofreading given a known hash skips reading the source file from disk.
func TestATranscriptProofreadingGivenAKnownHashDoesNotReadTheSourceFile(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, _, _, hash := newProofreadTranscript(t, map[int]string{1: corrects(1, "second thing")}, words...)
	u.readers = brokenReaders{}
	u.Hash = hash

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatalf("expected execute with known hash to succeed: %v", err)
	}
	if res.Fixed != 1 {
		t.Errorf("expected 1 fixed line, got %+v", res)
	}
}

// The proofreader is asked about a window of the transcript at a time: the
// whole transcript, where to begin, how many lines to answer and how the
// batches are cut.
func TestTheRequestIsForSpeechInWindowsOfBatches(t *testing.T) {
	u, v, _, by, _ := newProofreadTranscript(t, nil, "one", "two", "three", "four", "five")
	u.BatchSize, u.Overlap, u.InFlight = 2, 1, 2

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if len(by.requests) != 2 {
		t.Fatalf("asked %d times", len(by.requests))
	}
	first, second := by.requests[0], by.requests[1]
	if first.Kind != domain.ProofreadSpeech || first.Size != 2 || first.Overlap != 1 || first.LineLimit != 4 {
		t.Errorf("asked %+v", first)
	}
	if len(first.Groups) != 1 || len(first.Groups[0].Lines) != 5 {
		t.Errorf("the transcript was sent as %+v", first.Groups)
	}
	if first.FromLine != 0 || second.FromLine != 3 {
		t.Errorf("the windows began at %d and %d", first.FromLine, second.FromLine)
	}
}

// stuck is a proofreader that answers and says to continue from where it began.
type stuck struct{}

func (stuck) GetName() string { return "a proofreader that stays" }

func (stuck) Proofread(_ context.Context, request domain.ProofreadRequest) (domain.ProofreadAnswer, error) {
	return domain.ProofreadAnswer{
		Corrections: []domain.ProofreadCorrection{{Number: request.FromLine, Last: request.FromLine, Text: "changed"}},
		NextLine:    request.FromLine,
	}, nil
}

// A proofreader that does not move on ends the run before anything it said is
// written down, so the next run asks about the same lines.
func TestAProofreaderThatDoesNotMoveOnEndsTheRunUnwritten(t *testing.T) {
	u, v, shelved, _, hash := newProofreadTranscript(t, nil, "first thing", "secnd thing")
	u.by = stuck{}

	if _, err := u.Execute(t.Context(), v, recordingPath); err == nil {
		t.Fatal("a proofreader that stayed was waited on for ever")
	}
	if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
		t.Error("what a proofreader that stayed said was written down")
	}
}
