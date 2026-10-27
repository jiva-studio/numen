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
)

// newLinkedPages is the linked handler of the plugin over the folder of the test PDFs.
func newLinkedPages(t *testing.T, vaultRoot string) *httptest.Server {
	t.Helper()
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
		"NUMEN_VAULT_PATH":   vaultRoot,
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

// getPage asks web for a page at path with the plugin's token.
func getPage(t *testing.T, web *httptest.Server, path string) (*http.Response, []byte) {
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
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func getTestdata(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "pdf", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPageRouteServesAPageAsJPEG(t *testing.T) {
	web := newLinkedPages(t, getTestdata(t))

	resp, body := getPage(t, web, "/plugin/pages/0?path=tiny.pdf&wide=400")

	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("answered %d %q: %s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
	drawing, err := jpeg.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("the page is no JPEG: %v", err)
	}
	if drawing.Bounds().Dx() == 0 || drawing.Bounds().Dy() == 0 {
		t.Errorf("the page is %v", drawing.Bounds())
	}
}

func TestPageRouteRejectsAFileOutsideTheVault(t *testing.T) {
	vaultRoot := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.pdf")
	tiny, err := os.ReadFile(filepath.Join(getTestdata(t), "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, tiny, 0o644); err != nil {
		t.Fatal(err)
	}
	web := newLinkedPages(t, vaultRoot)

	for _, path := range []string{outside, "../" + filepath.Base(filepath.Dir(outside)) + "/outside.pdf"} {
		resp, _ := getPage(t, web, "/plugin/pages/0?path="+url.QueryEscape(path))
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%q was answered %d", path, resp.StatusCode)
		}
	}
}

func TestPageRouteRejectsABadRequest(t *testing.T) {
	web := newLinkedPages(t, getTestdata(t))

	for path, want := range map[string]int{
		"/plugin/pages/0":                 http.StatusBadRequest,
		"/plugin/pages/x?path=tiny.pdf":   http.StatusBadRequest,
		"/plugin/pages/-1?path=tiny.pdf":  http.StatusBadRequest,
		"/plugin/pages/0?path=gone.pdf":   http.StatusNotFound,
		"/plugin/pages/999?path=tiny.pdf": http.StatusUnprocessableEntity,
	} {
		if resp, body := getPage(t, web, path); resp.StatusCode != want {
			t.Errorf("%s was answered %d, want %d: %s", path, resp.StatusCode, want, body)
		}
	}
}

func TestPageRouteRequiresTheHostToken(t *testing.T) {
	web := newLinkedPages(t, getTestdata(t))

	resp, err := web.Client().Get(web.URL + "/plugin/pages/0?path=tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("answered %d", resp.StatusCode)
	}
}
