package source

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

const ocrPlugin = "test.ocr"

// positioned is one word a page, each in a rectangle of its own.
func positioned(words ...string) []domain.TextPart {
	parts := make([]domain.TextPart, 0, len(words))
	for i, word := range words {
		parts = append(parts, domain.TextPart{
			Text:  word + " ",
			Place: domain.TextPlace{Page: i, Rects: []domain.Rect{{MinX: float32(i), MaxX: float32(i) + 0.5, MaxY: 1}}},
		})
	}
	return parts
}

// newHighlight is a Highlight over one vault holding one scanned document, the
// text versions of it, and the vault itself.
func newHighlight(t *testing.T) (Highlight, *store, *shelfTexts, *library) {
	t.Helper()
	shelved := newLibrary()
	shelved.hold(documentPath, domain.KindBook, []byte("%PDF the first printing"), 1)
	index := newStore()
	shelves := vaults{first.ID: shelved}
	texts := newShelfTexts(shelves)
	return NewHighlight(shelves, index, texts), index, texts, shelved
}

// recordCut records the source the way the index holds one cut from a version.
func recordCut(t *testing.T, index *store, shelved *library, version domain.TextVersion) {
	t.Helper()
	ref, err := shelved.GetFingerprint(t.Context(), documentPath)
	if err != nil {
		t.Fatal(err)
	}
	index.put(first.ID, domain.Source{Fingerprint: ref, Hash: version.InputHash, Producer: string(version.GetID())})
}

func spanOf(t *testing.T, text, word string) domain.ByteSpan {
	t.Helper()
	pos := strings.Index(text, word)
	if pos < 0 {
		t.Fatalf("the text does not say %q", word)
	}
	return domain.ByteSpan{From: pos, To: pos + len(word)}
}

func getPages(boxes []domain.HighlightBox) []int {
	var out []int
	for _, box := range boxes {
		if !slices.Contains(out, box.Page) {
			out = append(out, box.Page)
		}
	}
	return out
}

func TestARunIsLitOnThePagesOfTheVersionItWasCutFrom(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	version := texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana", "vrksa", "bija"))
	recordCut(t, index, shelved, version)
	text := domain.NewVersionText(version, positioned("udyana", "vrksa", "bija")).Text

	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{
		spanOf(t, text, "vrksa"),
		spanOf(t, text, "a vrksa b"),
		spanOf(t, text, "udyana"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Fatalf("%d runs answered, want 3", len(runs))
	}
	if runs[0].Text != "vrksa" || !slices.Equal(getPages(runs[0].Boxes), []int{1}) {
		t.Errorf("vrksa says %q and is lit on %v, want page 1", runs[0].Text, getPages(runs[0].Boxes))
	}
	if !slices.Equal(getPages(runs[1].Boxes), []int{0, 1, 2}) {
		t.Errorf("a run across three words is lit on %v", getPages(runs[1].Boxes))
	}
	if !slices.Equal(getPages(runs[2].Boxes), []int{0}) {
		t.Errorf("the first word is lit on %v, want page 0", getPages(runs[2].Boxes))
	}
}

// The runs are places in the version the index cut, and a newer version of the
// same bytes has other words at those places.
func TestARunIsReadFromTheVersionTheIndexCutAndNotTheNewestOne(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	cut := texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana", "vrksa"))
	recordCut(t, index, shelved, cut)
	texts.put(first, documentPath, "test.proofreader", domain.TextFormatPositioned, positioned("jala", "grama"))

	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: 0, To: 6}})
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].Text != "udyana" {
		t.Fatalf("the run says %q, want the word of the version that was cut", runs[0].Text)
	}
}

func TestAFileRewrittenSinceItWasCutIsLitFromItsLiveVersion(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	recordCut(t, index, shelved, texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana")))

	shelved.hold(documentPath, domain.KindBook, []byte("%PDF the second printing"), 2)
	texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("aqua", "terra"))

	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: 5, To: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].Text != "terra" || !slices.Equal(getPages(runs[0].Boxes), []int{1}) {
		t.Fatalf("the run says %q on %v, want terra on page 1", runs[0].Text, getPages(runs[0].Boxes))
	}
}

func TestPlainTextIsReadAndLitNowhere(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	recordCut(t, index, shelved, texts.put(first, documentPath, "test.article", domain.TextFormatPlain,
		[]domain.TextPart{{Text: "udyana vrksa"}}))
	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: 7, To: 12}})
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].Text != "vrksa" || len(runs[0].Boxes) != 0 {
		t.Fatalf("got %+v, want the word and no boxes", runs[0])
	}
}

func TestASourceTheIndexDoesNotHoldIsLitNowhere(t *testing.T) {
	hl, _, texts, _ := newHighlight(t)
	texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana"))
	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: 0, To: 6}})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Text != "" || len(runs[0].Boxes) != 0 {
		t.Fatalf("got %+v, want one run lit nowhere", runs)
	}
}

func TestASourceWhoseVersionIsGoneIsLitNowhere(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	recordCut(t, index, shelved, texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana")))
	texts.drop(documentPath)
	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: 0, To: 6}})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].Boxes) != 0 {
		t.Fatalf("got %+v, want one run lit nowhere", runs)
	}
}

func TestARunOutsideTheTextReadsWhatIsThere(t *testing.T) {
	hl, index, texts, shelved := newHighlight(t)
	recordCut(t, index, shelved, texts.put(first, documentPath, ocrPlugin, domain.TextFormatPositioned, positioned("udyana")))
	runs, err := hl.Execute(t.Context(), first, documentPath, []domain.ByteSpan{{From: -3, To: 3}, {From: 4, To: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].Text != "udyana" {
		t.Errorf("a run beginning before the text says %q, want it read from the start", runs[0].Text)
	}
	if runs[1].Text != "" || len(runs[1].Boxes) != 0 {
		t.Errorf("a run of negative length is %+v", runs[1])
	}
}

func TestAPathTheVaultDoesNotHoldIsRefused(t *testing.T) {
	hl, _, _, _ := newHighlight(t)
	if _, err := hl.Execute(t.Context(), first, "library/absent.pdf", []domain.ByteSpan{{From: 0, To: 1}}); err == nil {
		t.Fatal("a path the vault does not hold was answered")
	}
}
