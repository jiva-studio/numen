package editor

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// A file of the vault is one part of its address, and what is asked of it is
// the rest.
//
// The path is written out whole, so a file standing in a folder named for a
// page is still one part. Read as the parts arrived, `pages/pages/x.pdf` and
// the page asked of it are the same three words twice over. A place in a book
// is as many parts as the archive names it with.
func TestAFileIsOnePartOfItsAddress(t *testing.T) {
	for _, one := range []struct {
		what  string
		url   string
		path  string
		where string
	}{
		{"a file", assetOf("library/a.pdf"), "library/a.pdf", ""},
		{"a page of it", pageOf("library/a.pdf", 3, 800, fingerprint{}),
			"library/a.pdf", "pages/3"},
		{"a file in a folder named for a page", pageOf("pages/pages/x.pdf", 2, 400, fingerprint{}),
			"pages/pages/x.pdf", "pages/2"},
		{"a picture of a book", pictureOf("library/a.epub", "OEBPS/pictures/plate.png", fingerprint{}),
			"library/a.epub", "OEBPS/pictures/plate.png"},
		{"a name with a space in it", assetOf("library/A Book.pdf"), "library/A Book.pdf", ""},
	} {
		t.Run(one.what, func(t *testing.T) {
			got, ok := parseAssetAddress(httptest.NewRequest("GET", one.url, nil))
			if !ok {
				t.Fatalf("%s is not an address", one.url)
			}
			if got.path != one.path || got.where != one.where {
				t.Errorf("%s reads as %q/%q, want %q/%q",
					one.url, got.path, got.where, one.path, one.where)
			}
		})
	}
}

func TestWhatIsNotAnAssetIsNotAnAddress(t *testing.T) {
	for _, url := range []string{"/assets/", "/assets", "/elsewhere/a.pdf"} {
		if _, ok := parseAssetAddress(httptest.NewRequest("GET", url, nil)); ok {
			t.Errorf("%s was read as an address", url)
		}
	}
}

// A file no reader reads has no places, and a file is asked with a place in it.
func TestAnAssetNoReaderReadsIsNotAnswered(t *testing.T) {
	handler := (&API{}).NewHandler(http.NotFoundHandler())

	for _, one := range []struct {
		what string
		url  string
		want int
	}{
		{"a file with no place named", assetOf("library/a.pdf"), http.StatusBadRequest},
		{"a file of no reader", pageOf("library/a.md", 0, 800, fingerprint{}), http.StatusNotFound},
	} {
		t.Run(one.what, func(t *testing.T) {
			if out := ask(handler, one.url); out.Code != one.want {
				t.Errorf("%s was answered %d, want %d", one.url, out.Code, one.want)
			}
		})
	}
}

// The window's own pieces reach it.
//
// A file of the vault is asked for under `assets`. The window's own scripts are
// served from elsewhere.
func TestTheWindowsOwnPiecesAreServed(t *testing.T) {
	files, err := Pages()
	if err != nil {
		t.Skipf("no interface in this binary: %v", err)
	}
	handler := (&API{}).NewHandler(files)

	page := ask(handler, "/")
	if page.Code != http.StatusOK {
		t.Fatalf("the page itself was answered %d", page.Code)
	}
	named := regexp.MustCompile(`(?:src|href)="(/[^"]+)"`).FindAllStringSubmatch(page.Body.String(), -1)
	if len(named) == 0 {
		t.Fatal("the page names nothing to load")
	}
	for _, one := range named {
		if out := ask(handler, one[1]); out.Code != http.StatusOK {
			t.Errorf("%s was answered %d", one[1], out.Code)
		}
	}
}
