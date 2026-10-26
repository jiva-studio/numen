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

// newHandler is handler as a browser reaches it over HTTP. The page it opens at loads the script
// that sends the page's calls as PUT, and a PUT is answered as the POST the page sent: nothing the
// window serves answers PUT.
//
// A browser that is not the window's own webview opens a connection for every POST, and WebKit
// opens at most six to one host. The window holds more server streams open than that, so in WebKit
// every call after the sixth stream waits forever. A PUT shares one HTTP/2 connection with every
// other request.
func newHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == fetchRoute:
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			_, _ = w.Write(fetchScript)
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

// isPage is whether path names the document a window opens at.
func isPage(path string) bool {
	return path == "" || path == "/" || path == "/index.html"
}

// writePage writes the page handler answers with, loading the script at fetchRoute before
// anything else in its head.
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
	_, _ = w.Write(body)
}

// addScript puts the script at fetchRoute first in the head of page. A page with no head is
// returned as it is.
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
