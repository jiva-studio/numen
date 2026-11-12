package recognition_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

const pluginID = "md.numen.engines.ocr"

// cacheEnv names the folder the tests of this package compile PDFium into once.
const cacheEnv = "NUMEN_TEST_WASM_CACHE"

// TestMain sets one shared folder to cache the compiled PDFium module across tests.
func TestMain(m *testing.M) {
	dir := os.Getenv(cacheEnv)
	isOwned := false
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "numen-ocr-wasm-")
		if err != nil {
			os.Exit(1)
		}
		if err := os.Setenv(cacheEnv, dir); err != nil {
			os.Exit(1)
		}
		isOwned = true
	}
	code := m.Run()
	if isOwned {
		if err := os.RemoveAll(dir); err != nil && code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// pages returns "page N" for the Nth page, in one block whose two lines are in the top
// and the bottom half of the page, and counts the pages.
type pages struct {
	mu          sync.Mutex
	pageCount   int
	err         error
	says        string
	onRecognize func()
}

func (p *pages) GetRecognitionModel() recognition.Model {
	return recognition.Model{Layout: "layout", Recognizer: "lines", DPI: 72, From: "a test"}
}

func (p *pages) Recognize(_ context.Context, page image.Image) ([]recognition.Block, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pageCount++
	if p.onRecognize != nil {
		p.onRecognize()
	}
	if p.err != nil {
		return nil, p.err
	}
	if p.says == "" {
		return nil, nil
	}
	b := page.Bounds()
	first := p.says + " one"
	text := first + " " + p.says + " two"
	return []recognition.Block{
		{Label: "doc_title", Text: "Heading", IsHeading: true},
		{Label: "text", Text: text, Boxes: []recognition.Box{
			{Rect: image.Rect(0, 0, b.Dx(), b.Dy()/2), Range: recognition.ByteRange{End: len(first)}},
			{Rect: image.Rect(0, b.Dy()/2, b.Dx(), b.Dy()), Range: recognition.ByteRange{Start: len(first) + 1, End: len(text)}},
		}},
	}, nil
}

func (p *pages) getPageCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pageCount
}

// textsOfVault are texts that reject a version of a missing file, as the host does.
type textsOfVault struct {
	host.TextVersions
	vault *hosttest.Vault
}

func (t textsOfVault) PutVersion(ctx context.Context, params host.TextPut) (domain.TextVersion, error) {
	if _, isThere := t.vault.Content(params.Path); !isThere {
		return domain.TextVersion{}, fmt.Errorf("%s: %w", params.Path, host.ErrNotExist)
	}
	return t.TextVersions.PutVersion(ctx, params)
}

func (s *scene) textsOf(id domain.VaultID) host.TextVersions {
	return textsOfVault{TextVersions: s.texts.With(id), vault: s.vault.With(id)}
}

func foldersOf(store *hosttest.Store) func(domain.VaultID) recognition.Folder {
	return func(id domain.VaultID) recognition.Folder { return store.With(id) }
}

// scene is one vault holding a scan and a PDF with a text layer, and the plugin over it.
type scene struct {
	vault      *hosttest.Vault
	store      *hosttest.Store
	texts      *hosttest.Texts
	queue      *hosttest.Queue
	recognizer *pages
	run        *recognition.Recognition
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("../../internal/adapter/document/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func newScene(t *testing.T) *scene {
	t.Helper()
	s := &scene{
		vault:      hosttest.New("Physics"),
		texts:      hosttest.NewTexts("Physics", pluginID),
		queue:      hosttest.NewQueue(),
		recognizer: &pages{says: "page"},
	}
	s.store = s.vault.Store()
	s.texts.SetStore(pluginID, s.store)
	for path, name := range map[string]string{"books/scan.pdf": "scan.pdf", "books/layer.pdf": "tiny.pdf"} {
		raw := readFixture(t, name)
		s.vault.File(path, raw)
		s.texts.File(path, []byte(raw))
	}
	documents := document.NewReader()
	run, err := recognition.New(
		recognition.Host{Vaults: func(id domain.VaultID) recognition.Files { return s.vault.With(id) }, Store: foldersOf(s.store), Texts: s.textsOf, Queue: s.queue, Documents: documents},
		func(context.Context) (recognition.Recognizer, error) { return s.recognizer, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	compilation := wasm.NewCompilationCache(os.Getenv(cacheEnv))
	t.Cleanup(func() {
		if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	documents.SetCompilationCache(compilation)
	s.run = run
	return s
}

func (s *scene) readLiveText(t *testing.T, path string) domain.VersionText {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), path)
	if err != nil || listing.Live == "" {
		t.Fatalf("%s has no live version: %+v (%v)", path, listing, err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			text, err := host.ReadWholeText(t.Context(), s.texts, version)
			if err != nil {
				t.Fatal(err)
			}
			return text
		}
	}
	t.Fatalf("the live version %s is not listed", listing.Live)
	return domain.VersionText{}
}

func TestRecognizePDFWritesPositionedText(t *testing.T) {
	s := newScene(t)

	res, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"})
	if err != nil || res.Pages != 2 || res.IsEmpty {
		t.Fatalf("recognized %+v (%v), want two pages", res, err)
	}
	text := s.readLiveText(t, "books/scan.pdf")
	if text.Version().Kind != "recognition" || text.Version().Key != recognition.VersionKey {
		t.Fatalf("the version is %+v", text.Version())
	}
	if !strings.Contains(text.Text, "page two\n\nHeading") {
		t.Errorf("the text reads %q, want the pages in order with their blocks apart", text.Text)
	}
	offset := strings.LastIndex(text.Text, "page two")
	var selector struct {
		Page  int          `json:"page"`
		Boxes [][4]float32 `json:"boxes"`
	}
	err = json.Unmarshal([]byte(text.GetSelector(offset)), &selector)
	if err != nil || selector.Page != 2 || len(selector.Boxes) != 1 || selector.Boxes[0][1] != 0.5 || selector.Boxes[0][3] != 1 {
		t.Errorf("selector of the last line: %+v (%v), want the bottom half of the second page", selector, err)
	}
	if got := text.Locate(offset); got != "Heading" {
		t.Errorf("location of the last line: %q", got)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a finished recognition still shows %+v", tasks)
	}
}

func TestRecognizeReadsAPDFWithATextLayer(t *testing.T) {
	s := newScene(t)

	res, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/layer.pdf"})
	if err != nil || res.Pages == 0 || s.recognizer.getPageCount() != res.Pages {
		t.Fatalf("a PDF with a layer: %+v (%v), %d pages read", res, err, s.recognizer.getPageCount())
	}
}

func TestRecognizeReadsARecognizedFileAgain(t *testing.T) {
	s := newScene(t)
	for range 2 {
		if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.recognizer.getPageCount(); got != 4 {
		t.Fatalf("%d pages were read, want the file read twice", got)
	}
}

func (s *scene) getStoreNames(t *testing.T) []string {
	t.Helper()
	files, err := s.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.Name
	}
	return names
}

func TestRecognizeKeepsOnlyTheVersionFile(t *testing.T) {
	s := newScene(t)

	if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}

	text := s.readLiveText(t, "books/scan.pdf")
	if names := s.getStoreNames(t); len(names) != 1 || names[0] != text.Version().File || !strings.HasSuffix(names[0], ".jsonl") {
		t.Errorf("the folder holds %v, want only the version file %s", names, text.Version().File)
	}
}

func TestRecognizeReadsAgainOnceTheVersionIsRemoved(t *testing.T) {
	s := newScene(t)
	if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}
	if err := s.texts.RemoveVersion(t.Context(), "books/scan.pdf", recognition.VersionKey); err != nil {
		t.Fatal(err)
	}

	res, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"})

	if err != nil || s.recognizer.getPageCount() != 4 {
		t.Fatalf("recognized %+v (%v) after %d pages, want the file read again", res, err, s.recognizer.getPageCount())
	}
}

func TestRecognizeResumesFromTheProgressRecord(t *testing.T) {
	s := newScene(t)
	raw := readFixture(t, "scan.pdf")
	sum := sha256.Sum256([]byte("books/scan.pdf"))
	progress := `{"path":"books/scan.pdf","inputHash":"` + domain.HashContent([]byte(raw)) +
		`","recipe":"a test|layout|lines|72","pageCount":2,"pages":[[{"text":"kept from before\n\n"}]]}`
	if err := s.store.Write(t.Context(), hex.EncodeToString(sum[:])+".progress.json", []byte(progress)); err != nil {
		t.Fatal(err)
	}

	res, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"})

	if err != nil || res.Pages != 2 || s.recognizer.getPageCount() != 1 {
		t.Fatalf("recognized %+v (%v) after %d pages, want only the second page read", res, err, s.recognizer.getPageCount())
	}
	if text := s.readLiveText(t, "books/scan.pdf"); !strings.HasPrefix(text.Text, "kept from before\n\nHeading") {
		t.Errorf("the text reads %q, want the kept page first", text.Text)
	}
	if names := s.getStoreNames(t); len(names) != 1 {
		t.Errorf("the folder holds %v, want the progress deleted", names)
	}
}

func TestRecognizeEmptyScanKeepsNothing(t *testing.T) {
	s := newScene(t)
	s.recognizer.says = ""

	res, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"})

	if err != nil || !res.IsEmpty {
		t.Fatalf("recognized %+v (%v), want nothing", res, err)
	}
	listing, err := s.texts.ListVersions(t.Context(), "books/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Errorf("a scan that says nothing has versions %+v", listing.Versions)
	}
	files, err := s.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("a scan that says nothing keeps %+v", files)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].FailureReason != "books/scan.pdf says nothing that could be read, and nothing was kept" {
		t.Errorf("tasks %+v, want the task failed with what was read", tasks)
	}
}

func TestRecognizeFailureReportsTaskError(t *testing.T) {
	s := newScene(t)
	s.recognizer.err = errors.New("the model ran out of memory")

	if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err == nil {
		t.Fatal("a recognizer that failed recognized the file")
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "out of memory") || !tasks[0].IsUserStarted {
		t.Fatalf("tasks %+v, want the failure pageCount", tasks)
	}

	s.recognizer.err = nil
	if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %+v after the file was recognized", tasks)
	}
}

func TestRecognizeMissingFileFails(t *testing.T) {
	s := newScene(t)
	if _, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/gone.pdf"}); !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("a file that is not there answered %v", err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %+v for a file that is not there", tasks)
	}
}

func TestRecognizeFileDeletedDuringRecognitionReportsNoTask(t *testing.T) {
	s := newScene(t)
	s.recognizer.onRecognize = func() { s.vault.Remove("books/scan.pdf") }

	_, err := s.run.Recognize(t.Context(), recognition.Request{Path: "books/scan.pdf"})

	if !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("a file deleted during its recognition answered %v", err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %+v after the file was deleted", tasks)
	}
}

func TestNewRequiresHostAndRecognizer(t *testing.T) {
	v := hosttest.New("Physics")
	open := func(context.Context) (recognition.Recognizer, error) { return &pages{}, nil }
	texts := hosttest.NewTexts("Physics", pluginID)
	full := recognition.Host{
		Vaults: func(id domain.VaultID) recognition.Files { return v.With(id) },
		Store:  foldersOf(v.Store()), Texts: func(id domain.VaultID) host.TextVersions { return texts.With(id) }, Queue: hosttest.NewQueue(), Documents: document.NewReader()}
	for name, h := range map[string]recognition.Host{
		"no vault":     {Store: full.Store, Texts: full.Texts, Queue: full.Queue, Documents: full.Documents},
		"no store":     {Vaults: full.Vaults, Texts: full.Texts, Queue: full.Queue, Documents: full.Documents},
		"no texts":     {Vaults: full.Vaults, Store: full.Store, Queue: full.Queue, Documents: full.Documents},
		"no queue":     {Vaults: full.Vaults, Store: full.Store, Texts: full.Texts, Documents: full.Documents},
		"no documents": {Vaults: full.Vaults, Store: full.Store, Texts: full.Texts, Queue: full.Queue},
	} {
		if _, err := recognition.New(h, open); err == nil {
			t.Errorf("%s: a recognition was made", name)
		}
	}
	if _, err := recognition.New(full, nil); err == nil {
		t.Error("a recognition was made with nothing to recognize with")
	}
}

func TestRecognizeRecordsTheVersionInTheVaultTheRequestNames(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	raw := readFixture(t, "scan.pdf")
	chemistry.File("books/scan.pdf", raw)
	other := s.texts.With("Chemistry")
	other.SetStore(pluginID, chemistry.Store())
	other.File("books/scan.pdf", []byte(raw))

	if _, err := s.run.Recognize(t.Context(), recognition.Request{Vault: "Chemistry", Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}

	inChemistry, err := other.ListVersions(t.Context(), "books/scan.pdf")
	if err != nil || inChemistry.Live == "" {
		t.Fatalf("Chemistry lists %+v (%v), want the recognition live", inChemistry, err)
	}
	inPhysics, err := s.texts.ListVersions(t.Context(), "books/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(inPhysics.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inPhysics.Versions)
	}
	names, err := s.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("the shown vault's folder holds %v, want nothing", names)
	}
}

func TestRecognizeInTwoVaultsOnTheSamePathReportsTwoTasks(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	raw := readFixture(t, "scan.pdf")
	chemistry.File("books/scan.pdf", raw)
	other := s.texts.With("Chemistry")
	other.SetStore(pluginID, chemistry.Store())
	other.File("books/scan.pdf", []byte(raw))

	sum := sha256.Sum256([]byte("books/scan.pdf"))
	progress := `{"path":"books/scan.pdf","inputHash":"` + domain.HashContent([]byte(raw)) +
		`","recipe":"a test|layout|lines|72","pageCount":2,"pages":[[{"text":"kept\n\n"}]]}`
	if err := chemistry.Store().Write(t.Context(), hex.EncodeToString(sum[:])+".progress.json", []byte(progress)); err != nil {
		t.Fatal(err)
	}

	firstPage := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	s.recognizer.onRecognize = func() {
		once.Do(func() {
			close(firstPage)
			<-release
		})
	}

	physicsErr := make(chan error, 1)
	go func() {
		_, err := s.run.Recognize(t.Context(), recognition.Request{Vault: "Physics", Path: "books/scan.pdf"})
		physicsErr <- err
	}()
	<-firstPage

	if _, err := s.run.Recognize(t.Context(), recognition.Request{Vault: "Physics", Path: "books/scan.pdf"}); !errors.Is(err, recognition.ErrAlreadyRecognizing) {
		t.Fatalf("second run in Physics returned %v, want %v", err, recognition.ErrAlreadyRecognizing)
	}

	chemErr := make(chan error, 1)
	go func() {
		_, err := s.run.Recognize(t.Context(), recognition.Request{Vault: "Chemistry", Path: "books/scan.pdf"})
		chemErr <- err
	}()

	physicsID := recognition.GetTaskID("Physics", "books/scan.pdf")
	chemID := recognition.GetTaskID("Chemistry", "books/scan.pdf")
	deadline := time.After(30 * time.Second)
	for {
		byID := map[string]host.Task{}
		for _, task := range s.queue.ListTasks() {
			byID[task.ID] = task
		}
		if p, okP := byID[physicsID]; okP {
			if c, okC := byID[chemID]; okC {
				if p.CompletedCount != 0 || p.TotalCount != 2 || c.CompletedCount != 1 || c.TotalCount != 2 {
					t.Errorf("tasks %+v and %+v, want 0/2 in Physics and 1/2 in Chemistry", p, c)
				}
				break
			}
		}
		select {
		case <-deadline:
			t.Fatalf("tasks %+v, want both %s and %s", s.queue.ListTasks(), physicsID, chemID)
		case <-time.After(10 * time.Millisecond):
		}
	}

	close(release)
	if err := <-physicsErr; err != nil {
		t.Fatalf("Physics failed: %v", err)
	}
	if err := <-chemErr; err != nil {
		t.Fatalf("Chemistry failed: %v", err)
	}
}

func TestStartReportsTaskBeforeReturning(t *testing.T) {
	s := newScene(t)
	release := make(chan struct{})
	var once sync.Once
	s.recognizer.onRecognize = func() {
		once.Do(func() { <-release })
	}

	done, started, err := s.run.Start(t.Context(), recognition.Request{Path: "books/scan.pdf"})
	if err != nil || !started {
		t.Fatalf("Start returned (%v, %v)", started, err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 1 || tasks[0].Detail != "books/scan.pdf" {
		t.Errorf("tasks right after Start = %+v, want the task reported", tasks)
	}
	close(release)
	<-done
}
