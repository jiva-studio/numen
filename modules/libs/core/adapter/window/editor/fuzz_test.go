package editor

import (
	"net/http/httptest"
	"strconv"
	"testing"
)

// pathSeeds are the names a file of a vault arrives under: one in a folder, one
// carrying the separator the address is read on, the punctuation a URL is taken
// apart by, a name in another script, a name that is not UTF-8 at all, and the
// names a filesystem gives its own places.
var pathSeeds = []string{
	"a.pdf",
	"library/a.pdf",
	"library/A Book.pdf",
	"pages/pages/x.pdf",
	"a?b.pdf",
	"a#b.pdf",
	"a%2Fb.pdf",
	"a%b.pdf",
	"a+b.pdf",
	"a&b=c.pdf",
	"a;b.pdf",
	"книга.pdf",
	"\xff\xfe.pdf",
	"a\nb.pdf",
	"..",
	"a/../b.pdf",
	"/absolute.pdf",
}

// A file of the vault is addressed by its path and nothing else, so the address
// has to carry every name a filesystem allows and come back as itself. The
// escaped path is what is read: Go decodes before a handler is reached, and a
// decoded separator would run the file and what is asked of it together.
//
// A path arrives from a person's own folder, synchronised from another machine,
// so the bytes are a stranger's.
func FuzzAssetAddress(f *testing.F) {
	for _, path := range pathSeeds {
		f.Add(path, 3, int64(1000), int64(2))
	}

	f.Fuzz(func(t *testing.T, path string, page int, size, mtime int64) {
		// A file has no other name the window holds, so a file of no name is no
		// address, and a page is one of a document's own.
		if path == "" || page < 0 {
			return
		}
		print := fingerprint{size: size, mtime: mtime}
		url := pageOf(path, page, 800, print)

		at, ok := addressed(httptest.NewRequest("GET", url, nil))
		if !ok {
			t.Fatalf("%q is addressed as %q, which is no address", path, url)
		}
		if at.path != path {
			t.Fatalf("%q is addressed as %q and read back as %q", path, url, at.path)
		}
		if want := pagesName + "/" + strconv.Itoa(page); at.where != want {
			t.Fatalf("page %d of %q is addressed as %q and asks for %q", page, path, url, at.where)
		}

		// Which bytes the address is about rides beside it, and says the same
		// thing on the way back.
		got, err := printed(httptest.NewRequest("GET", url, nil).URL.Query())
		if err != nil {
			t.Fatalf("%q carries %+v and came back: %v", url, print, err)
		}
		if got != print {
			t.Fatalf("%q carries %+v and came back as %+v", url, print, got)
		}
	})
}
