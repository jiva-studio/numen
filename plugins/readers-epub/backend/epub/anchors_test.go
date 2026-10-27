package epub_test

import (
	"strings"
	"testing"
)

func TestAnchorsNameEachBlockOfTheBookByItsCFI(t *testing.T) {
	book := read(t, buildSpineArchive(t, laidOutOpf, map[string]string{
		"OEBPS/one.xhtml": `<html><head><title>One</title></head><body><h1>Iota</h1><p>Kappa.</p></body></html>`,
		"OEBPS/two.xhtml": `<html><head><title>Two</title></head><body><p>Lambda.</p></body></html>`,
	}))

	anchors := book.Anchors()

	got := map[string]int{}
	for _, one := range anchors {
		got[one.CFI] = one.Offset
	}
	for cfi, word := range map[string]string{
		"epubcfi(/6/2!/4/2)": "Iota",
		"epubcfi(/6/2!/4/4)": "Kappa",
		"epubcfi(/6/4!/4/2)": "Lambda",
	} {
		offset, isNamed := got[cfi]
		if !isNamed || !strings.HasPrefix(book.Text[offset:], word) {
			t.Errorf("%s is at offset %d (%v), want where %s begins in %q", cfi, offset, isNamed, word, book.Text)
		}
	}
	for i := 1; i < len(anchors); i++ {
		if anchors[i].Offset < anchors[i-1].Offset {
			t.Errorf("anchors %v do not ascend by offset", anchors)
		}
	}
}
