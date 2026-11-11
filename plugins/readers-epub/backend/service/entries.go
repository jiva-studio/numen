package service

import (
	"log"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/books"
)

// entriesPattern is where the entries of a book's archive are served below the plugin route. The
// window requests /plugin/md.numen.readers.epub/entries/<entry>?path=<vault path>&size=<bytes>&mtime=<ns>.
const entriesPattern = "entries/{entry...}"

// EntryServer serves the entries of the books of a vault, such as the pictures a book carries.
type EntryServer struct {
	reader *books.Reader
}

// NewEntryServer serves the entries the book reader reads.
func NewEntryServer(reader *books.Reader) *EntryServer {
	return &EntryServer{reader: reader}
}

// ServeHTTP answers a request for one entry: the entry is the path value "entry", the book is the
// query parameter "path", and the bytes the book was read from are the query parameters "size" and
// "mtime".
func (s *EntryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	req := api.ReadBookEntryRequest{
		Path:     query.Get("path"),
		Entry:    r.PathValue("entry"),
		Baseline: books.ParseBaseline(query.Get("size"), query.Get("mtime")),
	}
	res, err := s.reader.ReadBookEntry(books.WithVault(r.Context(), host.GetRequestVault(r.Header)), req)
	if err != nil {
		failure := toAPIError(err)
		http.Error(w, failure.Error(), getStatus(failure.Code))
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="HTTP response write errors are logged to stderr"
	if _, err := w.Write(res.Body); err != nil {
		log.Printf("readers-epub: write the entry %s: %v", r.PathValue("entry"), err)
	}
}
