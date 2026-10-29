package service

import (
	"net/http"
	"net/url"
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
	_, _ = w.Write(res.Body)
}

// parseBytesQuery reads the size and the modification time of a file from the query, and is nil
// where the query names no size.
func parseBytesQuery(query url.Values) *api.FileStat {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return nil
	}
	stamp, _ := strconv.ParseInt(query.Get("mtime"), 10, 64)
	return &api.FileStat{Size: size, Mtime: stamp}
}
