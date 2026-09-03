package source

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/fixes"
	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

const scan = "books/one.pdf"

// printed is a reading of one line to a page, as the artifact and the boxes its
// prose was read from.
func printed(lines []string) ([]byte, []byte) {
	pages := make([]ocr.Page, 0, len(lines))
	for at, said := range lines {
		pages = append(pages, ocr.Page{
			At:   at,
			Size: image.Point{X: 100, Y: 100},
			Blocks: []ocr.Block{{
				Text:  said,
				Spans: []ocr.Span{{Box: image.Rect(0, 0, 100, 10), Length: len(said)}},
			}},
		})
	}
	artifact, boxes, _ := ocr.Write(pages)
	return artifact, lit.Pack(boxes)
}

// halted is a Recognising over a vault holding one document, with the reading of
// it on the shelf and a proofreading of it standing at through pages.
func halted(
	t *testing.T,
	by port.Proofreader,
	through int,
	lines ...string,
) (*watched, domain.Vault, port.DerivedStore, string) {
	t.Helper()
	root := t.TempDir()
	at := filepath.Join(root, filepath.FromSlash(scan))
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte("not a document: " + scan)
	if err := os.WriteFile(at, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	v := domain.Vault{ID: "v", Path: root}

	w := recognising(t, nil)
	w.Recognising.with.Proofreading = Correcting{
		Named: true, Automatically: true, Batch: 1,
		By:    func(string) (port.Proofreader, error) { return by, nil },
		Queue: func(string) (port.ProofreadQueue, error) { return nil, nil },
	}

	hash := text.Fingerprint(raw)
	store, err := derivedStores.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	artifact, boxes := printed(lines)
	if err := store.Write(t.Context(), text.Artifact("ocr", hash), artifact); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(t.Context(), text.Boxes("ocr", hash), boxes); err != nil {
		t.Fatal(err)
	}
	if through == 0 {
		return w, v, store, hash
	}

	// The shelf as a run that ended among the batches left it: the corrections
	// it wrote, and the count standing after them.
	put := make([]fixes.Line, 0, through)
	for at := range through {
		put = append(put, fixes.Line{At: at, Text: corrected(lines[at])})
	}
	if err := store.Append(t.Context(), text.Fixes("ocr", hash), fixes.Pack(put)); err != nil {
		t.Fatal(err)
	}
	stood, err := json.Marshal(standing{By: by.Name(), Pages: through})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Write(t.Context(), text.Proofread("ocr", hash), stood); err != nil {
		t.Fatal(err)
	}
	return w, v, store, hash
}

// books is the document one vault holds, as the index answers for it.
type books struct{ port.SourceQueries }

func (books) Recognised(
	_ context.Context, _ string, _ domain.SourceKind,
) ([]port.Recognised, error) {
	return []port.Recognised{{Path: scan, From: "ocr", Hash: "x"}}, nil
}

// corrected is a line as a proofreader puts it right.
func corrected(line string) string { return strings.Replace(line, "words", "WORDS", 1) }

// numbered is a reply putting one line right.
func numbered(at int, line string) string {
	return fmt.Sprintf("%d|%s", at, corrected(line))
}

// corrections is the lines the shelf holds a correction for.
func corrections(t *testing.T, store port.DerivedStore, name string) []int {
	t.Helper()
	raw, err := store.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	out := []int{}
	for _, line := range fixes.Unpack(raw) {
		out = append(out, line.At)
	}
	return out
}

// A proofreading that ended among the batches stands at the page it reached,
// and the pages after it are asked about when the application opens.
func TestAReadingLeftPartWayThroughIsTakenUpWhenTheApplicationOpens(t *testing.T) {
	lines := []string{"the words one", "the words two", "the words three"}
	by := &puts{says: map[int]string{
		1: numbered(1, lines[1]),
		2: numbered(2, lines[2]),
	}}
	w, v, store, hash := halted(t, by, 1, lines...)

	w.TakingUp(t.Context(), books{}, v)
	w.Wait()

	if got := by.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if got := corrections(t, store, text.Fixes("ocr", hash)); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("the corrections stand for lines %v", got)
	}
	if at, held := w.said(t); held {
		t.Errorf("the reading is left in the list as %+v", at)
	}
}

// A reading put right to its last page is asked about nothing and shown
// nowhere, however often the application opens.
func TestAReadingAlreadyPutRightIsAskedAboutNothing(t *testing.T) {
	lines := []string{"the words one", "the words two"}
	by := &puts{}
	w, v, _, _ := halted(t, by, 2, lines...)

	w.TakingUp(t.Context(), books{}, v)
	w.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if at, held := w.said(t); held {
		t.Errorf("the reading is in the list as %+v", at)
	}
}

// A proofreader with a queue leaves a batch behind it, and the run that comes
// back for it is the one that collects it.
func TestAReadingWithABatchOutIsLeftToTheCollection(t *testing.T) {
	lines := []string{"the words one", "the words two"}
	by := &puts{says: map[int]string{1: numbered(1, lines[1])}}
	w, v, _, _ := halted(t, by, 1, lines...)
	w.Recognising.with.Proofreading.Queue = func(string) (port.ProofreadQueue, error) { return leaves{}, nil }

	w.TakingUp(t.Context(), books{}, v)
	w.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// leaves is a proofreader that takes the pages away and answers about them
// later.
type leaves struct{}

func (leaves) Name() string { return "a queue" }

func (leaves) Read(context.Context, []proofread.Batch) (map[int]string, error) { return nil, nil }

func (leaves) Leave(context.Context, []proofread.Batch) (string, error) { return "a batch", nil }

func (leaves) Collect(context.Context, string) (map[int]string, bool, error) {
	return nil, false, nil
}

// One run to a reading. A run that finds it held leaves the list to the run that
// holds it.
func TestAReadingAnotherRunHoldsKeepsItsPlaceInTheList(t *testing.T) {
	lines := []string{"the words one", "the words two"}
	by := &puts{}
	w, v, store, hash := halted(t, by, 1, lines...)
	release, err := store.Claim(t.Context(), text.Fixes("ocr", hash))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	w.tasks.Set(task.Task{
		ID: correctingLine(scan), Doing: "Proofreading a reading", About: scan,
		Done: 1, Total: 2,
	})

	w.TakingUp(t.Context(), books{}, v)
	w.Wait()

	if at, held := w.said(t); !held || at.Doing != "Proofreading a reading" {
		t.Errorf("the run holding the reading is in the list as %+v", at)
	}
	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}
