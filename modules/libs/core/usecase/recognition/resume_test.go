package recognition

import (
	"context"
	"encoding/json"
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const scan = "books/one.pdf"

// newScanArtifact is a recognition of one line to a page, as the artifact and the
// boxes its prose was read from.
func newScanArtifact(lines []string) ([]byte, []byte) {
	pages := make([]ocr.Page, 0, len(lines))
	for item, response := range lines {
		pages = append(pages, ocr.Page{
			Index: item,
			Size:  image.Point{X: 100, Y: 100},
			Blocks: []domain.OCRBlock{{
				Text:  response,
				Boxes: []domain.OCRBox{{Rect: image.Rect(0, 0, 100, 10), Span: domain.ByteSpan{To: len(response)}}},
			}},
		})
	}
	artifact, boxes, _ := ocr.Write(pages)
	return artifact, highlight.Pack(boxes)
}

// newHaltedWorker is a Worker over a vault holding one document,
// with the recognition of it on the shelf and a proofreading of it standing at
// through pages.
func newHaltedWorker(
	t *testing.T,
	delta port.Proofreader,
	paths int,
	lines ...string,
) (*watched, domain.Vault, port.DerivedStore, string) {
	t.Helper()
	raw := []byte("not a document: " + scan)
	shelved := newLibrary()
	shelved.hold(scan, domain.KindBook, raw, 1)
	v := domain.Vault{ID: "v", Path: "/vault"}

	w := newWatched(t, nil)
	w.Worker.with.Readers = vaults{v.ID: shelved}
	w.Worker.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func() (port.Proofreader, error) { return delta, nil },
	}

	hash := text.Fingerprint(raw)
	store, err := w.Worker.with.Derived.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	artifact, boxes := newScanArtifact(lines)
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, "ocr", hash), artifact); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactBoxes, "ocr", hash), boxes); err != nil {
		t.Fatal(err)
	}
	if paths == 0 {
		return w, v, store, hash
	}

	// The shelf as a run that ended among the batches left it: the corrections
	// it wrote, and the count standing after them.
	correctionLines := make([]domain.CorrectionLine, 0, paths)
	for item := range paths {
		correctionLines = append(correctionLines, domain.CorrectionLine{Number: item, Text: correctLine(lines[item])})
	}
	if err := store.AppendArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, "ocr", hash), correction.Pack(correctionLines)); err != nil {
		t.Fatal(err)
	}
	status, err := json.Marshal(checkpoint{By: delta.GetName(), Pages: paths})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, "ocr", hash), status); err != nil {
		t.Fatal(err)
	}
	return w, v, store, hash
}

// books is the document one vault holds, as the index answers for it.
type books struct {
	port.SourceQueries
	hash string
}

func (b books) GetRecognisedSources(
	_ context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]port.SourceText, error) {
	h := b.hash
	if h == "" {
		h = text.Fingerprint([]byte("not a document: " + scan))
	}
	return []port.SourceText{{
		Fingerprint: domain.Fingerprint{Path: scan}, Producer: "ocr", Hash: h,
	}}, nil
}

// correctLine is a line as a proofreader puts it right.
func correctLine(line string) string { return strings.Replace(line, "words", "WORDS", 1) }

// newReply is the correction putting one line right.
func newReply(pos int, line string) domain.ProofreadCorrection {
	return corrects(pos, correctLine(line))
}

// corrections is the lines the shelf holds a correction for.
func corrections(t *testing.T, store port.DerivedStore, name domain.ArtifactName) []int {
	t.Helper()
	raw, err := store.ReadArtifact(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	out := []int{}
	for _, line := range correction.Unpack(raw) {
		out = append(out, line.Number)
	}
	return out
}

// and the pages after it are asked about when the application opens.
func TestARecognitionLeftPartWayThroughIsTakenUpWhenTheApplicationOpens(t *testing.T) {
	lines := []string{"the words one", "the words two", "the words three"}
	delta := &puts{says: map[int]domain.ProofreadCorrection{
		1: newReply(1, lines[1]),
		2: newReply(2, lines[2]),
	}}
	w, v, store, hash := newHaltedWorker(t, delta, 1, lines...)

	w.TakeUp(t.Context(), books{}, v)
	w.Wait()

	if got := delta.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if got := corrections(t, store, domain.NewArtifactName(domain.ArtifactCorrections, "ocr", hash)); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("the corrections stand for lines %v", got)
	}
	if pos, target := w.getTask(t); target {
		t.Errorf("the recognition is left in the list as %+v", pos)
	}
}

// A recognition put right to its last page is asked about nothing and shown
// nowhere, however often the application opens.
func TestARecognitionAlreadyInsertRightIsAskedAboutNothing(t *testing.T) {
	lines := []string{"the words one", "the words two"}
	delta := &puts{}
	w, v, _, _ := newHaltedWorker(t, delta, 2, lines...)

	w.TakeUp(t.Context(), books{}, v)
	w.Wait()

	if got := delta.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if pos, target := w.getTask(t); target {
		t.Errorf("the recognition is in the list as %+v", pos)
	}
}

// One run to a recognition. A run that finds it held leaves the list to the run that
// holds it.
func TestARecognitionAnotherRunContainsKeepsItsPlaceInTheList(t *testing.T) {
	lines := []string{"the words one", "the words two"}
	delta := &puts{}
	w, v, store, hash := newHaltedWorker(t, delta, 1, lines...)
	release, err := store.ClaimArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, "ocr", hash))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	w.tasks.Set(run.Task{
		ID: run.GetProofreadID(scan), Title: "Proofreading a recognition", About: scan,
		Count: 1, Total: 2,
	})

	w.TakeUp(t.Context(), books{}, v)
	w.Wait()

	if pos, target := w.getTask(t); !target || pos.Title != "Proofreading a recognition" {
		t.Errorf("the run holding the recognition is in the list as %+v", pos)
	}
	if got := delta.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// Taking up a proofreading uses the known hash from the index and avoids reading
// the source file from disk.
func TestTakingUpProofreadingDoesNotReadTheSourceFileWhenHashIsKnown(t *testing.T) {
	lines := []string{"the words one", "the words two", "the words three"}
	delta := &puts{says: map[int]domain.ProofreadCorrection{
		1: newReply(1, lines[1]),
		2: newReply(2, lines[2]),
	}}
	w, v, store, hash := newHaltedWorker(t, delta, 1, lines...)
	w.Worker.with.Readers = brokenReaders{}

	w.TakeUp(t.Context(), books{hash: hash}, v)
	w.Wait()

	if got := delta.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if got := corrections(t, store, domain.NewArtifactName(domain.ArtifactCorrections, "ocr", hash)); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("the corrections stand for lines %v", got)
	}
}
