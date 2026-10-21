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
	shelved *shelf
}

// put is a proofreading of that recognition, by the proofreader given.
func (p proofreadable) put(t *testing.T, by port.Proofreader) Proofread {
	t.Helper()
	made, err := NewProofread(p.readers, p.derived, by)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// newProofreadRecognition is that recognition, a proofreading of it, and the
// proofreader the proofreading was made with.
func newProofreadRecognition(t *testing.T, says map[int]domain.ProofreadCorrection) (Proofread, domain.Vault, *shelf, *corrector) {
	t.Helper()
	held := newProofreadable(t)
	by := &corrector{says: says}
	return held.put(t, by), held.vault, held.shelved, by
}

// newProofreadable is the fixture's recognition, written down and ready to be put
// right.
func newProofreadable(t *testing.T) proofreadable {
	t.Helper()
	read, v, _, shelved, _ := createRecognise(t, "the words", outline)
	if _, err := read.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	return proofreadable{readers: read.readers, derived: shelves{shelved}, vault: v, shelved: shelved}
}

// A proofreading with nothing to proofread with is refused where it would be
// made, so no such thing exists to be called. An installation that named no
// profile has no proofreader at all, so this is a value the settings produce
// and not a caller's slip.
func TestNoProofreadingIsMadeWithNothingToProofreadWith(t *testing.T) {
	held := newProofreadable(t)

	for name, one := range map[string]struct {
		readers port.VaultReaders
		derived port.DerivedStores
		by      port.Proofreader
		want    error
	}{
		"nothing to proofread with": {held.readers, held.derived, nil, run.ErrNothingProofreads},
		"no vault":                  {nil, held.derived, &corrector{}, run.ErrNoVaultToProofread},
		"no store":                  {held.readers, nil, &corrector{}, run.ErrNoStoreToProofread},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProofread(one.readers, one.derived, one.by); !errors.Is(err, one.want) {
				t.Fatalf("made with %v", err)
			}
		})
	}

	// The control: all three, and it is made.
	if _, err := NewProofread(held.readers, held.derived, &corrector{}); err != nil {
		t.Fatalf("a proofreading with everything it needs was refused: %v", err)
	}
}

func TestTheCorrectionsGoBesideTheRecognitionAndTheRecognitionIsNotTouched(t *testing.T) {
	put, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	was := readShelf(t, shelved, textNames(t, shelved).artifact)

	res, err := put.Execute(t.Context(), v, documentPath)
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
	if put := correction.Unpack(readShelf(t, shelved, names.corrections)); len(put) != 2 {
		t.Errorf("what is kept beside the recognition is %+v", put)
	}
	var stood checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, names.far), &stood); err != nil {
		t.Fatal(err)
	}
	if stood.By != "a proofreader" || stood.Pages != 4 {
		t.Errorf("got %+v", stood)
	}
}

func TestACorrectedRecognitionIsTheTextTheSourceIsCutFrom(t *testing.T) {
	put, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
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
	put, v, shelved, by := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		1: corrects(1, "the WORDS 2"),
	})
	by.refuses = map[int]bool{0: true}

	res, err := put.Execute(t.Context(), v, documentPath)
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
	held := newProofreadable(t)
	names := textNames(t, held.shelved)
	stood, err := json.Marshal(checkpoint{By: "a proofreader", Pages: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := held.shelved.WriteArtifact(t.Context(), names.far, stood); err != nil {
		t.Fatal(err)
	}
	if err := held.shelved.WriteArtifact(t.Context(), names.corrections, correction.Pack([]domain.CorrectionLine{
		{Number: 0, Text: "the WORDS 1"},
	})); err != nil {
		t.Fatal(err)
	}

	again := &corrector{says: map[int]domain.ProofreadCorrection{2: corrects(2, "the WORDS 3")}}
	res, err := held.put(t, again).Execute(t.Context(), held.vault, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed != 2 {
		t.Errorf("took up %d pages of the recognition", res.Resumed)
	}
	for _, asked := range again.asked {
		for _, page := range asked {
			if page < 2 {
				t.Errorf("asked about page %d again", page)
			}
		}
	}

	doc, err := text.ReadComposed(t.Context(), held.shelved, "ocr", names.hash, readShelf(t, held.shelved, names.artifact))
	if err != nil {
		t.Fatal(err)
	}
	for _, said := range []string{"the WORDS 1", "the WORDS 3"} {
		if !strings.Contains(doc.Text, said) {
			t.Errorf("%q is not in %q", said, doc.Text)
		}
	}
}

func TestCorrectionsAnotherProofreaderMadeAreNotTakenUp(t *testing.T) {
	put, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})
	names := textNames(t, shelved)
	stood, err := json.Marshal(checkpoint{By: "somebody else", Pages: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), names.far, stood); err != nil {
		t.Fatal(err)
	}
	if err := shelved.WriteArtifact(t.Context(), names.corrections, []byte("what another one said")); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
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
	put, v, shelved, by := newProofreadRecognition(t, nil)
	shelved.hold(textNames(t, shelved).corrections)

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsBusy {
		t.Error("two runs to one recognition")
	}
	if len(by.asked) != 0 {
		t.Errorf("asked about %v", by.asked)
	}
}

func TestARecognitionThatIsNotThereIsNothingToProofread(t *testing.T) {
	put, v, shelved, _ := newProofreadRecognition(t, nil)
	names := textNames(t, shelved)
	if err := shelved.RemoveArtifact(t.Context(), names.artifact); err != nil {
		t.Fatal(err)
	}

	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNone {
		t.Errorf("got %+v", res)
	}
}

func TestASourceIsCutAgainAsItsPagesArePutRight(t *testing.T) {
	put, v, _, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})
	cuts := 0
	put.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		return nil
	}

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if cuts != 1 {
		t.Errorf("cut %d times", cuts)
	}
}

func TestABatchThatWasNotCutIsAskedAboutAgain(t *testing.T) {
	put, v, shelved, by := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
		2: corrects(2, "the WORDS 3"),
	})
	cuts := 0
	put.Cut = func(context.Context, domain.Vault, string) error {
		cuts++
		if cuts == 1 {
			return errors.New("the index would not take it")
		}
		return nil
	}

	if _, err := put.Execute(t.Context(), v, documentPath); err == nil {
		t.Fatal("a cut that failed was taken as a batch done")
	}
	if _, err := shelved.ReadArtifact(t.Context(), textNames(t, shelved).far); err == nil {
		t.Error("a batch nothing cut was counted")
	}

	by.asked = nil
	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}
	if len(by.asked) != 1 || len(by.asked[0]) != 4 || by.asked[0][0] != 0 {
		t.Errorf("the run after it asked about %v", by.asked)
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
func textNames(t *testing.T, shelved *shelf) names {
	t.Helper()
	for _, name := range shelved.names() {
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
func getCheckpoint(t *testing.T, shelved *shelf) checkpoint {
	t.Helper()
	var stood checkpoint
	if err := json.Unmarshal(readShelf(t, shelved, textNames(t, shelved).far), &stood); err != nil {
		t.Fatal(err)
	}
	return stood
}

// A recognition nothing is left to be asked about is not work, and nothing is told
// about it.
func TestARecognitionAtItsLastPageReportsNoProgress(t *testing.T) {
	held := newProofreadable(t)
	v := held.vault
	first := held.put(t, &corrector{says: map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")}})
	if _, err := first.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	told := 0
	put := held.put(t, &corrector{})
	put.OnProgress = func(ProofreadResult) { told++ }
	res, err := put.Execute(t.Context(), v, documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.Read != res.Pages {
		t.Fatalf("got %+v", res)
	}
	if told != 0 {
		t.Errorf("a recognition with nothing left to put right was told about %d times", told)
	}
}

// A proofreading given a known hash skips reading the source file from disk.
func TestAProofreadingGivenAKnownHashDoesNotReadTheSourceFile(t *testing.T) {
	put, v, _, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{
		0: corrects(0, "the WORDS 1"),
	})
	put.readers = brokenReaders{}
	put.Hash = text.Fingerprint(printPages(outline))

	res, err := put.Execute(t.Context(), v, documentPath)
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
	put, v, shelved, by := newProofreadRecognition(t, nil)
	by.fail = fmt.Errorf("%w: no route for the purpose", port.ErrUnavailable)

	_, err := put.Execute(t.Context(), v, documentPath)

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

func TestTheCheckpointNamesTheProofreaderAndThePagesDone(t *testing.T) {
	put, v, shelved, _ := newProofreadRecognition(t, map[int]domain.ProofreadCorrection{0: corrects(0, "the WORDS 1")})

	if _, err := put.Execute(t.Context(), v, documentPath); err != nil {
		t.Fatal(err)
	}

	if got := getCheckpoint(t, shelved); got.By != "a proofreader" || got.Pages != 4 {
		t.Errorf("got %+v", got)
	}
}

func TestOnlyTheCorrectionsOfPagesTheCountClaimsAreKept(t *testing.T) {
	held := newProofreadable(t)
	names := textNames(t, held.shelved)
	store, err := held.derived.Open(held.vault)
	if err != nil {
		t.Fatal(err)
	}
	pages := []domain.ProofreadPage{
		{Number: 0, Lines: []domain.ProofreadLine{{Number: 0}, {Number: 1}}},
		{Number: 1, Lines: []domain.ProofreadLine{{Number: 2}, {Number: 3}}},
	}
	written := []domain.CorrectionLine{{Number: 0, Text: "a"}, {Number: 2, Text: "b"}, {Number: 3, Text: "c"}}
	if err := store.WriteArtifact(t.Context(), names.corrections, correction.Pack(written)); err != nil {
		t.Fatal(err)
	}

	if err := dropCorrections(t.Context(), store, names.corrections, pages, 1); err != nil {
		t.Fatal(err)
	}

	kept := correction.Unpack(readShelf(t, held.shelved, names.corrections))
	if len(kept) != 1 || kept[0].Number != 0 {
		t.Errorf("kept %+v, want the correction of the first page alone", kept)
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
