package webui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// A file the vault holds is an asset, addressed as one: the collection, the
// file, and what hangs off it.
//
//	GET /assets/<id>                          what it is
//	GET /assets/<id>/pages/<n>?wide=W         one page, where it has any
//	GET /assets/<id>/marks?start=N&length=M   where a run of its text sits
//
// The id is the file's path in the vault, escaped. A file has no other name the
// window holds, and a recording asked what it is would answer with a duration
// and grow a facet of its own.
const assetsRoute = "/assets/"

// The facets an asset offers.
const (
	pagesFacet = "pages"
	marksFacet = "marks"
)

// An address is one question about one asset: which file, which facet, and what
// the facet was named with.
type address struct {
	path  string
	facet string
	at    string
}

// addressed takes an asset's address apart.
//
// The escaped path is what is read: Go decodes before a handler is reached, and
// a decoded separator runs the id and the facet after it together.
func addressed(r *http.Request) (address, bool) {
	rest := strings.TrimPrefix(r.URL.EscapedPath(), assetsRoute)
	if rest == "" || rest == r.URL.EscapedPath() {
		return address{}, false
	}
	parts := strings.Split(rest, "/")
	path, err := url.PathUnescape(parts[0])
	if err != nil || path == "" {
		return address{}, false
	}
	held := address{path: path}
	if len(parts) > 1 {
		held.facet = parts[1]
	}
	if len(parts) > 2 {
		held.at = parts[2]
	}
	if len(parts) > 3 {
		return address{}, false
	}
	return held, true
}

// Asset answers one question about one file of the vault.
func (a *API) Asset(w http.ResponseWriter, r *http.Request) {
	at, ok := addressed(r)
	if !ok {
		http.Error(w, "not an asset", http.StatusBadRequest)
		return
	}
	switch at.facet {
	case "":
		a.Document(w, r, at.path)
	case pagesFacet:
		a.Page(w, r, at.path, at.at)
	case marksFacet:
		a.Marks(w, r, at.path)
	default:
		http.Error(w, "an asset has no "+at.facet, http.StatusNotFound)
	}
}

// assetOf is where a file of the vault is asked about, and pageOf and marksOf
// are its facets. A path is escaped whole, so a file in a folder is one segment
// and what hangs off it is the next.
func assetOf(path string) string { return assetsRoute + url.PathEscape(path) }

func pageOf(path string, at, wide int) string {
	return fmt.Sprintf("%s/%s/%d?wide=%d", assetOf(path), pagesFacet, at, wide)
}

func marksOf(path string, start, length int) string {
	return fmt.Sprintf("%s/%s?start=%d&length=%d", assetOf(path), marksFacet, start, length)
}
