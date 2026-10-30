package service

import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// pagesPattern is the route under the plugin route that serves the page images.
// The window requests /plugin/md.numen.readers.pdf/pages/<index>?path=<vault path>.
const pagesPattern = "pages/{page}"

// pageDPI is the resolution a page image is drawn at.
const pageDPI = 150

// PageServer draws the pages of the PDF files in a vault as JPEG images.
type PageServer struct {
	files host.Vault
}

// NewPageServer returns a PageServer that reads files from files.
func NewPageServer(files host.Vault) *PageServer {
	return &PageServer{files: files}
}

// ServeHTTP answers a request for one page: the index is the path value "page", counted from zero,
// and the file is the query parameter "path", relative to the vault the Numen-Vault header names.
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
	raw, _, err := getVault(s.files, r.Header).Read(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), getStatus(err))
		return
	}
	image, err := renderPage(raw, index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(image); err != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: write the page %d of %s: %v\n", index, path, err)
	}
}

// renderPage draws one page of the PDF bytes raw as JPEG bytes.
func renderPage(raw []byte, index int) (encoded []byte, err error) {
	scan, err := pdf.OpenBytes(raw)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, scan.Close()) }()
	drawing, err := scan.RenderPage(index, pageDPI)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, drawing, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
