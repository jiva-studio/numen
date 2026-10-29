package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// twoVaults is a current vault without the file and a second vault holding tiny.pdf with its text
// layer written, and the texts of the current vault.
type twoVaults struct {
	current *hosttest.Vault
	other   *hosttest.Vault
	texts   *hosttest.Texts
	server  *TextServer
}

func newTwoVaults(t *testing.T) twoVaults {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	current := hosttest.OnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.OnDisk(t, "Chemistry", map[string]string{"tiny.pdf": string(raw)})
	current.Beside(other)
	texts := hosttest.NewTexts("Physics", pluginID)
	chemistry := texts.With("Chemistry").(*hosttest.Texts)
	chemistry.File("tiny.pdf", raw)
	chemistry.SetStore(pluginID, other.Store())
	server := NewTextServer(current, texts, current.Store(), hosttest.NewQueue())
	t.Cleanup(server.Close)
	createText(t, server, host.TextCreation{Vault: "Chemistry", Path: "tiny.pdf"})
	return twoVaults{current: current, other: other, texts: texts, server: server}
}

// postAPI asks a method of the document API of a server over HTTP, naming the vault where it is not
// empty, and decodes the response.
func postAPI(t *testing.T, documents *DocumentServer, method, vault, body string, response any) int {
	t.Helper()
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "token"}, func(s *server.Server) error {
		documents.Mount(s.NewAPI(APIService))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/"+APIService+"/"+method, strings.NewReader(body))
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
	documents := NewDocumentServer(v.current, v.texts)
	var res GetHighlightsResponse

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
	documents := NewDocumentServer(v.current, v.texts)
	var res GetDocumentResponse

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
	pages := NewPageServer(v.current)
	mux := http.NewServeMux()
	mux.Handle("GET /"+pagesPattern, pages)

	named := httptest.NewRequest(http.MethodGet, "/pages/0?path=tiny.pdf", nil)
	named.Header.Set("Numen-Vault", "Chemistry")
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, named)
	if page.Code != http.StatusOK || page.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("the page of Chemistry answered %d %q", page.Code, page.Body.String())
	}

	current := httptest.NewRecorder()
	mux.ServeHTTP(current, httptest.NewRequest(http.MethodGet, "/pages/0?path=tiny.pdf", nil))
	if current.Code == http.StatusOK {
		t.Error("a request naming no vault drew a file the current vault does not hold")
	}
}

func TestCreateTextWritesTheLayerInATurnOfTheQueue(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	files := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(raw)})
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File("tiny.pdf", raw)
	queue := hosttest.NewQueue()
	held, err := queue.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	server := NewTextServer(files, texts, texts.GetStore(pluginID), queue)
	t.Cleanup(server.Close)

	if err := server.CreateText(t.Context(), host.TextCreation{Path: "tiny.pdf"}); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	if listing, _ := texts.ListVersions(t.Context(), "tiny.pdf"); len(listing.Versions) != 0 {
		t.Fatalf("versions %v while the queue is held, want none", listing.Versions)
	}
	held.Release()
	server.Wait()

	if listing, _ := texts.ListVersions(t.Context(), "tiny.pdf"); len(listing.Versions) != 1 {
		t.Errorf("versions %v after the turn, want the text layer", listing.Versions)
	}
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %v after the layer was written, want none", tasks)
	}
}
