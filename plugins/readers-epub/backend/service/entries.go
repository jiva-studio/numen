package service

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
)

// entriesPattern is where the entries of a book's archive are served below the plugin route. The
// window requests /plugin/md.numen.readers.epub/entries/<entry>?path=<vault path>&size=<bytes>&mtime=<ns>.
const entriesPattern = "entries/{entry...}"

// EntryServer serves the entries of the books of a vault, such as the pictures a book carries.
type EntryServer struct {
	books *BookServer
}

// NewEntryServer serves the entries the book server reads.
func NewEntryServer(books *BookServer) *EntryServer {
	return &EntryServer{books: books}
}

// ServeHTTP answers a request for one entry: the entry is the path value "entry", the book is the
// query parameter "path", and the bytes the book was read from are the query parameters "size" and
// "mtime".
func (s *EntryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := api.ReadBookEntryRequest{
		Path:     r.URL.Query().Get("path"),
		Entry:    r.PathValue("entry"),
		Baseline: parseBytesQuery(r.URL.Query()),
	}
	res, err := s.books.ReadBookEntry(withRequestVault(r.Context(), r.Header), req)
	if err != nil {
		failure := toAPIError(err)
		http.Error(w, failure.Error(), getStatus(failure.Code))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="HTTP response write errors are logged to stderr"
	if _, err := w.Write(res.Body); err != nil {
		fmt.Fprintf(os.Stderr, "readers-epub: write the entry %s: %v\n", r.PathValue("entry"), err)
	}
}

func parseBytesQuery(query url.Values) *api.FileStat {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="size query parameter is a base-10 64-bit integer"
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return nil
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mtime query parameter is a base-10 64-bit integer"
	stamp, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	// gomutants:disable-next-line BRANCH_IF reason="strconv.ParseInt returns 0 on error"
	if err != nil {
		stamp = 0
	}
	return &api.FileStat{Path: "", Size: size, Mtime: stamp}
}
