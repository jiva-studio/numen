package webui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// A page of a document the vault holds is bytes, and bytes are what this route
// answers:
//
//	GET /assets/<id>/pages/<n>?wide=W   one page, drawn to that width
//
// The id is the file's path in the vault, escaped. A file has no other name the
// window holds.
//
// Nothing else of a file is here. What a file is, where a run of its text sits,
// and what a model made from it are questions of the schema; a page is a
// picture, and a picture is what an `img` loads.
const assetsRoute = "/assets/"

// The one facet an asset offers.
const pagesFacet = "pages"

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

// Asset answers with the bytes of one page of one file of the vault.
func (a *API) Asset(w http.ResponseWriter, r *http.Request) {
	at, ok := addressed(r)
	if !ok {
		http.Error(w, "not an asset", http.StatusBadRequest)
		return
	}
	if at.facet != pagesFacet {
		http.Error(w, "an asset answers with a page and nothing else", http.StatusNotFound)
		return
	}
	a.Page(w, r, at.path, at.at)
}

// assetOf is where a file of the vault is drawn from, and pageOf one page of
// it. A path is escaped whole, so a file in a folder is one segment and what
// hangs off it is the next.
func assetOf(path string) string { return assetsRoute + url.PathEscape(path) }

func pageOf(path string, at, wide int) string {
	return fmt.Sprintf("%s/%s/%d?wide=%d", assetOf(path), pagesFacet, at, wide)
}
