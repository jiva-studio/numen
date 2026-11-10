package service

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/document"
)

// pagesPattern is the route under the plugin route that serves the page images.
// The window requests /plugin/md.numen.readers.pdf/pages/<index>?path=<vault path>.
const pagesPattern = "pages/{page}"

// PageServer draws the pages of the PDF files in a vault as JPEG images.
type PageServer struct {
	reader *document.Reader
}

// NewPageServer returns a PageServer that draws the pages reader reads.
func NewPageServer(reader *document.Reader) *PageServer {
	return &PageServer{reader: reader}
}

// ServeHTTP answers a request for one page: the index is the path value "page", counted from zero,
// the file is the query parameter "path", relative to the vault the Numen-Vault header names, and the
// query parameter "wide" is how many pixels across the page is drawn.
func (s *PageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || index < 0 {
		http.Error(w, "invalid page", http.StatusBadRequest)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	pixelWidth, err := strconv.Atoi(r.URL.Query().Get("wide"))
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="strconv.Atoi returns pixelWidth 0 on parse failure"
	if err != nil || pixelWidth <= 0 {
		http.Error(w, "invalid wide", http.StatusBadRequest)
		return
	}
	image, err := s.reader.RenderPage(r.Context(), host.GetRequestVault(r.Header), path, index, pixelWidth)
	if err != nil {
		http.Error(w, err.Error(), getPageStatus(err))
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(image); err != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: write the page %d of %s: %v\n", index, path, err)
	}
}

func getPageStatus(err error) int {
	var render *document.RenderError
	var gaveUp *document.GaveUpError
	switch {
	case errors.As(err, &render):
		return http.StatusUnprocessableEntity
	case errors.As(err, &gaveUp):
		return http.StatusRequestTimeout
	default:
		return getStatus(err)
	}
}
