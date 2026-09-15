package editor

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// What a file of the vault is made of is bytes, and bytes are what this route
// answers:
//
//	GET /assets/<id>/<where>    the bytes at that place in the file
//
// The id is the file's path in the vault, escaped whole, and where in the file
// is the rest. Which places a file has is the reader's to say: a document is
// asked for a page of it, a book that reflows for a picture it carries. Bytes
// are the whole of what this route answers; everything else about a file is
// asked over the schema.
const assetsRoute = "/assets/"

// An address is one question about one file: which file, and where in it.
type address struct {
	path  string
	where string
}

// parseAssetAddress takes an asset's address apart.
//
// The escaped path is what is read: Go decodes before a handler is reached, and
// a decoded separator runs the id and the place after it together. A place is
// as many segments as the file names it with, each escaped on its own.
func parseAssetAddress(r *http.Request) (address, bool) {
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
	if len(parts) == 1 {
		return held, true
	}
	asked := make([]string, 0, len(parts)-1)
	for _, one := range parts[1:] {
		said, err := url.PathUnescape(one)
		if err != nil {
			return address{}, false
		}
		asked = append(asked, said)
	}
	held.where = strings.Join(asked, "/")
	return held, true
}

// Asset answers with the bytes of one file of the vault, at the place asked of
// it.
//
// The reader the file is read by names the places it has, and a file no reader
// reads has none.
func (a *API) Asset(w http.ResponseWriter, r *http.Request) {
	at, ok := parseAssetAddress(r)
	if !ok {
		http.Error(w, "not an asset", http.StatusBadRequest)
		return
	}
	if at.where == "" {
		http.Error(w, "an asset is asked with where in the file", http.StatusBadRequest)
		return
	}
	switch named, _ := derived.ReaderName(domain.Fingerprint{Path: at.path}); named {
	case derived.ReaderEPUB:
		a.Entry(w, r, at.path, at.where)
	case derived.ReaderPDF:
		a.Page(w, r, at.path, at.where)
	default:
		http.Error(w, "no reader reads that file", http.StatusNotFound)
	}
}

// assetOf is where a file of the vault is asked about, and pageOf one page of
// it. A path is escaped whole, so a file in a folder is one segment and what is
// asked of it is the next.
func assetOf(path string) string { return assetsRoute + url.PathEscape(path) }

func pageOf(path string, at, wide int, print fingerprint) string {
	return fmt.Sprintf("%s/%s/%d?wide=%d&%s", assetOf(path), pagesName, at, wide, formatFingerprint(print))
}

// An address that names which bytes it is about answers those bytes or nothing,
// so what it answers with may be kept for as long as anything keeps anything.
const immutable = "public, max-age=31536000, immutable"

// errChanged is a file that is no longer the one the address was given out for.
// The address is gone: the caller asks the vault again and is given another.
var errChanged = errors.New("the file changed since this address was given out")

// formatFingerprint is a fingerprint as an address carries it, and parseFingerprint is it read
// back. The path is a segment of the address already, so what is written here
// is the rest of what says which bytes the file is.
func formatFingerprint(print fingerprint) string {
	return fmt.Sprintf("size=%d&mtime=%d", print.size, print.mtime)
}

func parseFingerprint(query url.Values) (fingerprint, error) {
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
