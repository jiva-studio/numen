//go:build numen_debug

package debugserve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestNewHandlerAddsFetchScriptToPage(t *testing.T) {
	window := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<html><head><script type="module" src="/built/index.js"></script></head></html>`)
	})
	rec := httptest.NewRecorder()
	newHandler(window).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	got := rec.Body.String()
	target := `<html><head><script src="/numen-debug/fetch.js"></script><script type="module"`
	if !strings.HasPrefix(got, target) {
		t.Errorf("page reads %q", got)
	}
	if rec.Header().Get("Content-Length") != strconv.Itoa(len(got)) {
		t.Errorf("Content-Length %q for %d bytes", rec.Header().Get("Content-Length"), len(got))
	}
}

func TestNewHandlerServesPUTAsPOST(t *testing.T) {
	var method string
	window := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { method = r.Method })
	newHandler(window).ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPut, "/numen.v1.VaultsService/ListVaults", strings.NewReader("{}")))
	if method != http.MethodPost {
		t.Errorf("the window was asked with %s", method)
	}
}

func TestNewHandlerServesFetchScript(t *testing.T) {
	rec := httptest.NewRecorder()
	newHandler(http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fetchRoute, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "method: 'PUT'") {
		t.Errorf("%d %q", rec.Code, rec.Body.String())
	}
}
