package epub_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
)

// guidedOpf puts a guide and a collection before the spine, so the spine is the fifth child of the
// package element, and a comment before the second itemref.
const guidedOpf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Guided</dc:title></metadata>
  <manifest>
    <item id="a" href="one.xhtml" media-type="application/xhtml+xml"/>
    <item id="b" href="two.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <guide><reference type="text" href="one.xhtml"/></guide>
  <collection role="index"/>
  <spine>
    <itemref idref="a"/>
    <!-- the second chapter -->
    <itemref idref="b"/>
  </spine>
</package>`

func readGuidedBook(t *testing.T) *epub.Book {
	t.Helper()
	return read(t, buildSpineArchive(t, guidedOpf, map[string]string{
		"OEBPS/one.xhtml": `<html><head><title>One</title></head><body><h1>Iota</h1><p>
      Alpha  beta <em>gamma</em> delta.</p></body></html>`,
		"OEBPS/two.xhtml": `<html><head><title>Two</title></head><body><p>Lambda<!-- note -->mu &amp; nu.</p></body></html>`,
	}))
}

func TestPlacesFindTheCFIOfACharacterInsideAParagraph(t *testing.T) {
	book := readGuidedBook(t)
	places := book.ReadPlaces()

	for word, want := range map[string]string{
		"Iota":   "epubcfi(/10/2!/4/2/1:0)",
		"Alpha":  "epubcfi(/10/2!/4/4/1:7)",
		"beta":   "epubcfi(/10/2!/4/4/1:14)",
		"gamma":  "epubcfi(/10/2!/4/4/2/1:0)",
		"delta":  "epubcfi(/10/2!/4/4/3:1)",
		"Lambda": "epubcfi(/10/4!/4/2/1:0)",
		"mu":     "epubcfi(/10/4!/4/2/1:6)",
		"nu":     "epubcfi(/10/4!/4/2/1:11)",
	} {
		offset := strings.Index(book.Text, word)
		got, ok := places.GetCFI(offset)
		if !ok || got.String() != want {
			t.Errorf("the CFI of %s is %s (%v), want %s", word, got, ok, want)
		}
		if one, _ := book.GetCFI(offset); one.String() != want {
			t.Errorf("the book names %s by %s, want %s", word, one, want)
		}
	}
}

func TestFindOffsetReturnsTheCharacterACFINames(t *testing.T) {
	book := readGuidedBook(t)

	for cfiText, word := range map[string]string{
		"epubcfi(/10/2!/4/4/1:14)":                  "beta",
		"epubcfi(/10/2!/4/4/1:13)":                  "beta",
		"epubcfi(/10/2!/4/4/2[em]/1:2)":             "mma",
		"epubcfi(/10/2!/4/4/3:1)":                   "delta",
		"epubcfi(/10/2!/4/4)":                       "Alpha",
		"epubcfi(/10/2!/4/4/2)":                     "gamma",
		"epubcfi(/10/4[b]!/4/2/1:9)":                "& nu",
		"epubcfi(/10/4!/4/2,/1:6,/1:8)":             "mu",
		"epubcfi(/10/2!/4/4/1:0)":                   "Alpha",
		"epubcfi(/10/2!/4/4/1:14[ beta^] gam;s=a])": "beta",
	} {
		place, err := cfi.Parse(cfiText)
		if err != nil {
			t.Fatal(err)
		}
		offset, ok := book.FindOffset(place)
		if !ok || !strings.HasPrefix(book.Text[offset:], word) {
			t.Errorf("%s names offset %d (%v), want where %q begins in %q", cfiText, offset, ok, word, book.Text)
		}
	}
}

func TestFindOffsetFindsNoDocumentOutsideTheSpine(t *testing.T) {
	book := readGuidedBook(t)

	for _, cfiText := range []string{"epubcfi(/6/2!/4/2/1:0)", "epubcfi(/10/6!/4/2)"} {
		place, err := cfi.Parse(cfiText)
		if err != nil {
			t.Fatal(err)
		}
		if offset, ok := book.FindOffset(place); ok {
			t.Errorf("%s names offset %d, want no place", cfiText, offset)
		}
	}
}

func TestEachCFIOfTheBookFindsItsOwnOffset(t *testing.T) {
	book := readGuidedBook(t)
	places := book.ReadPlaces()

	for offset, r := range book.Text {
		if r == ' ' || r == '\n' {
			continue
		}
		place, _ := places.GetCFI(offset)
		if got, ok := book.FindOffset(place); !ok || got != offset {
			t.Errorf("%s of offset %d finds %d (%v)", place, offset, got, ok)
		}
	}
}

func TestPlacesListWhereEachBlockBegins(t *testing.T) {
	book := readGuidedBook(t)

	blocks := book.ReadPlaces().Blocks

	var got []string
	for _, offset := range blocks {
		got = append(got, strings.Fields(book.Text[offset:])[0])
	}
	if strings.Join(got, " ") != "Iota Alpha Lambdamu" {
		t.Errorf("blocks begin at %q, want Iota, Alpha and Lambdamu", got)
	}
}
