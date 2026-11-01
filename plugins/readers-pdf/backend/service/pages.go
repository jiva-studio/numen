package service

import (
	"bytes"
	"context"
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

// mostPixels is the most pixels one page image has. A page asked for wider is drawn as wide as fits,
// and the window scales the image up.
const mostPixels = 1 << 24

// PageServer draws the pages of the PDF files in a vault as JPEG images.
type PageServer struct {
	files   host.Vault
	workers *pdf.Workers
}

// NewPageServer returns a PageServer that reads files from files and draws them on the workers.
func NewPageServer(files host.Vault, workers *pdf.Workers) *PageServer {
	return &PageServer{files: files, workers: workers}
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
	if err != nil || pixelWidth <= 0 {
		http.Error(w, "invalid wide", http.StatusBadRequest)
		return
	}
	raw, _, err := getVault(s.files, r.Header).Read(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), getStatus(err))
		return
	}
	// A window that asked for another width, or left the page, no longer wants this one.
	if err := r.Context().Err(); err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}
	image, err := renderPage(r.Context(), s.workers, raw, index, pixelWidth)
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

// renderPage draws one page of the PDF bytes raw as JPEG bytes, pixelWidth pixels across or as large as fits.
func renderPage(ctx context.Context, workers *pdf.Workers, raw []byte, index, pixelWidth int) (encoded []byte, err error) {
	scan, err := workers.OpenBytes(raw)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, scan.Close()) }()
	// The wait for a worker can outlast the window's interest in the page.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fitting, err := scan.GetFittingWidth(index, pixelWidth, mostPixels)
	if err != nil {
		return nil, err
	}
	drawing, err := scan.RenderPageAtWidth(index, fitting, mostPixels)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, drawing, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
