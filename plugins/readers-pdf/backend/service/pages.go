package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/document"
)

// pagesPattern is the route under the plugin route that serves the page images.
// The window requests /plugin/md.numen.readers.pdf/pages/<index>?path=<vault path>.
const pagesPattern = "pages/{page}"

// PageServer draws the pages of the PDF files in a vault as JPEG images.
type PageServer struct {
	reader *document.Reader
	report func(what string, err error)
}

// NewPageServer returns a PageServer that draws the pages reader reads. A failure to write a page goes to
// report.
func NewPageServer(reader *document.Reader, report func(what string, err error)) *PageServer {
	return &PageServer{reader: reader, report: report}
}

// ServeHTTP answers a request for one page: the index is the path value "page", counted from zero,
// the file is the query parameter "path", relative to the vault the Numen-Vault header names, and the
// query parameter "wide" is how many pixels across the page is drawn.
func (s *PageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := document.ParsePageRequest(r.PathValue("page"), query.Get("path"), query.Get("wide"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	image, err := s.reader.RenderPage(r.Context(), host.GetRequestVault(r.Header), page.Path, page.Index, page.PixelWidth)
	if err != nil {
		http.Error(w, err.Error(), getPageStatus(err))
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(image); err != nil {
		s.report(fmt.Sprintf("write the page %d of %s", page.Index, page.Path), err)
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
