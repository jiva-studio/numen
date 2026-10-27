package service

import (
	"bytes"
	"errors"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// pagesPattern is where the page images are served below the plugin route. The window asks for
// /plugin/md.numen.readers.pdf/pages/<index>?path=<vault path>.
const pagesPattern = "pages/{page}"

// pageDPI is the resolution a page image is drawn at.
const pageDPI = 150

// PageServer draws the pages of the PDF files in a vault as JPEG images.
type PageServer struct {
	vaults vaultRoots
}

// NewPageServer returns a PageServer that finds the vault of a request through files, and works the
// vault at defaultRoot for a request naming none.
func NewPageServer(files host.Vault, defaultRoot func() string) *PageServer {
	return &PageServer{vaults: vaultRoots{files: files, defaultRoot: defaultRoot}}
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
	cleanPath, err := s.vaults.getFilePath(r.Context(), r.Header, path)
	if err != nil {
		http.Error(w, err.Error(), getStatus(err))
		return
	}
	image, err := renderPage(cleanPath, index)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(image)
}

// renderPage draws one page of the PDF at path as JPEG bytes.
func renderPage(path string, index int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, err
	}
	defer scan.Close()
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
