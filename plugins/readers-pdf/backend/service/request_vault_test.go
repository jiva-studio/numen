package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

// twoVaults is a current vault without the file and a second vault holding tiny.pdf with its text
// layer written, and the texts of the current vault.
type twoVaults struct {
	current *hosttest.Vault
	other   *hosttest.Vault
	texts   *hosttest.Texts
	writer  *extract.Writer
}

func newTwoVaults(t *testing.T) twoVaults {
	t.Helper()
	raw := testsupport.ReadFixture(t, "tiny.pdf")
	current := hosttest.OnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.OnDisk(t, "Chemistry", map[string]string{"tiny.pdf": string(raw)})
	current.Beside(other)
	texts := hosttest.NewTexts("Physics", testsupport.PluginID)
	chemistry := texts.With("Chemistry")
	chemistry.File("tiny.pdf", raw)
	chemistry.SetStore(testsupport.PluginID, other.Store())
	writer := testsupport.NewWriter(t, testsupport.FilesOf(current), current, texts, hosttest.NewQueue())
	testsupport.CreateText(t, writer, extract.Request{Vault: "Chemistry", Path: "tiny.pdf"})
	return twoVaults{current: current, other: other, texts: texts, writer: writer}
}

func postAPI(t *testing.T, documents *DocumentServer, method, vault, body string, response any) int {
	t.Helper()
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "token"}, func(s *server.Server) error {
		return documents.Mount(s.NewAPI(api.Service))
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/"+api.Service+"/"+method, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	if vault != "" {
		req.Header.Set("Numen-Vault", vault)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), response); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code
}

func TestGetHighlightsReadsTheTextOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	documents := NewDocumentServer(newReader(t, v.current, v.texts))
	var res api.GetHighlightsResponse

	code := postAPI(t, documents, "GetHighlights", "Chemistry", `{"path":"tiny.pdf","ranges":[{"start":0,"end":4}]}`, &res)

	if code != http.StatusOK {
		t.Fatalf("GetHighlights answered %d", code)
	}
	if len(res.Runs) != 1 || len(res.Runs[0].Places) == 0 {
		t.Errorf("runs %v, want the run placed by Chemistry's text layer", res.Runs)
	}
}

func TestGetDocumentOpensTheFileOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	documents := NewDocumentServer(newReader(t, v.current, v.texts))
	var res api.GetDocumentResponse

	code := postAPI(t, documents, "GetDocument", "Chemistry", `{"path":"tiny.pdf"}`, &res)

	if code != http.StatusOK || len(res.Pages) == 0 {
		t.Fatalf("GetDocument in Chemistry = %d %v, want its pages", code, res)
	}
	if code := postAPI(t, documents, "GetDocument", "", `{"path":"tiny.pdf"}`, &res); code == http.StatusOK {
		t.Error("GetDocument without a vault opened a file the current vault does not hold")
	}
}

func TestPageServerDrawsThePageOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	pages := NewPageServer(newReader(t, v.current, v.texts), func(string, error) {})
	mux := http.NewServeMux()
	mux.Handle("GET /"+pagesPattern, pages)

	named := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pages/0?path=tiny.pdf&wide=400", nil)
	named.Header.Set("Numen-Vault", "Chemistry")
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, named)
	if page.Code != http.StatusOK || page.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("the page of Chemistry answered %d %q", page.Code, page.Body.String())
	}

	current := httptest.NewRecorder()
	mux.ServeHTTP(current, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pages/0?path=tiny.pdf&wide=400", nil))
	if current.Code == http.StatusOK {
		t.Error("a request naming no vault drew a file the current vault does not hold")
	}
}
