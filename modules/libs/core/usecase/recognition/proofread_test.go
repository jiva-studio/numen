package recognition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// proofreadable is a recognition of the fixture, already written down, and what a
// Proofread over it is made out of.
type proofreadable struct {
	readers port.VaultReaders
	derived port.DerivedStores
	vault   domain.Vault
	shelved *fakeStorage
}

// put is a proofreading of that recognition, by the proofreader given.
func (p proofreadable) put(t *testing.T, delta port.Proofreader) Proofread {
	t.Helper()
	made, err := NewProofread(p.readers, p.derived, delta)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// newProofreadRecognition is that recognition, a proofreading of it, and the
// proofreader the proofreading was made with.
func newProofreadRecognition(t *testing.T, says map[int]domain.ProofreadCorrection) (Proofread, domain.Vault, *fakeStorage, *corrector) {
	t.Helper()
	target := newProofreadable(t)
	delta := &corrector{says: says}
	return target.put(t, delta), target.vault, target.shelved, delta
}

// newProofreadable is the fixture's recognition, written down and ready to be put
// right.
func newProofreadable(t *testing.T) proofreadable {
	t.Helper()
	read, v, _, shelved, _ := createRecognise(t, "the words", outline)
	if _, err := read.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	return proofreadable{readers: read.readers, derived: fakeStores{shelved}, vault: v, shelved: shelved}
}

// A proofreading with nothing to proofread with fails where it would be
// made, so no such thing exists to be called. An installation that named no
// profile has no proofreader at all, so this is a value the settings produce
// and not a caller's slip.
func TestNoProofreadingIsMadeWithNothingToProofreadWith(t *testing.T) {
	target := newProofreadable(t)

	for name, one := range map[string]struct {
		readers port.VaultReaders
		derived port.DerivedStores
		by      port.Proofreader
		want    error
	}{
		"nothing to proofread with": {target.readers, target.derived, nil, run.ErrNothingProofreads},
		"no vault":                  {nil, target.derived, &corrector{}, run.ErrNoVaultToProofread},
		"no store":                  {target.readers, nil, &corrector{}, run.ErrNoStoreToProofread},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProofread(one.readers, one.derived, one.by); !errors.Is(err, one.want) {
				t.Fatalf("made with %v", err)
			}
		})
	}

	// The control: all three, and it is made.
	if _, err := NewProofread(target.readers, target.derived, &corrector{}); err != nil {
		t.Fatalf("a proofreading with everything it needs failed: %v", err)
	}
}

func TestTheCorrectionsGoAdjacentTheRecognitionAndTheRecognitionIsNotTouched(t *testing.T) {
	placed, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	was := readShelf(t, shelved, textNames(t, shelved).artifact)

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pages != 4 || res.Read != 4 || res.Fixed != 2 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	if string(readShelf(t, shelved, names.artifact)) != string(was) {
		t.Error("the recognition was rewritten")
	}
	if placedVal := correction.Unpack(readShelf(t, shelved, names.corrections)); len(placedVal) != 2 {
		t.Errorf("what is kept beside the recognition is %+v", placedVal)
	}
	var status checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, names.far), &status); err != nil {
		t.Fatal(err)
	}
	if status.By != "a proofreader" || status.Pages != 4 {
		t.Errorf("got %+v", status)
	}
}

func TestACorrectedRecognitionIsTheTextTheSourceIsSplitFrom(t *testing.T) {
	placed, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})

	if _, err := placed.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	names := textNames(t, shelved)
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the recognition still says %q", doc.Text)
	}
}

func TestAPageWhoseReplyIsNoAnswerIsLeftAsItWasRead(t *testing.T) {
	placed, v, shelved, delta := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		1: corrects(1, "the WORDS 2"),
	})
	delta.refuses = map[int]bool{0: true}

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.UncorrectedPages != 1 || res.Fixed != 1 {
		t.Errorf("got %+v", res)
	}

	names := textNames(t, shelved)
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the words 1") {
		t.Errorf("a page nothing answered about was changed: %q", doc.Text)
	}
}

func TestARunStoppedPartWayIsTakenUpAtThePageItStoppedOn(t *testing.T) {
	target := newProofreadable(t)
	names := textNames(t, target.shelved)
	status, err := json.Marshal(checkpoint{By: "a proofreader", Pages: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.shelved.WriteArtifact(t.Context(), names.far, status); err != nil {
		t.Fatal(err)
	}
	if err := target.shelved.WriteArtifact(t.Context(), names.corrections, correction.Pack([]domain.CorrectionLine{
		{Number: 0, Text: "the WORDS 1"},
	})); err != nil {
		t.Fatal(err)
	}

	again := &corrector{says: map[int]domain.ProofreadCorrection{2: corrects(2, "the WORDS 3")}}
	res, err := target.put(t, again).Execute(t.Context(), target.vault, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d pages of the recognition", res.Resumed)
	}
	for _, request := range again.asked {
		for _, page := range request {
			if page < 2 {
				t.Errorf("asked about page %d again", page)
			}
		}
	}

	doc, err := text.ReadComposed(t.Context(), target.shelved, "ocr", names.hash, readShelf(t, target.shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []string{"the WORDS 1", "the WORDS 3"} {
		if !strings.Contains(doc.Text, response) {
			t.Errorf("%q is not in %q", response, doc.Text)
		}
	}
}

func TestCorrectionsAnotherProofreaderMadeAreNotTakenUp(t *testing.T) {
	placed, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})
	names := textNames(t, shelved)
	status, err := json.Marshal(checkpoint{By: "somebody else", Pages: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), names.far, status); err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), names.corrections, []byte("what another one said")); err != nil {
		t.Fatal(err)
	}

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 0 {
		t.Errorf("took up %d pages another proofreader had", res.Resumed)
	}
	doc, err := text.ReadComposed(t.Context(), shelved, "ocr", names.hash, readShelf(t, shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Text, "the WORDS 1") {
		t.Errorf("the recognition says %q", doc.Text)
	}
}

func TestOneRunToARecognition(t *testing.T) {
	placed, v, shelved, delta := newProofreadRecognition(t, nil)
	shelved.hold(textNames(t, shelved).corrections)

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsBusy {
		t.Error("two runs to one recognition")
	}
	if len(delta.asked) != 0 {
		t.Errorf("asked about %v", delta.asked)
	}
}

func TestARecognitionThatIsNotThereIsNothingToProofread(t *testing.T) {
	placed, v, shelved, _ := newProofreadRecognition(t, nil)
	names := textNames(t, shelved)
	if err := shelved.RemoveArtifact(t.Context(), names.artifact); err != nil {
		t.Fatal(err)
	}

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNone {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsSplitAgainAsItsPagesAreInsertRight(t *testing.T) {
	placed, v, _, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})
	cuts := 0
	placed.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}

	if _, err := placed.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if cuts != 1 {
		t.Errorf("cut %d times", cuts)
	}
}

func TestABatchThatWasNotSplitIsAskedAboutAgain(t *testing.T) {
	placed, v, shelved, delta := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	cuts := 0
	placed.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		if cuts == 1 {
			return errors.New("the index would not take it")
		}
		return nil
	}

	if _, err := placed.Execute(t.Context(), v, documentPath); err == nil {
		t.Fatal("a cut that failed was taken as a batch done")
	}
	if _, err := shelved.ReadArtifact(t.Context(), textNames(t, shelved).far); err == nil {
		t.Error("a batch nothing cut was counted")
	}

	delta.asked = nil
	if _, err := placed.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if len(delta.asked) != 1 || len(delta.asked[0]) != 4 || delta.asked[0][0] != 0 {
		t.Errorf("the run after it asked about %v", delta.asked)
	}
}

// names are what one recognition of the fixture is kept under.
type names struct {
	hash        string
	artifact    domain.ArtifactName
	corrections domain.ArtifactName
	far         domain.ArtifactName
}

// textNames finds the one recognition on the shelf and says what its files are
// called.
func textNames(t *testing.T, shelved *fakeStorage) names {
	t.Helper()
	for _, name := range shelved.artifactNames() {
		if name.File != domain.ArtifactText || name.Producer != domain.OCR {
			continue
		}
		hash := name.Hash
		return names{
			hash:        hash,
			artifact:    domain.NewArtifactName(domain.ArtifactText, domain.OCR, hash),
			corrections: domain.NewArtifactName(domain.ArtifactCorrections, domain.OCR, hash),
			far:         domain.NewArtifactName(domain.ArtifactProofread, domain.OCR, hash),
		}
	}
	t.Fatal("nothing was read")
	return names{}
}

// getCheckpoint is what the recognition says about who put it right and how far
// they got.
func getCheckpoint(t *testing.T, shelved *fakeStorage) checkpoint {
	t.Helper()
	var status checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, textNames(t, shelved).far), &status); err != nil {
		t.Fatal(err)
	}
	return status
}

// A recognition nothing is left to be asked about is not work, and nothing is told
// about it.
func TestARecognitionAtItsLastPageReportsNoProgress(t *testing.T) {
	target := newProofreadable(t)
	v := target.vault
	first := target.put(t, &corrector{says: map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")}})
	if _, err := first.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	notifyCount := 0
	putResult := target.put(t, &corrector{})
	putResult.OnProgress = func(ProofreadResult) { notifyCount++ }
	res, err := putResult.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Pages {
		t.Fatalf("got %+v", res)
	}
	if notifyCount != 0 {
		t.Errorf("a recognition with nothing left to put right received about %d times", notifyCount)
	}
}

// A proofreading given a known hash skips reading the source file from disk.
func TestAProofreadingGivenAKnownHashDoesNotReadTheSourceFile(t *testing.T) {
	placed, v, _, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
	})
	placed.readers = brokenReaders{}
	placed.Hash = text.Fingerprint(printPages(outline))

	res, err := placed.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatalf("expected execute with known hash to succeed: %v", err)
	}
	if res.Fixed != 1 {
		t.Errorf("expected 1 fixed line, got %+v", res)
	}
}

// A proofreader that cannot answer ends the run, and nothing is written for a
// page it did not answer about.
func TestAProofreaderThatIsUnavailableEndsTheRunAndWritesNothing(t *testing.T) {
	placed, v, shelved, delta := newProofreadRecognition(t, nil)
	delta.fail = fmt.Errorf("%w: no route for the purpose", port.ErrUnavailable)

	_, err := placed.Execute(t.Context(), v, documentPath)

	if !errors.Is(err, port.ErrUnavailable) {
		t.Fatalf("got %v, want the proofreader to be unavailable", err)
	}
	names := textNames(t, shelved)
	if _, err := shelved.ReadArtifact(t.Context(), names.corrections); err == nil {
		t.Error("corrections were written for pages nobody answered about")
	}
	if _, err := shelved.ReadArtifact(t.Context(), names.far); err == nil {
		t.Error("a checkpoint was written for pages nobody answered about")
	}
}

func TestTheCheckpointNamesTheProofreaderAndThePagesCompleted(t *testing.T) {
	placed, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})

	if _, err := placed.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	if got := getCheckpoint(t, shelved); got.By != "a proofreader" || got.Pages != 4 {
		t.Errorf("got %+v", got)
	}
}

func TestOnlyTheCorrectionsOfPagesTheCountClaimsAreCached(t *testing.T) {
	target := newProofreadable(t)
	names := textNames(t, target.shelved)
	store, err := target.derived.Open(target.vault)
	if err != nil {
		t.Fatal(err)
	}
	pages := []domain.ProofreadPage{
		{Number: 0, Lines: []domain.ProofreadLine{{Number: 0}, {Number: 1}}},
		{Number: 1, Lines: []domain.ProofreadLine{{Number: 2}, {Number: 3}}},
	}
	content := []domain.CorrectionLine{{Number: 0, Text: "a"}, {Number: 2, Text: "b"}, {Number: 3, Text: "c"}}
	if err := store.WriteArtifact(t.Context(), names.corrections, correction.Pack(content)); err != nil {
		t.Fatal(err)
	}

	if err := dropCorrections(t.Context(), store, names.corrections, pages, 1); err != nil {
		t.Fatal(err)
	}

	unpacked := correction.Unpack(readShelf(t, target.shelved, names.corrections))
	if len(unpacked) != 1 || unpacked[0].Number != 0 {
		t.Errorf("kept %+v, want the correction of the first page alone", unpacked)
	}
}

func TestProofreadRecognitionRejectsInvalidCorrectionBatches(t *testing.T) {
	put1, v1, shelved1, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: {Number: 0, Last: 1, Text: "joined"},
	})
	if _, err := put1.Execute(t.Context(), v1, documentPath); !errors.Is(err, ErrJoinedCorrectionNotAllowed) {
		t.Errorf("Execute() error = %v, want %v", err, ErrJoinedCorrectionNotAllowed)
	}
	names1 := textNames(t, shelved1)
	if _, err := shelved1.ReadArtifact(t.Context(), names1.corrections); !errors.Is(err, fs.ErrNotExist) {
		t.Error("corrections artifact written for rejected joined batch")
	}

	put2, v2, shelved2, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: {Number: 9999, Text: "unasked"},
	})
	if _, err := put2.Execute(t.Context(), v2, documentPath); !errors.Is(err, ErrUnaskedCorrectionLine) {
		t.Errorf("Execute() error = %v, want %v", err, ErrUnaskedCorrectionLine)
	}
	names2 := textNames(t, shelved2)
	if _, err := shelved2.ReadArtifact(t.Context(), names2.corrections); !errors.Is(err, fs.ErrNotExist) {
		t.Error("corrections artifact written for rejected unasked batch")
	}
}
