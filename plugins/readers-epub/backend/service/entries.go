package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"connectrpc.com/connect"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
)

// entriesPattern is where the entries of a book's archive are served below the plugin route. The
// window asks for /plugin/md.numen.readers.epub/entries/<entry>?path=<vault path>&size=<bytes>&mtime=<ns>.
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
	req := connect.NewRequest(&epubv1.ReadBookEntryRequest{
		Path:     r.URL.Query().Get("path"),
		Entry:    r.PathValue("entry"),
		Baseline: parseBytesQuery(r.URL.Query()),
	})
	if named := r.Header.Get(vaultHeader); named != "" {
		req.Header().Set(vaultHeader, named)
	}
	res, err := s.books.ReadBookEntry(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), getHTTPStatus(err))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(res.Msg.GetBody()); err != nil {
		fmt.Fprintf(os.Stderr, "readers-epub: write the entry %s: %v\n", r.PathValue("entry"), err)
	}
}

// vaultHeader names the vault a request works.
const vaultHeader = "Numen-Vault"

// parseBytesQuery reads the size and the modification time of a file from the query, and is nil
// where the query names no size.
func parseBytesQuery(query url.Values) *epubv1.FileStat {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return nil
	}
	stamp, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	if err != nil {
		stamp = 0
	}
	return &epubv1.FileStat{Size: size, Mtime: stamp}
}

// getHTTPStatus is the HTTP status of a Connect error.
func getHTTPStatus(err error) int {
	var named *connect.Error
	if !errors.As(err, &named) {
		return http.StatusInternalServerError
	}
	switch named.Code() {
	case connect.CodeInvalidArgument:
		return http.StatusBadRequest
	case connect.CodePermissionDenied:
		return http.StatusForbidden
	case connect.CodeNotFound:
		return http.StatusNotFound
	case connect.CodeAborted, connect.CodeFailedPrecondition:
		return http.StatusConflict
	case connect.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
