package webui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// A file of the vault is one part of its address, and what is asked of it is the
// next.
//
// The path is written out whole, so a file standing in a folder named for a
// facet is still one part. Read as the parts arrived, `pages/pages/x.pdf` and
// the page asked of it are the same three words twice over.
func TestAFileIsOnePartOfItsAddress(t *testing.T) {
	for _, one := range []struct {
		what  string
		url   string
		path  string
		facet string
		at    string
	}{
		{"a file", assetOf("library/a.pdf"), "library/a.pdf", "", ""},
		{"a page of it", pageOf("library/a.pdf", 3, 800), "library/a.pdf", pagesFacet, "3"},
		{"where a run of it sits", marksOf("library/a.pdf", 0, 5), "library/a.pdf", marksFacet, ""},
		{"a file in a folder named for a facet", pageOf("pages/pages/x.pdf", 2, 400), "pages/pages/x.pdf", pagesFacet, "2"},
		{"a name with a space in it", assetOf("library/A Book.pdf"), "library/A Book.pdf", "", ""},
	} {
		t.Run(one.what, func(t *testing.T) {
			got, ok := addressed(httptest.NewRequest("GET", one.url, nil))
			if !ok {
				t.Fatalf("%s is not an address", one.url)
			}
			if got.path != one.path || got.facet != one.facet || got.at != one.at {
				t.Errorf("%s reads as %q/%q/%q, want %q/%q/%q",
					one.url, got.path, got.facet, got.at, one.path, one.facet, one.at)
			}
		})
	}
}

func TestWhatIsNotAnAssetIsNotAnAddress(t *testing.T) {
	for _, url := range []string{"/assets/", "/assets", "/elsewhere/a.pdf", assetOf("a.pdf") + "/pages/1/more"} {
		if _, ok := addressed(httptest.NewRequest("GET", url, nil)); ok {
			t.Errorf("%s was read as an address", url)
		}
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
	handler := (&API{}).Serving(files)

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
