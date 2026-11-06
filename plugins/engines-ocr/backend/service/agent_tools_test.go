package service

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

const (
	// scanPath is the two-page scan of the shown vault.
	scanPath = "books/scan.pdf"
	// callDeadline is how long a tool call may take before the test fails it as waiting for the work.
	callDeadline = 30 * time.Second
	// workDeadline is how long a recognition may run before the test fails.
	workDeadline = 5 * time.Minute
)

// blockingRecogniser reads pages as the mock does, but each page waits until release is closed. The
// first page it is asked for closes entered.
type blockingRecogniser struct {
	engine.MockRecogniser
	entered   chan struct{}
	enterOnce *sync.Once
	release   chan struct{}
}

func (r blockingRecogniser) Recognise(ctx context.Context, page image.Image) ([]recognition.Block, error) {
	r.enterOnce.Do(func() { close(r.entered) })
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return r.MockRecogniser.Recognise(ctx, page)
}

// finishingQueue is the queue in memory that sends the id of every task that ends on finished.
type finishingQueue struct {
	*hosttest.Queue
	finished chan string
}

func (q finishingQueue) FinishTask(ctx context.Context, id string) error {
	err := q.Queue.FinishTask(ctx, id)
	q.finished <- id
	return err
}

// toolScene is the recognition tool over the vault Physics, shown, holding the scan, and the vault
// Chemistry beside it, holding a scan of its own.
type toolScene struct {
	vault      *hosttest.Vault
	texts      *hosttest.Texts
	chemistry  *hosttest.Vault
	chemTexts  *hosttest.Texts
	queue      finishingQueue
	recogniser blockingRecogniser
	tools      *Tools
	registry   *tools.Registry
}

// chemistryScan is the scan held only by the vault Chemistry.
const chemistryScan = "papers/titration.pdf"

func newToolScene(t *testing.T) *toolScene {
	t.Helper()
	raw, err := os.ReadFile("../document/testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	vault := hosttest.New("Physics")
	vault.File(scanPath, string(raw))
	texts := hosttest.NewTexts("Physics", PluginID)
	texts.SetStore(PluginID, vault.Store())

	chemistry := hosttest.New("Chemistry")
	vault.Beside(chemistry)
	chemistry.File(chemistryScan, string(raw))
	chemTexts, isTexts := texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of Chemistry are not hosttest texts")
	}
	chemTexts.SetStore(PluginID, chemistry.Store())

	s := &toolScene{
		vault: vault, texts: texts, chemistry: chemistry, chemTexts: chemTexts,
		queue: finishingQueue{Queue: hosttest.NewQueue(), finished: make(chan string, 64)},
		recogniser: blockingRecogniser{
			entered: make(chan struct{}), enterOnce: &sync.Once{}, release: make(chan struct{}),
		},
	}
	recognitions, err := recognition.New(
		recognition.Host{Vault: vault, Store: vault.Store(), Texts: texts, Queue: s.queue},
		func(context.Context) (recognition.Recogniser, error) { return s.recogniser, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("NUMEN_TEST_WASM_CACHE"); dir != "" {
		compilation := wasm.NewCompilationCache(dir)
		t.Cleanup(func() {
			if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
				t.Error(err)
			}
		})
		recognitions.SetCompilationCache(compilation)
	}
	s.tools = NewTools(recognitions)
	t.Cleanup(s.tools.Close)
	s.registry, err = s.tools.Registry()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// invokeAtOnce calls the tool and fails the test when the call does not answer within callDeadline.
func invokeAtOnce(t *testing.T, registry *tools.Registry, call tools.Call) *tools.CallResponse {
	t.Helper()
	type answer struct {
		res *tools.CallResponse
		err error
	}
	responses := make(chan answer, 1)
	go func() {
		res, err := registry.InvokeTool(t.Context(), call)
		responses <- answer{res: res, err: err}
	}()
	select {
	case got := <-responses:
		if got.err != nil {
			t.Fatalf("call %s: %v", call.Name, got.err)
		}
		return got.res
	case <-time.After(callDeadline):
		t.Fatalf("%s did not answer within %v: the call waits for the recognition", call.Name, callDeadline)
		return nil
	}
}

func (s *toolScene) recognise(t *testing.T, vault, path string) RecognitionCreateOutput {
	t.Helper()
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: vault, Name: "recognition_create", Arguments: []byte(`{"path":"` + path + `"}`),
	})
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("recognition_create of %s in %s was refused: %s", path, vault, res.GetResult().GetText())
	}
	var out RecognitionCreateOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func (s *toolScene) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-s.recogniser.entered:
	case <-time.After(workDeadline):
		t.Fatalf("no page was read within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}
}

// waitFinished waits for the task of the id to end.
func (s *toolScene) waitFinished(t *testing.T, id string) {
	t.Helper()
	deadline := time.After(workDeadline)
	for {
		select {
		case got := <-s.queue.finished:
			if got == id {
				return
			}
		case <-deadline:
			t.Fatalf("the task %s did not end within %v; tasks %+v", id, workDeadline, s.queue.ListTasks())
		}
	}
}

func hasRecognition(t *testing.T, texts *hosttest.Texts, path string) bool {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return slices.ContainsFunc(listing.Versions, func(v domain.TextVersion) bool { return v.Key == recognition.VersionKey })
}

func isListed(tasks []host.Task, id string) bool {
	return slices.ContainsFunc(tasks, func(task host.Task) bool { return task.ID == id })
}

func TestRegistryListsRecognitionCreateOnTheFullServerOnly(t *testing.T) {
	s := newToolScene(t)

	infos, err := s.registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].GetName() != "recognition_create" {
		t.Fatalf("the registry lists %+v, want recognition_create alone", infos)
	}
	if infos[0].GetEffect() != tools.ReadWrite {
		t.Errorf("recognition_create has effect %v, want ReadWrite", infos[0].GetEffect())
	}
	if !slices.Equal(infos[0].GetServers(), []tools.Server{tools.Full}) {
		t.Errorf("recognition_create is listed on %v, want the full server alone", infos[0].GetServers())
	}
}

func TestTheInstructionsAreOneParagraphNamingRecognitionCreate(t *testing.T) {
	if !strings.Contains(Instructions, "recognition_create") {
		t.Errorf("the instructions do not name recognition_create: %q", Instructions)
	}
	if strings.Contains(strings.TrimSpace(Instructions), "\n\n") {
		t.Errorf("the instructions are more than one paragraph: %q", Instructions)
	}
}

func TestRecognitionCreateReturnsBeforeTheRecognitionEnds(t *testing.T) {
	s := newToolScene(t)

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: "recognise:" + scanPath, Path: scanPath, State: "started"}
	if out != want {
		t.Fatalf("recognition_create returned %+v, want %+v", out, want)
	}
	s.waitEntered(t)
	if tasks := s.queue.ListTasks(); !isListed(tasks, want.TaskID) {
		t.Errorf("the tasks %+v do not list %s while the recognition runs", tasks, want.TaskID)
	}
	if hasRecognition(t, s.texts, scanPath) {
		t.Error("the text version exists before the recognition ended")
	}

	close(s.recogniser.release)
	s.waitFinished(t, want.TaskID)
	s.tools.Close()

	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the recognition ended and the scan has no text version")
	}
}

func TestRecognitionCreateOfAScanBeingRecognisedReportsItRunning(t *testing.T) {
	s := newToolScene(t)
	first := s.recognise(t, "Physics", scanPath)
	s.waitEntered(t)

	second := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: first.TaskID, Path: scanPath, State: "running"}
	if second != want {
		t.Errorf("the second call returned %+v, want %+v", second, want)
	}
	close(s.recogniser.release)
	s.waitFinished(t, first.TaskID)
}

func TestRecognitionCreateRefusesAFileNotInTheVault(t *testing.T) {
	s := newToolScene(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"books/missing.pdf"}`),
	})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a file that is not there returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a refused call shows the tasks %+v", tasks)
	}
}

func TestRecognitionCreateRefusesACallThatNamesNoVault(t *testing.T) {
	s := newToolScene(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "", Name: "recognition_create", Arguments: []byte(`{"path":"` + scanPath + `"}`),
	})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a call with no vault returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestRecognitionCreateRecognisesInTheVaultTheCallNames(t *testing.T) {
	s := newToolScene(t)

	inShown := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"` + chemistryScan + `"}`),
	})
	if !inShown.GetResult().GetIsRefusal() {
		t.Errorf("a scan only Chemistry holds was accepted in Physics: %s", inShown.GetResult().GetStructured())
	}

	out := s.recognise(t, "Chemistry", chemistryScan)
	if out.State != "started" || out.TaskID != "recognise:"+chemistryScan {
		t.Fatalf("recognition_create in Chemistry returned %+v", out)
	}
	close(s.recogniser.release)
	s.waitFinished(t, out.TaskID)
	s.tools.Close()

	if !hasRecognition(t, s.chemTexts, chemistryScan) {
		t.Error("Chemistry holds no text version of its scan")
	}
	if hasRecognition(t, s.texts, chemistryScan) {
		t.Error("the shown vault holds a text version of the scan of Chemistry")
	}
}
