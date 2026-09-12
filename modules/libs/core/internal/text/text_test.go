package text_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// The book is the chapter as the models read it: the titles come out mangled,
// each is its own region.
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
		{Index: 0, Blocks: []ocr.Block{
			{Label: "text", Text: frontMatter},
		}},
		{Index: 1, Blocks: []ocr.Block{
			{Label: "doc_title", Text: docTitle},
			{Label: "text", Text: birth},
		}},
		{Index: 2, Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: sectionOne},
			{Label: "text", Text: ganges},
		}},
		{Index: 3, Blocks: []ocr.Block{
			{Label: "paragraph_title", Text: sectionTwo},
			{Label: "text", Text: padmavati},
		}},
	}
}

// writeRecognition is the artifact a recognition of the book leaves, and the
// parts a layout model named in it.
func writeRecognition(t *testing.T) (raw []byte, parts []byte) {
	t.Helper()
	raw, _, _ = ocr.Write(book())
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

func checkLocation(t *testing.T, doc *text.Document, passage, want string) {
	t.Helper()
	if got := doc.Locate(at(t, doc, passage)); got != want {
		t.Errorf("%q is located at %q, want %q", passage, got, want)
	}
}

func TestAReadingWithPartsLocatesAPassageBySectionAndPage(t *testing.T) {
	raw, parts := writeRecognition(t)
	doc := text.Recognised(raw, parts, nil, nil)

	checkLocation(t, doc, ganges, sectionOne+", page 3 of the file")
	checkLocation(t, doc, padmavati, sectionTwo+", page 4 of the file")
	checkLocation(t, doc, birth, docTitle+", page 2 of the file")
}

func TestAReadingWithPartsNamesThem(t *testing.T) {
	raw, parts := writeRecognition(t)
	doc := text.Recognised(raw, parts, nil, nil)

	want := []chunking.PartStart{
		{Title: docTitle, Offset: at(t, doc, docTitle)},
		{Title: sectionOne, Offset: at(t, doc, sectionOne)},
		{Title: sectionTwo, Offset: at(t, doc, sectionTwo)},
	}
	if !slices.Equal(doc.Parts, want) {
		t.Errorf("the document names %+v, want %+v", doc.Parts, want)
	}
}

func TestAReadingWithNoPartsLocatesAPassageByPageAlone(t *testing.T) {
	raw, _ := writeRecognition(t)
	doc := text.Recognised(raw, nil, nil, nil)

	checkLocation(t, doc, ganges, "page 3 of the file")
	checkLocation(t, doc, padmavati, "page 4 of the file")
	if len(doc.Parts) != 0 {
		t.Errorf("a reading with no parts names %+v", doc.Parts)
	}
}

func TestAPassageBeforeTheFirstPartIsLocatedByPageAlone(t *testing.T) {
	raw, parts := writeRecognition(t)
	doc := text.Recognised(raw, parts, nil, nil)

	checkLocation(t, doc, frontMatter, "page 1 of the file")
}

func TestAPartAtTheVeryStartNamesTheTextFromItsFirstByte(t *testing.T) {
	raw, _, _ := ocr.Write([]ocr.Page{
		{Index: 0, Blocks: []ocr.Block{
			{Label: "doc_title", Text: docTitle},
			{Label: "text", Text: birth},
		}},
	})
	parts := ocr.Pack([]ocr.Part{{Start: 0, Length: len(docTitle), Depth: 0}})
	doc := text.Recognised(raw, parts, nil, nil)

	if doc.Text[:len(docTitle)] != docTitle {
		t.Fatalf("the prose begins %q, want it to begin with the title", doc.Text[:len(docTitle)])
	}
	checkLocation(t, doc, docTitle, docTitle+", page 1 of the file")
	checkLocation(t, doc, birth, docTitle+", page 1 of the file")
}

func TestPartsOutOfOrderAreNotTrusted(t *testing.T) {
	raw, _ := writeRecognition(t)
	prose, _ := ocr.Read(raw)
	parts := ocr.Pack([]ocr.Part{
		{Start: strings.Index(prose, sectionTwo), Length: len(sectionTwo), Depth: 1},
		{Start: strings.Index(prose, sectionOne), Length: len(sectionOne), Depth: 1},
	})
	doc := text.Recognised(raw, parts, nil, nil)

	if len(doc.Parts) != 0 {
		t.Errorf("the document names %+v", doc.Parts)
	}
	checkLocation(t, doc, ganges, "page 3 of the file")
}

func TestPartsNamingOffsetsPastTheTextAreNotTrusted(t *testing.T) {
	raw, _ := writeRecognition(t)
	prose, _ := ocr.Read(raw)
	parts := ocr.Pack([]ocr.Part{
		{Start: strings.Index(prose, sectionOne), Length: len(sectionOne), Depth: 1},
		{Start: len(prose) - 4, Length: 100, Depth: 1},
	})
	doc := text.Recognised(raw, parts, nil, nil)

	if len(doc.Parts) != 0 {
		t.Errorf("the document names %+v", doc.Parts)
	}
	checkLocation(t, doc, ganges, "page 3 of the file")
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

func TestAPageIsSaidByWhereItStandsInTheFile(t *testing.T) {
	// One name to a page, and it is the one a viewer opens at. What the paper
	// printed is a second number for the same page, and a person shown both has
	// to work out which is being talked about.
	raw, _, _ := ocr.Write([]ocr.Page{
		{Index: 0, Blocks: []ocr.Block{{Label: "text", Text: "Alpha beta."}}},
		{Index: 1, Blocks: []ocr.Block{{Label: "text", Text: "Gamma delta."}}},
	})
	doc := text.Recognised(raw, nil, nil, nil)

	if said := doc.Locate(0); said != "page 1 of the file" {
		t.Errorf("the first page is located at %q", said)
	}
	if said := doc.Locate(len(doc.Text) - 3); said != "page 2 of the file" {
		t.Errorf("the second page is located at %q", said)
	}
}
