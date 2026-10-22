package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// scripted is a proofreader answering every call with the same reply.
type scripted struct {
	noScan
	reply domain.SpeechProofreading
}

func (s scripted) GetName() string { return "a proofreader" }

func (s scripted) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return s.reply, nil
}

// newProofreadTranscript is a Proofread over one vault holding one recording
// whose transcript is written down already, and the hash that transcript is
// kept under.
//
// One line to a request and one request to a call, so a call answers about one
// line.
func newProofreadTranscript(
	t *testing.T, told map[int]domain.ProofreadCorrection, words ...string,
) (Proofread, domain.Vault, *shelf, *corrector, string) {
	t.Helper()
	raw := newRecording(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	shelf := newShelf()

	hash := text.Fingerprint(raw)
	if err := shelf.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash), transcript.Marshal(newCues(words))); err != nil {
		t.Fatal(err)
	}
	by := &corrector{says: told, limit: 1}
	return newTranscriptProofreading(t, vaults{first.ID: shelved}, shelves{shelf}, by),
		first, shelf, by, hash
}

// newTranscriptProofreading is one built the only way there is, asking about a
// line at a time.
func newTranscriptProofreading(
	t *testing.T, readers port.VaultReaders, derived port.DerivedStores, by port.Proofreader,
) Proofread {
	t.Helper()
	made, err := NewProofread(readers, derived, by)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// newCues is the words as the cues a model wrote them down as.
func newCues(words []string) []domain.TranscriptCue {
	out := make([]domain.TranscriptCue, 0, len(words))
	for n, response := range words {
		out = append(out, domain.TranscriptCue{Text: response, From: getSpan(n).From, To: getSpan(n).To})
	}
	return out
}

// readCues is what a transcript on the shelf says, cue by cue.
func readCues(t *testing.T, shelved *shelf, name domain.ArtifactName) []domain.TranscriptCue {
	t.Helper()
	_, cues := transcript.Parse(readShelf(t, shelved, name))
	return cues
}

// A proofreading with nothing to proofread with fails where it would be
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
		t.Fatalf("a proofreading with everything it needs failed: %v", err)
	}
}

func TestATranscriptIsInsertRightAndEveryTimingExists(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, says(fixes(1, "second thing")), words...)

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

	var status putting
	if err := json.Unmarshal(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash)), &status); err != nil {
		t.Fatal(err)
	}
	if status.By != "a proofreader" || status.At != getSpan(2).To {
		t.Errorf("got %+v", status)
	}
}

func TestWhatTheModelHeardIsNotSavedOver(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, says(fixes(1, "second thing")), words...)
	was := string(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash)))

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if now := string(readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash))); now != was {
		t.Errorf("the artifact now says %q", now)
	}
}

func TestACallThatIsNoAnswerPreservesItsLinesAsHeard(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, by, hash := newProofreadTranscript(t, nil, words...)
	by.uncorrected = 1

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	// Two calls, one line to each.
	if res.UncorrectedBatches != 2 || res.Fixed != 0 || res.Left != 2 {
		t.Errorf("got %+v", res)
	}
	if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
		t.Error("a transcript nothing put right was written beside the artifact")
	}
}

func TestATranscriptIsTakenUpWhereTheRunBeforeStopped(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, by, hash := newProofreadTranscript(t, says(
		fixes(0, "the first thing"),
		fixes(1, "second thing"),
	), words...)

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	by.stop = func(calls int) {
		if calls == 3 {
			stop()
		}
	}
	if _, err := u.Execute(ctx, v, recordingPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped with %v", err)
	}

	response := &corrector{says: says(fixes(3, "fourth thing"))}
	again := newTranscriptProofreading(t, u.readers, u.derived, response)
	res, err := again.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d lines of the transcript", res.Resumed)
	}
	if from := response.froms[0]; from != 2 {
		t.Errorf("began at line %d", from)
	}
	for _, line := range response.lines() {
		if line < 2 {
			t.Errorf("asked about line %d again", line)
		}
	}

	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
	if len(cues) != 4 {
		t.Fatalf("the transcript says %+v", cues)
	}
	for at, responseVal := range []string{"the first thing", "second thing", "third thing", "fourth thing"} {
		if cues[at].Text != responseVal {
			t.Errorf("line %d says %q", at, cues[at].Text)
		}
	}
}

// A call reaches back over the lines it shares with the one before it, and a
// run taking up in it reads those lines again. What it says it has read never
// passes what the transcript holds.
func TestAResumedRunReadsNoMoreLinesThanTheTranscriptHas(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, nil, words...)
	if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), transcript.Marshal(newCues(words))); err != nil {
		t.Fatal(err)
	}
	status, err := json.Marshal(putting{By: "a proofreader", At: getSpan(1).To})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash), status); err != nil {
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

func TestOneRunToARecordingBeingInsertRight(t *testing.T) {
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
			u, v, shelved, by, hash := newProofreadTranscript(t, says(fixes(1, "second thing")), words...)
			if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), one.said); err != nil {
				t.Fatal(err)
			}
			if one.stood.By != "" {
				status, err := json.Marshal(one.stood)
				if err != nil {
					t.Fatal(err)
				}
				if err := shelved.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash), status); err != nil {
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

func TestATranscriptThatIsNotThereIsNothingToInsertRight(t *testing.T) {
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

func TestASourceIsSplitAgainAsItsLinesAreInsertRight(t *testing.T) {
	u, v, _, _, _ := newProofreadTranscript(t, says(fixes(0, "the first thing")), "first thing", "secnd thing")
	cuts := 0
	u.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	// Two lines, one to a call.
	if cuts != 2 {
		t.Errorf("cut %d times", cuts)
	}
}

// Every call is given the whole transcript's spoken lines, numbered by their
// cue, and the line to continue from.
func TestEveryCallIsGivenTheWholeTranscriptAndTheLineToContinueFrom(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, _, by, _ := newProofreadTranscript(t, nil, words...)
	by.limit = 2

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	want := []domain.ProofreadLine{
		{Number: 0, Text: "first thing"}, {Number: 1, Text: "secnd thing"},
		{Number: 2, Text: "third thing"}, {Number: 3, Text: "forth thing"},
	}
	if len(by.given) != 2 {
		t.Fatalf("it was called %d times", len(by.given))
	}
	for at, provided := range by.given {
		if !slices.Equal(provided, want) {
			t.Errorf("call %d was given %+v", at, provided)
		}
	}
	wantFroms := []int{0, 2}
	if !slices.Equal(by.froms, wantFroms) {
		t.Errorf("froms %+v, want %+v", by.froms, wantFroms)
	}
}

// A proofreader that stopped short of the end is called again from the line it
// says to continue from, and is given the same lines.
func TestAWindowShortOfTheEndIsFollowedByTheNext(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, by, hash := newProofreadTranscript(t, says(fixes(1, "second thing"), fixes(3, "fourth thing")), words...)
	by.limit = 2

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(by.froms) != 2 || by.froms[0] != 0 || by.froms[1] != 2 {
		t.Fatalf("froms %+v", by.froms)
	}
	if !slices.Equal(by.given[0], by.given[1]) {
		t.Errorf("the calls were given %+v and %+v", by.given[0], by.given[1])
	}
	if res.Fixed != 2 || res.Read != 4 || res.Left != 2 {
		t.Errorf("got %+v", res)
	}
	cues := readCues(t, shelved, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash))
	if cues[1].Text != "second thing" || cues[3].Text != "fourth thing" {
		t.Errorf("the transcript says %+v", cues)
	}
}

// The checkpoint stands at the end of the last line before the one to continue
// from, so a run taking up asks about that line first.
func TestTheCheckpointExistsWhereTheNextWindowBegins(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, shelved, by, hash := newProofreadTranscript(t, nil, words...)
	by.stop = func(calls int) {
		if calls == 2 {
			var status putting
			raw := readShelf(t, shelved, domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash))
			if err := json.Unmarshal(raw, &status); err != nil {
				t.Error(err)
			}
			if status.At != getSpan(0).To {
				t.Errorf("the checkpoint stands at %d, want %d", status.At, getSpan(0).To)
			}
		}
	}

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
}

// A transcript nothing is left to be asked about is not work, and nothing is
// told about it.
func TestATranscriptAtItsLastLineReportsNoProgress(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, _, _, _ := newProofreadTranscript(t, says(fixes(1, "second thing")), words...)
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	notifyCount := 0
	response := &corrector{}
	again := newTranscriptProofreading(t, u.readers, u.derived, response)
	again.OnProgress = func(ProofreadResult) { notifyCount++ }
	res, err := again.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Lines || !res.IsAlready {
		t.Fatalf("got %+v", res)
	}
	if notifyCount != 0 || len(response.froms) != 0 {
		t.Errorf("a transcript with nothing left to put right was told about %d times and asked %d", notifyCount, len(response.froms))
	}
}

// A proofreader that cannot be reached ends the run, and nothing it did not
// answer is written.
func TestAProofreaderWithNothingRoutedEndsTheRunAndWritesNothing(t *testing.T) {
	u, v, shelved, by, hash := newProofreadTranscript(t, nil, "first thing", "secnd thing")
	by.fail = fmt.Errorf("%w: no route for the purpose", port.ErrUnavailable)

	_, err := u.Execute(t.Context(), v, recordingPath)

	if !errors.Is(err, port.ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
		t.Error("corrections were written for a proofreading nothing answered")
	}
}

// A proofreader that says to continue from the line it began at would be asked
// the same thing for ever.
func TestAProofreaderThatMakesNoProgressEndsTheRun(t *testing.T) {
	u, v, _, _, _ := newProofreadTranscript(t, nil, "first thing", "secnd thing")
	u.by = scripted{reply: domain.SpeechProofreading{Next: 0}}

	if _, err := u.Execute(t.Context(), v, recordingPath); !errors.Is(err, errNoProgress) {
		t.Errorf("got %v", err)
	}
}

// A correction of a line the transcript does not have is no correction.
func TestACorrectionOfALineTheTranscriptDoesNotHaveEndsTheRun(t *testing.T) {
	for name, corrections := range map[string][]domain.ProofreadCorrection{
		"past the end":     {fixes(9, "invented")},
		"a run past it":    {joins(0, 9, "invented")},
		"before the start": {fixes(-1, "invented")},
	} {
		t.Run(name, func(t *testing.T) {
			u, v, shelved, _, hash := newProofreadTranscript(t, nil, "first thing", "secnd thing")
			u.by = scripted{reply: domain.SpeechProofreading{Corrections: corrections, IsDone: true, Next: 2}}

			if _, err := u.Execute(t.Context(), v, recordingPath); !errors.Is(err, errCorrectionOutside) {
				t.Fatalf("got %v", err)
			}
			if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
				t.Error("a correction of nothing was written")
			}
		})
	}
}

// A correction of a line before where this window began is no answer to what was
// asked.
func TestACorrectionOfAnUnaskedLineEndsTheRun(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, shelved, _, hash := newProofreadTranscript(t, nil, words...)
	u.by = &sequenced{replies: []domain.SpeechProofreading{
		{Next: 1},
		{Corrections: []domain.ProofreadCorrection{fixes(0, "past speech")}, Next: 3, IsDone: true},
	}}

	if _, err := u.Execute(t.Context(), v, recordingPath); !errors.Is(err, errCorrectionUnasked) {
		t.Fatalf("got %v, want %v", err, errCorrectionUnasked)
	}
	if _, err := shelved.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); err == nil {
		t.Error("a correction of an unasked line was written")
	}
}

// The end of the context stops the run between two windows.
func TestTheEndOfTheContextStopsTheRunBetweenWindows(t *testing.T) {
	u, v, _, by, _ := newProofreadTranscript(t, nil, "first thing", "secnd thing", "third thing")
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	u.OnProgress = func(res ProofreadResult) {
		if res.Read > res.Resumed {
			stop()
		}
	}

	_, err := u.Execute(ctx, v, recordingPath)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(by.froms) != 1 {
		t.Errorf("it was called %d times", len(by.froms))
	}
}

// A transcript proofreading given a known hash skips reading the source file from disk.
func TestATranscriptProofreadingGivenAKnownHashDoesNotReadTheSourceFile(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, _, _, hash := newProofreadTranscript(t, says(fixes(1, "second thing")), words...)
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
