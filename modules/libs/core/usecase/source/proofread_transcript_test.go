package source

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// hearing is a ProofreadTranscript over one vault holding one recording whose
// transcript is written down already, and the hash that transcript is kept
// under.
//
// One line to a batch and one batch to a request, so the number a reply is
// about is the number of the line it puts right.
func hearing(
	t *testing.T, says map[int]string, words ...string,
) (ProofreadTranscript, domain.Vault, *shelf, *corrector, string) {
	t.Helper()
	raw := recorded(words)
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, raw, 1)
	kept := newShelf()

	hash := text.Fingerprint(raw)
	if err := kept.Write(t.Context(), text.Artifact(text.ASR, hash), transcript.Marshal(heard(words))); err != nil {
		t.Fatal(err)
	}
	by := &corrector{says: says}
	return ProofreadTranscript{
		Readers:   vaults{first.ID: shelved},
		Derived:   kept,
		By:        by,
		BatchSize: 1,
		InFlight:  1,
	}, first, kept, by, hash
}

// heard is the words as the cues a model wrote them down as.
func heard(words []string) []transcript.Cue {
	out := make([]transcript.Cue, 0, len(words))
	for n, said := range words {
		out = append(out, transcript.Cue{Text: said, From: stretch(n).From, To: stretch(n).To})
	}
	return out
}

// cued is what a transcript on the shelf says, cue by cue.
func cued(t *testing.T, shelved *shelf, name string) []transcript.Cue {
	t.Helper()
	_, cues := transcript.Parse(kept(t, shelved, name))
	return cues
}

// TestNothingIsPutRightWhereNothingWasConfiguredToProofreadWith. An
// installation that named no profile has no proofreader, so nil is what the
// settings hand over and the constructor takes it without a word. A caller that
// forgot to refuse it first is answered, not brought down mid-transcript.
func TestNothingIsPutRightWhereNothingWasConfiguredToProofreadWith(t *testing.T) {
	u, v, _, by, _ := hearing(t, nil, "first thing", "secnd thing")

	if _, err := NewProofreadTranscript(u.Readers, u.Derived, nil).
		Execute(t.Context(), v, recordingPath); !errors.Is(err, errNothingProofreads) {
		t.Fatalf("a transcript was put right with no proofreader: %v", err)
	}
	// The control: the same transcript, through the same constructor, with a
	// proofreader.
	if _, err := NewProofreadTranscript(u.Readers, u.Derived, by).
		Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatalf("a transcript with a proofreader was refused: %v", err)
	}
}

func TestATranscriptIsPutRightAndEveryTimingStands(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing"}
	u, v, shelved, _, hash := hearing(t, map[int]string{1: corrects(1, "second thing")}, words...)

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 3 || res.Read != 3 || res.Fixed != 1 || res.Left != 2 || res.Refused != 0 {
		t.Errorf("got %+v", res)
	}

	cues := cued(t, shelved, text.Corrected(text.ASR, hash))
	if len(cues) != len(words) {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[1].Text != "second thing" {
		t.Errorf("the second line says %q", cues[1].Text)
	}
	for at, cue := range cues {
		if cue.From != stretch(at).From || cue.To != stretch(at).To {
			t.Errorf("line %d is now %d to %d", at, cue.From, cue.To)
		}
	}
	if cues[0].Text != words[0] || cues[2].Text != words[2] {
		t.Errorf("a line nothing was said about says %q and %q", cues[0].Text, cues[2].Text)
	}

	var stood putting
	if err := json.Unmarshal(kept(t, shelved, text.Proofread(text.ASR, hash)), &stood); err != nil {
		t.Fatal(err)
	}
	if stood.By != "a proofreader" || stood.At != stretch(2).To {
		t.Errorf("got %+v", stood)
	}
}

func TestWhatTheModelHeardIsNotWrittenOver(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, _, hash := hearing(t, map[int]string{1: corrects(1, "second thing")}, words...)
	was := string(kept(t, shelved, text.Artifact(text.ASR, hash)))

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if now := string(kept(t, shelved, text.Artifact(text.ASR, hash))); now != was {
		t.Errorf("the artifact now says %q", now)
	}
}

func TestAReplyThatIsNoAnswerLeavesItsLinesAsHeard(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, shelved, _, hash := hearing(t, map[int]string{
		1: proofread.Opens + "1" + proofread.Closes + "second thing",
	}, words...)

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Refused != 1 || res.Fixed != 0 || res.Left != 2 {
		t.Errorf("got %+v", res)
	}
	if _, err := shelved.Read(t.Context(), text.Corrected(text.ASR, hash)); err == nil {
		t.Error("a transcript nothing put right was written beside the artifact")
	}
}

func TestATranscriptIsTakenUpWhereTheRunBeforeStopped(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, by, hash := hearing(t, map[int]string{
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

	again := &corrector{says: map[int]string{3: corrects(3, "fourth thing")}}
	u.By = again
	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d lines of the transcript", res.Resumed)
	}
	for _, asked := range again.asked {
		for _, batch := range asked {
			if batch < 2 {
				t.Errorf("asked about line %d again", batch)
			}
		}
	}

	cues := cued(t, shelved, text.Corrected(text.ASR, hash))
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
	u, v, shelved, _, hash := hearing(t, nil, words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 1, 1
	if err := shelved.Write(t.Context(), text.Corrected(text.ASR, hash), transcript.Marshal(heard(words))); err != nil {
		t.Fatal(err)
	}
	stood, err := json.Marshal(putting{By: "a proofreader", At: stretch(1).To})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), text.Proofread(text.ASR, hash), stood); err != nil {
		t.Fatal(err)
	}

	furthest := 0
	u.OnProgress = func(res ProofreadTranscriptResult) {
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
	u, v, shelved, by, hash := hearing(t, nil, "first thing", "secnd thing")
	shelved.hold(text.Partial(text.ASR, hash))

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Busy {
		t.Error("a recording another run holds was put right")
	}
	if len(by.asked) != 0 {
		t.Errorf("it asked about %v", by.asked)
	}
}

func TestATranscriptSomebodyElseWroteIsLeftAsTheyLeftIt(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	own := transcript.Marshal(heard([]string{"first thing", "what a person typed"}))

	for _, one := range []struct {
		name  string
		said  []byte
		stood putting
	}{
		{"a person wrote it in the window", append(own, transcript.Hand()...), putting{By: "a proofreader", At: stretch(1).To}},
		{"nobody here wrote it", own, putting{}},
		{"another proofreader wrote it", own, putting{By: "somebody else", At: stretch(1).To}},
	} {
		t.Run(one.name, func(t *testing.T) {
			u, v, shelved, by, hash := hearing(t, map[int]string{1: corrects(1, "second thing")}, words...)
			if err := shelved.Write(t.Context(), text.Corrected(text.ASR, hash), one.said); err != nil {
				t.Fatal(err)
			}
			if one.stood.By != "" {
				stood, err := json.Marshal(one.stood)
				if err != nil {
					t.Fatal(err)
				}
				if err := shelved.Write(t.Context(), text.Proofread(text.ASR, hash), stood); err != nil {
					t.Fatal(err)
				}
			}

			res, err := u.Execute(t.Context(), v, recordingPath)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Edited || res.Fixed != 0 {
				t.Errorf("got %+v", res)
			}
			if len(by.asked) != 0 {
				t.Errorf("it asked about %v", by.asked)
			}
			if now := string(kept(t, shelved, text.Corrected(text.ASR, hash))); now != string(one.said) {
				t.Errorf("the transcript now says %q", now)
			}
		})
	}
}

func TestATranscriptThatIsNotThereIsNothingToPutRight(t *testing.T) {
	u, v, shelved, _, hash := hearing(t, nil, "first thing")
	if err := shelved.Remove(t.Context(), text.Artifact(text.ASR, hash)); err != nil {
		t.Fatal(err)
	}

	res, err := u.Execute(t.Context(), v, recordingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.None {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsCutAgainAsItsLinesArePutRight(t *testing.T) {
	u, v, _, _, _ := hearing(t, map[int]string{0: corrects(0, "the first thing")}, "first thing", "secnd thing")
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

// Every batch of a transcript carries what the whole recording holds: how its
// speech opens, and the words that recur through it.
func TestEveryBatchCarriesWhatTheRecordingHolds(t *testing.T) {
	words := []string{
		"The assembly at Mithila heard Ganaka.",
		"The teacher listened. Then Ganaka spoke of Mithila",
		"as a city nobody had named before him.",
	}
	u, v, _, by, _ := hearing(t, nil, words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if len(by.about) != len(words) {
		t.Fatalf("it was asked about %d batches", len(by.about))
	}
	for _, about := range by.about {
		for _, want := range []string{"The speech opens: The assembly at Mithila", "Mithila, Ganaka"} {
			if !strings.Contains(about, want) {
				t.Errorf("a batch says the recording holds %q, and it does not carry %q", about, want)
			}
		}
	}
}

// A transcript nothing is left to be asked about is not work, and nothing is
// told about it.
func TestATranscriptAtItsLastLineReportsNoProgress(t *testing.T) {
	words := []string{"first thing", "secnd thing"}
	u, v, _, _, _ := hearing(t, map[int]string{1: corrects(1, "second thing")}, words...)
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	told := 0
	u.By = &corrector{}
	u.OnProgress = func(ProofreadTranscriptResult) { told++ }
	res, err := u.Execute(t.Context(), v, recordingPath)
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

// The seams are asked about after the whole transcript is, and a run taking up
// among them is work a person is told about.
func TestARunTakingUpAmongTheSeamsIsToldAbout(t *testing.T) {
	words := []string{"first thing", "secnd thing", "third thing", "forth thing"}
	u, v, shelved, _, hash := hearing(t, nil, words...)
	u.BatchSize, u.Overlap, u.InFlight = 2, 1, 1
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	// The shelf as a run that ended between the two passes left it: every line
	// asked about, and no seam.
	stood, err := json.Marshal(putting{By: u.By.Name(), At: stretch(len(words) - 1).To})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.Write(t.Context(), text.Proofread(text.ASR, hash), stood); err != nil {
		t.Fatal(err)
	}

	told := 0
	u.OnProgress = func(ProofreadTranscriptResult) { told++ }
	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}
	if told == 0 {
		t.Error("a run over the seams was told about no times")
	}
}
