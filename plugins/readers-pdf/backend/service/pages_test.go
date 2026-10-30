package service

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// newLinkedPages is the linked handler of the plugin over files.
func newLinkedPages(t *testing.T, files host.Vault) *httptest.Server {
	t.Helper()
	handler, err := server.NewLinkedHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	}, func(s *server.Server) error {
		s.RegisterRoute(pagesPattern, NewPageServer(files))
		return nil
	})
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

// page is one answer of the page route.
type page struct {
	status int
	header http.Header
	body   []byte
}

// getPage requests a page at path from web with the plugin's token.
func getPage(t *testing.T, web *httptest.Server, path string) page {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, web.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer linked-token")
	resp, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return page{status: resp.StatusCode, header: resp.Header, body: body}
}

func newTestdataVault(t *testing.T) *hosttest.Vault {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	return hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(raw)})
}

func TestPageRouteServesAPageAsJPEG(t *testing.T) {
	web := newLinkedPages(t, newTestdataVault(t))

	got := getPage(t, web, "/plugin/pages/0?path=tiny.pdf&wide=400")

	if got.status != http.StatusOK || got.header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("answered %d %q: %s", got.status, got.header.Get("Content-Type"), got.body)
	}
	drawing, err := jpeg.Decode(bytes.NewReader(got.body))
	if err != nil {
		t.Fatalf("the page is no JPEG: %v", err)
	}
	if drawing.Bounds().Dx() == 0 || drawing.Bounds().Dy() == 0 {
		t.Errorf("the page is %v", drawing.Bounds())
	}
}

func TestPageRouteRejectsAFileOutsideTheVault(t *testing.T) {
	files := hosttest.New("vault")
	outside := filepath.Join(t.TempDir(), "outside.pdf")
	web := newLinkedPages(t, files)

	for _, path := range []string{outside, "../" + filepath.Base(filepath.Dir(outside)) + "/outside.pdf"} {
		got := getPage(t, web, "/plugin/pages/0?path="+url.QueryEscape(path))
		if got.status != http.StatusForbidden {
			t.Errorf("%q was answered %d", path, got.status)
		}
	}
}

func TestPageRouteRejectsABadRequest(t *testing.T) {
	web := newLinkedPages(t, newTestdataVault(t))

	for path, want := range map[string]int{
		"/plugin/pages/0":                 http.StatusBadRequest,
		"/plugin/pages/x?path=tiny.pdf":   http.StatusBadRequest,
		"/plugin/pages/-1?path=tiny.pdf":  http.StatusBadRequest,
		"/plugin/pages/0?path=gone.pdf":   http.StatusNotFound,
		"/plugin/pages/999?path=tiny.pdf": http.StatusUnprocessableEntity,
	} {
		if got := getPage(t, web, path); got.status != want {
			t.Errorf("%s was answered %d, want %d: %s", path, got.status, want, got.body)
		}
	}
}

func TestPageRouteRequiresTheHostToken(t *testing.T) {
	web := newLinkedPages(t, newTestdataVault(t))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, web.URL+"/plugin/pages/0?path=tiny.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("answered %d", resp.StatusCode)
	}
}
