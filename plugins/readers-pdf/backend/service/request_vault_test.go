package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	pdfv1 "github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1"
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
	chemistry, isTexts := texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of the vault are not hosttest texts")
	}
	chemistry.File("tiny.pdf", raw)
	chemistry.SetStore(pluginID, other.Store())
	server := NewTextServer(current, texts, current.Store(), hosttest.NewQueue())
	t.Cleanup(server.Close)
	createText(t, server, host.TextCreation{Vault: "Chemistry", Path: "tiny.pdf"})
	return twoVaults{current: current, other: other, texts: texts, server: server}
}

func TestGetHighlightsReadsTheTextOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	documents := NewDocumentServer(v.current, v.texts)
	req := connect.NewRequest(&pdfv1.GetHighlightsRequest{Path: "tiny.pdf", Ranges: []*pdfv1.Range{{Start: 0, End: 4}}})
	req.Header().Set("Numen-Vault", "Chemistry")

	res, err := documents.GetHighlights(t.Context(), req)
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	if runs := res.Msg.GetRuns(); len(runs) != 1 || len(runs[0].GetPlaces()) == 0 {
		t.Errorf("runs %v, want the run placed by Chemistry's text layer", runs)
	}
}

func TestGetDocumentOpensTheFileOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	documents := NewDocumentServer(v.current, v.texts)

	named := connect.NewRequest(&pdfv1.GetDocumentRequest{Path: "tiny.pdf"})
	named.Header().Set("Numen-Vault", "Chemistry")
	res, err := documents.GetDocument(t.Context(), named)
	if err != nil || len(res.Msg.GetPages()) == 0 {
		t.Fatalf("GetDocument in Chemistry = %v, %v, want its pages", res, err)
	}
	if _, err := documents.GetDocument(t.Context(), connect.NewRequest(&pdfv1.GetDocumentRequest{Path: "tiny.pdf"})); err == nil {
		t.Error("GetDocument without a vault opened a file the current vault does not hold")
	}
}

func TestPageServerDrawsThePageOfTheVaultTheRequestNames(t *testing.T) {
	v := newTwoVaults(t)
	pages := NewPageServer(v.current)
	mux := http.NewServeMux()
	mux.Handle("GET /"+pagesPattern, pages)

	named := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pages/0?path=tiny.pdf", nil)
	named.Header.Set("Numen-Vault", "Chemistry")
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, named)
	if page.Code != http.StatusOK || page.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("the page of Chemistry answered %d %q", page.Code, page.Body.String())
	}

	current := httptest.NewRecorder()
	mux.ServeHTTP(current, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pages/0?path=tiny.pdf", nil))
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
	listing, err := texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Fatalf("versions %v while the queue is held, want none", listing.Versions)
	}
	held.Release()
	server.Wait()

	listing, err = texts.ListVersions(t.Context(), "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 {
		t.Errorf("versions %v after the turn, want the text layer", listing.Versions)
	}
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %v after the layer was written, want none", tasks)
	}
}
