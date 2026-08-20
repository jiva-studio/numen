package text_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// The book is the chapter as the models read it: the titles come out mangled,
// each is its own region, and the page numbers are the printed ones.
const (
	frontMatter = "This book was printed at Nabadwip in the year 1908."
	docTitle    = "Srī Javadeva Gosvāmi"
	birth       = "The poet was born in the village of Kenduli."
	sectionOne  = "IAYADEVA GOSVAMI'S LIFEINNABADWI"
	ganges      = "He came to Nabadwip and lived beside the Ganges."
	sectionTwo  = "LAYADEVA GOSVAMI'S MARRIAGE TO PADMAVAT"
	padmavati   = "Padmavati was given to him by her father."
)

func book() []ocr.Page {
	return []ocr.Page{
		{At: 0, Label: "iii", Blocks: []ocr.Block{
			{Label: "text", Text: frontMatter},
		}},
		{At: 1, Label: "iv", Blocks: []ocr.Block{
			{Label: "doc_title", Text: docTitle},
			{Label: "text", Text: birth},
		}},
		{At: 2, Label: "v", Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: sectionOne},
			{Label: "text", Text: ganges},
		}},
		{At: 3, Label: "vi", Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: sectionTwo},
			{Label: "text", Text: padmavati},
		}},
	}
}

// written is the artifact a recognition of the book leaves, and the parts a
// layout model named in it.
func written(t *testing.T) (raw []byte, parts []byte) {
	t.Helper()
	raw, _ = ocr.Write(book())
	prose, _ := ocr.Read(raw)
	named := []struct {
		heading string
		depth   int
	}{{docTitle, 0}, {sectionOne, 1}, {sectionTwo, 1}}

	var found []ocr.Part
	for _, n := range named {
		at := strings.Index(prose, n.heading)
		if at < 0 {
			t.Fatalf("the artifact does not carry the heading %q", n.heading)
		}
		found = append(found, ocr.Part{Start: at, Length: len(n.heading), Depth: n.depth})
	}
	return raw, ocr.Pack(found)
}

// at is where a passage begins in a document's text.
func at(t *testing.T, doc *text.Document, passage string) int {
	t.Helper()
	offset := strings.Index(doc.Text, passage)
	if offset < 0 {
		t.Fatalf("the document does not say %q", passage)
	}
	return offset
}

func located(t *testing.T, doc *text.Document, passage, want string) {
	t.Helper()
	if got := doc.Locate(at(t, doc, passage)); got != want {
		t.Errorf("%q is located at %q, want %q", passage, got, want)
	}
}

func TestAReadingWithPartsLocatesAPassageBySectionAndPage(t *testing.T) {
	raw, parts := written(t)
	doc := text.Recognised(raw, parts)

	located(t, doc, ganges, sectionOne+", v")
	located(t, doc, padmavati, sectionTwo+", vi")
	located(t, doc, birth, docTitle+", iv")
}

func TestAReadingWithPartsNamesThemAsPlaces(t *testing.T) {
	raw, parts := written(t)
	doc := text.Recognised(raw, parts)

	want := []window.Place{
		{Title: docTitle, Offset: at(t, doc, docTitle)},
		{Title: sectionOne, Offset: at(t, doc, sectionOne)},
		{Title: sectionTwo, Offset: at(t, doc, sectionTwo)},
	}
	if !slices.Equal(doc.Places, want) {
		t.Errorf("the document names %+v, want %+v", doc.Places, want)
	}
}

func TestAReadingWithNoPartsLocatesAPassageByPageAlone(t *testing.T) {
	raw, _ := written(t)
	doc := text.Recognised(raw, nil)

	located(t, doc, ganges, "v")
	located(t, doc, padmavati, "vi")
	if len(doc.Places) != 0 {
		t.Errorf("a reading with no parts names %+v", doc.Places)
	}
}

func TestAPassageBeforeTheFirstPartIsLocatedByPageAlone(t *testing.T) {
	raw, parts := written(t)
	doc := text.Recognised(raw, parts)

	located(t, doc, frontMatter, "iii")
}

func TestAPartAtTheVeryStartNamesTheTextFromItsFirstByte(t *testing.T) {
	raw, _ := ocr.Write([]ocr.Page{
		{At: 0, Label: "1", Blocks: []ocr.Block{
			{Label: "doc_title", Text: docTitle},
			{Label: "text", Text: birth},
		}},
	})
	parts := ocr.Pack([]ocr.Part{{Start: 0, Length: len(docTitle), Depth: 0}})
	doc := text.Recognised(raw, parts)

	if doc.Text[:len(docTitle)] != docTitle {
		t.Fatalf("the prose begins %q, want it to begin with the title", doc.Text[:len(docTitle)])
	}
	located(t, doc, docTitle, docTitle+", 1")
	located(t, doc, birth, docTitle+", 1")
}

func TestPartsOutOfOrderAreNotTrusted(t *testing.T) {
	raw, _ := written(t)
	prose, _ := ocr.Read(raw)
	parts := ocr.Pack([]ocr.Part{
		{Start: strings.Index(prose, sectionTwo), Length: len(sectionTwo), Depth: 1},
		{Start: strings.Index(prose, sectionOne), Length: len(sectionOne), Depth: 1},
	})
	doc := text.Recognised(raw, parts)

	if len(doc.Places) != 0 {
		t.Errorf("the document names %+v", doc.Places)
	}
	located(t, doc, ganges, "v")
}

func TestPartsNamingOffsetsPastTheTextAreNotTrusted(t *testing.T) {
	raw, _ := written(t)
	prose, _ := ocr.Read(raw)
	parts := ocr.Pack([]ocr.Part{
		{Start: strings.Index(prose, sectionOne), Length: len(sectionOne), Depth: 1},
		{Start: len(prose) - 4, Length: 100, Depth: 1},
	})
	doc := text.Recognised(raw, parts)

	if len(doc.Places) != 0 {
		t.Errorf("the document names %+v", doc.Places)
	}
	located(t, doc, ganges, "v")
}

func TestAPartsSidecarIsSweptWithTheRest(t *testing.T) {
	name := text.Parts("ocr", "abc123")
	if name != "ocr/abc123.parts" {
		t.Errorf("parts are kept under %q", name)
	}
	if !slices.Contains(text.Names("ocr", "abc123"), name) {
		t.Errorf("a sweep of %v leaves the parts behind", text.Names("ocr", "abc123"))
	}
}

func TestAPageIsSaidByWhatItPrintsOrByWhereItStands(t *testing.T) {
	// A page that prints its number is called that. A page that prints none is
	// called by where it stands in the file, and said so — a person told a
	// number looks for it on the page, and the position is not there.
	raw, _ := ocr.Write([]ocr.Page{
		{At: 0, Number: "2", Blocks: []ocr.Block{{Label: "text", Text: "Alpha beta."}}},
		{At: 1, Blocks: []ocr.Block{{Label: "text", Text: "Gamma delta."}}},
	})
	doc := text.Recognised(raw, nil)

	if said := doc.Locate(0); said != "2" {
		t.Errorf("a page printing 2 is located at %q", said)
	}
	if said := doc.Locate(len(doc.Text) - 3); said != "page 2 of the file" {
		t.Errorf("a page printing nothing is located at %q", said)
	}
}
