//go:build numen_debug

package debugserve

import (
	"bytes"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

// fetchRoute is where the page finds the script that sends its calls as PUT.
const fetchRoute = "/numen-debug/fetch.js"

//go:embed fetch.js
var fetchScript []byte

func newHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == fetchRoute:
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			writeBody(w, fetchScript)
		case r.Method == http.MethodPut:
			r.Method = http.MethodPost
			handler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && isPage(r.URL.Path):
			writePage(w, r, handler)
		default:
			handler.ServeHTTP(w, r)
		}
	})
}

func isPage(path string) bool {
	return path == "" || path == "/" || path == "/index.html"
}

func writePage(w http.ResponseWriter, r *http.Request, handler http.Handler) {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	body := rec.Body.Bytes()
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		body = addScript(body)
	}
	for name, values := range rec.Header() {
		w.Header()[name] = values
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(rec.Code)
	writeBody(w, body)
}

func writeBody(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		return
	}
}

func addScript(page []byte) []byte {
	head := []byte("<head>")
	headEnd := bytes.Index(page, head)
	if headEnd < 0 {
		return page
	}
	headEnd += len(head)
	tag := []byte(`<script src="` + fetchRoute + `"></script>`)
	return append(page[:headEnd:headEnd], append(tag, page[headEnd:]...)...)
}
