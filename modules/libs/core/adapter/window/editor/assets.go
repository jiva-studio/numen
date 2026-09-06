package editor

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// What a file the vault holds is made of is bytes, and bytes are what this route
// answers:
//
//	GET /assets/<id>/pages/<n>?wide=W&size=S&mtime=T   one page, drawn to that width
//	GET /assets/<id>/markup/<doc>?size=S&mtime=T       one document of a book
//	GET /assets/<id>/entries/<name>?size=S&mtime=T     one picture the book carries
//
// The id is the file's path in the vault, escaped, and so is what a facet is
// named with. Bytes are the whole of what this route answers; everything else
// about a file is asked over the schema.
const assetsRoute = "/assets/"

// The facets an asset offers.
const (
	pagesFacet   = "pages"
	markupFacet  = "markup"
	entriesFacet = "entries"
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
		if held.at, err = url.PathUnescape(parts[2]); err != nil {
			return address{}, false
		}
	}
	if len(parts) > 3 {
		return address{}, false
	}
	return held, true
}

// Asset answers with the bytes one file of the vault is made of.
func (a *API) Asset(w http.ResponseWriter, r *http.Request) {
	at, ok := addressed(r)
	if !ok {
		http.Error(w, "not an asset", http.StatusBadRequest)
		return
	}
	switch at.facet {
	case pagesFacet:
		a.Page(w, r, at.path, at.at)
	case markupFacet:
		a.Markup(w, r, at.path, at.at)
	case entriesFacet:
		a.Entry(w, r, at.path, at.at)
	default:
		http.Error(w, "an asset answers with none of that", http.StatusNotFound)
	}
}

// assetOf is where a file of the vault is drawn from, and pageOf one page of
// it. A path is escaped whole, so a file in a folder is one segment and what
// hangs off it is the next.
func assetOf(path string) string { return assetsRoute + url.PathEscape(path) }

func pageOf(path string, at, wide int, print fingerprint) string {
	return fmt.Sprintf("%s/%s/%d?wide=%d&%s", assetOf(path), pagesFacet, at, wide, printing(print))
}

// markupOf is one document of a book, and entryOf one entry of the archive it
// was read out of. What names either is a path inside that archive, escaped
// whole the way the file's own path is.
func markupOf(path, document string, print fingerprint) string {
	return facetOf(path, markupFacet, document, print)
}

func entryOf(path, entry string, print fingerprint) string {
	return facetOf(path, entriesFacet, entry, print)
}

func facetOf(path, facet, at string, print fingerprint) string {
	return fmt.Sprintf("%s/%s/%s?%s", assetOf(path), facet, url.PathEscape(at), printing(print))
}

// An address that names which bytes it is about answers those bytes or nothing,
// so what it answers with may be kept for as long as anything keeps anything.
const immutable = "public, max-age=31536000, immutable"

// errChanged is a file that is no longer the one the address was given out for.
// The address is gone: the caller asks the vault again and is given another.
var errChanged = errors.New("the file changed since this address was given out")

// printing is a fingerprint as an address carries it, and printed is it read
// back. The path is a segment of the address already, so what is written here
// is the rest of what says which bytes the file is.
func printing(print fingerprint) string {
	return fmt.Sprintf("size=%d&mtime=%d", print.size, print.mtime)
}

func printed(query url.Values) (fingerprint, error) {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return fingerprint{}, fmt.Errorf("size: %q is not a size", query.Get("size"))
	}
	mtime, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	if err != nil {
		return fingerprint{}, fmt.Errorf("mtime: %q is not a time", query.Get("mtime"))
	}
	return fingerprint{size: size, mtime: mtime}, nil
}

// stamp and instant are a modification time as this adapter carries it — an
// address and the schema both say nanoseconds since the epoch — and as the core
// holds one. Zero is a file nothing was said about, not the epoch.

func stamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func instant(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
