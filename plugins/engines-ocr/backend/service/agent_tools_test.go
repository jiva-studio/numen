package service

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/launch"
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
// first page it reads closes startCh.
type blockingRecogniser struct {
	engine.MockRecogniser
	startCh   chan struct{}
	startOnce *sync.Once
	release   chan struct{}
	failure   error
}

func (r blockingRecogniser) Recognize(ctx context.Context, page image.Image) ([]recognition.Block, error) {
	r.startOnce.Do(func() { close(r.startCh) })
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if r.failure != nil {
		return nil, r.failure
	}
	return r.MockRecogniser.Recognize(ctx, page)
}

// recordingQueue is the queue in memory that sends the id of every task that ends on doneCh, and
// counts the tasks reported and the turns requested in calls.
type recordingQueue struct {
	*hosttest.Queue
	doneCh chan string
	calls  *queueCalls
}

func (q recordingQueue) FinishTask(ctx context.Context, id string) error {
	err := q.Queue.FinishTask(ctx, id)
	q.doneCh <- id
	return err
}

// queueCalls counts the requests the runs made to the queue.
type queueCalls struct {
	mu       sync.Mutex
	reported int
	turns    int
}

func (c *queueCalls) getCounts() (reported, turns int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reported, c.turns
}

func (q recordingQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	q.calls.mu.Lock()
	q.calls.turns++
	q.calls.mu.Unlock()
	return q.Queue.EnqueueWork(ctx, work)
}

func (q recordingQueue) ReportTask(ctx context.Context, task host.Task) error {
	q.calls.mu.Lock()
	q.calls.reported++
	q.calls.mu.Unlock()
	return q.Queue.ReportTask(ctx, task)
}

func foldersOf(store *hosttest.Store) func(domain.VaultID) recognition.Folder {
	return func(id domain.VaultID) recognition.Folder { return store.With(id) }
}

func textsOf(texts *hosttest.Texts) func(domain.VaultID) host.TextVersions {
	return func(id domain.VaultID) host.TextVersions { return texts.With(id) }
}

// toolFixture is the recognition tool over the open vault Physics with the scan, and the vault
// Chemistry with a scan of its own.
type toolFixture struct {
	vault        *hosttest.Vault
	texts        *hosttest.Texts
	chemistry    *hosttest.Vault
	chemTexts    *hosttest.Texts
	queue        recordingQueue
	recogniser   blockingRecogniser
	openCount    atomic.Int32
	recognitions *recognition.Recognition
	launcher     *launch.Launcher
	registry     *tools.Registry
}

// chemistryScan is the scan stored only in the vault Chemistry.
const chemistryScan = "papers/titration.pdf"

func newToolFixture(t *testing.T) *toolFixture {
	t.Helper()
	raw, err := os.ReadFile("../document/testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	vault := hosttest.New("Physics")
	vault.File(scanPath, string(raw))
	texts := hosttest.NewTexts("Physics", PluginID)
	texts.SetStore(PluginID, vault.Store())
	texts.File(scanPath, raw)

	chemistry := hosttest.New("Chemistry")
	vault.Beside(chemistry)
	chemistry.File(chemistryScan, string(raw))
	chemTexts := texts.With("Chemistry")
	chemTexts.SetStore(PluginID, chemistry.Store())
	chemTexts.File(chemistryScan, raw)

	s := &toolFixture{
		vault: vault, texts: texts, chemistry: chemistry, chemTexts: chemTexts,
		queue: recordingQueue{Queue: hosttest.NewQueue(), doneCh: make(chan string, 64), calls: &queueCalls{}},
		recogniser: blockingRecogniser{
			startCh: make(chan struct{}), startOnce: &sync.Once{}, release: make(chan struct{}),
		},
	}
	recognitions, err := recognition.New(
		recognition.Host{Vaults: func(id domain.VaultID) recognition.Files { return vault.With(id) }, Store: foldersOf(vault.Store()), Texts: textsOf(texts), Queue: s.queue},
		func(context.Context) (recognition.Recogniser, error) {
			s.openCount.Add(1)
			return s.recogniser, nil
		},
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
	s.recognitions = recognitions
	s.launcher = launch.New(recognitions, PluginID)
	t.Cleanup(s.launcher.Close)
	s.registry, err = newToolRegistry(s.launcher)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

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
		t.Fatalf("%s did not return within %v: the call waits for the recognition", call.Name, callDeadline)
		return nil
	}
}

func (s *toolFixture) recognize(t *testing.T, vault, path string) RecognitionCreateOutput {
	t.Helper()
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: vault, Name: "recognition_create", Arguments: []byte(`{"path":"` + path + `"}`),
	})
	if res.GetResult().GetIsError() {
		t.Fatalf("recognition_create of %s in %s was rejected: %s", path, vault, res.GetResult().GetText())
	}
	var out RecognitionCreateOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func (s *toolFixture) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.recogniser.startCh:
	case <-time.After(workDeadline):
		t.Fatalf("no page was read within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}
}

func (s *toolFixture) waitFinished(t *testing.T, id string) {
	t.Helper()
	deadline := time.After(workDeadline)
	for {
		select {
		case got := <-s.queue.doneCh:
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
	s := newToolFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	infos := listing.Tools
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

func TestRecognitionCreateReturnsBeforeTheRecognitionEnds(t *testing.T) {
	s := newToolFixture(t)

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "started"}
	if out != want {
		t.Fatalf("recognition_create returned %+v, want %+v", out, want)
	}
	s.waitStarted(t)
	if tasks := s.queue.ListTasks(); !isListed(tasks, want.TaskID) {
		t.Errorf("the tasks %+v do not list %s while the recognition runs", tasks, want.TaskID)
	}
	if hasRecognition(t, s.texts, scanPath) {
		t.Error("the text version exists before the recognition ended")
	}

	close(s.recogniser.release)
	s.waitFinished(t, want.TaskID)
	s.launcher.Close()

	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the recognition ended and the scan has no text version")
	}
}

func TestRecognitionCreateOfAScanBeingRecognisedReportsItRunning(t *testing.T) {
	s := newToolFixture(t)
	first := s.recognize(t, "Physics", scanPath)
	s.waitStarted(t)

	second := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: first.TaskID, Path: scanPath, State: "running"}
	if second != want {
		t.Errorf("the second call returned %+v, want %+v", second, want)
	}
	close(s.recogniser.release)
	s.waitFinished(t, first.TaskID)
}

func TestRecognitionCreateRejectsAFileNotInTheVault(t *testing.T) {
	s := newToolFixture(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"books/missing.pdf"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a file that is not there returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if text := res.GetResult().GetText(); !strings.Contains(text, "not found") || !strings.Contains(text, "books/missing.pdf") {
		t.Errorf("the error reads %q, want it to state that books/missing.pdf is not found", text)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a rejected call shows the tasks %+v", tasks)
	}
}

func (s *toolFixture) putRecognition(t *testing.T) domain.TextVersion {
	t.Helper()
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	const file = "scan-recognised.jsonl"
	segments := []domain.TextSegment{{Text: "Ohm measured the current through a copper wire.\n"}}
	if err := host.WriteText(t.Context(), s.vault.Store(), file, segments); err != nil {
		t.Fatal(err)
	}
	version, err := s.texts.PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: recognition.VersionKey, Source: "", Kind: "recognition",
		InputHash: domain.HashContent(raw), File: file,
	})
	if err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	if listing.Live != version.GetID() {
		t.Fatalf("the scan lists %+v, want the recognition %s live", listing, version.GetID())
	}
	return version
}

func (s *toolFixture) checkNoRun(t *testing.T) {
	t.Helper()
	close(s.recogniser.release)
	s.launcher.Close()
	if reported, turns := s.queue.calls.getCounts(); reported != 0 || turns != 0 {
		t.Errorf("a run reported %d tasks and requested %d turns, want none", reported, turns)
	}
	if ended := len(s.queue.doneCh); ended != 0 {
		t.Errorf("%d tasks ended, want none", ended)
	}
	if count := s.openCount.Load(); count != 0 {
		t.Errorf("the recogniser was opened %d times, want never", count)
	}
}

func TestRecognitionCreateOfRecognisedScanReturnsItsVersion(t *testing.T) {
	s := newToolFixture(t)
	version := s.putRecognition(t)

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestRecognitionCreateAfterFinishedRunReturnsItsVersion(t *testing.T) {
	s := newToolFixture(t)
	close(s.recogniser.release)
	if _, err := s.recognitions.Recognize(t.Context(), recognition.Request{
		Vault: "Physics", Path: scanPath,
	}); err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.Key == recognition.VersionKey })
	if i < 0 || listing.Live != listing.Versions[i].GetID() {
		t.Fatalf("the scan lists %+v, want the recognition live", listing)
	}

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "done", Version: string(listing.Live),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
}

// proofreaderPluginID is the plugin that makes versions from other versions in the chain tests.
const proofreaderPluginID = "md.numen.tools.proofreading"

func (s *toolFixture) putProofread(t *testing.T, source domain.TextVersion) domain.TextVersion {
	t.Helper()
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	const file = "proofread-1.jsonl"
	segments := []domain.TextSegment{{Text: "Ohm measured the current through a copper wire, twice.\n"}}
	if err := host.WriteText(t.Context(), s.texts.GetStore(proofreaderPluginID), file, segments); err != nil {
		t.Fatal(err)
	}
	params := host.TextPut{
		Path: scanPath, Key: "proofread-1", Source: "", Kind: "recognition",
		InputHash: domain.HashContent(raw), File: file,
	}
	if source != (domain.TextVersion{}) {
		params.Source, params.InputHash = source.GetID(), source.TextHash
	}
	version, err := s.texts.As(proofreaderPluginID).PutVersion(t.Context(), params)
	if err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == version.GetID() })
	if i < 0 || listing.Live != version.GetID() || listing.Versions[i].Source != params.Source ||
		listing.Versions[i].PluginID != proofreaderPluginID {
		t.Fatalf("the scan lists %+v, want %s live, made from %q", listing, version.GetID(), params.Source)
	}
	return version
}

func TestRecognitionCreateOfLiveProofreadReturnsItWithoutARun(t *testing.T) {
	s := newToolFixture(t)
	recognized := s.putRecognition(t)
	proofread := s.putProofread(t, recognized)

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "done", Version: string(proofread.GetID()),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestRecognitionCreateOfALiveProofreadOfTheScanStartsTheRun(t *testing.T) {
	s := newToolFixture(t)
	s.putRecognition(t)
	s.putProofread(t, domain.TextVersion{})

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "started"}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	close(s.recogniser.release)
	s.launcher.Close()
}

func TestRecognitionCreateRejectsACallWithoutAVault(t *testing.T) {
	s := newToolFixture(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "", Name: "recognition_create", Arguments: []byte(`{"path":"` + scanPath + `"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a call with no vault returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestRecognitionCreateRecognisesInTheNamedVault(t *testing.T) {
	s := newToolFixture(t)

	inShown := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"` + chemistryScan + `"}`),
	})
	if !inShown.GetResult().GetIsError() {
		t.Errorf("a scan only Chemistry has was accepted in Physics: %s", inShown.GetResult().GetStructured())
	}

	out := s.recognize(t, "Chemistry", chemistryScan)
	if out.State != "started" || out.TaskID != "recognize:Chemistry:"+chemistryScan {
		t.Fatalf("recognition_create in Chemistry returned %+v", out)
	}
	close(s.recogniser.release)
	s.waitFinished(t, out.TaskID)
	s.launcher.Close()

	if !hasRecognition(t, s.chemTexts, chemistryScan) {
		t.Error("Chemistry has no text version of its scan")
	}
	if hasRecognition(t, s.texts, chemistryScan) {
		t.Error("the shown vault has a text version of the scan of Chemistry")
	}
}

func TestRecognitionCreateStartsRunsInTwoVaultsOnTheSamePath(t *testing.T) {
	s := newToolFixture(t)
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	s.chemistry.File(scanPath, string(raw))
	s.chemTexts.File(scanPath, raw)

	first := s.recognize(t, "Physics", scanPath)
	if want := (RecognitionCreateOutput{TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "started"}); first != want {
		t.Fatalf("first call returned %+v, want %+v", first, want)
	}
	s.waitStarted(t)

	again := s.recognize(t, "Physics", scanPath)
	if want := (RecognitionCreateOutput{TaskID: first.TaskID, Path: scanPath, State: "running"}); again != want {
		t.Errorf("second call in Physics returned %+v, want %+v", again, want)
	}

	second := s.recognize(t, "Chemistry", scanPath)
	if want := (RecognitionCreateOutput{TaskID: "recognize:Chemistry:" + scanPath, Path: scanPath, State: "started"}); second != want {
		t.Fatalf("call in Chemistry returned %+v, want %+v", second, want)
	}
	deadline := time.After(workDeadline)
	for !isListed(s.queue.ListTasks(), second.TaskID) {
		select {
		case <-deadline:
			t.Fatalf("the task %s was not listed within %v; tasks %+v", second.TaskID, workDeadline, s.queue.ListTasks())
		case <-time.After(10 * time.Millisecond):
		}
	}
	if tasks := s.queue.ListTasks(); !isListed(tasks, first.TaskID) || !isListed(tasks, second.TaskID) {
		t.Errorf("tasks %+v, want both %s and %s", tasks, first.TaskID, second.TaskID)
	}

	close(s.recogniser.release)
	s.waitFinished(t, first.TaskID)
	s.waitFinished(t, second.TaskID)
	s.launcher.Close()

	if !hasRecognition(t, s.texts, scanPath) || !hasRecognition(t, s.chemTexts, scanPath) {
		t.Error("one of the two vaults has no recognition after both runs ended")
	}
}

func TestTheRecognitionToolHasItsInstructions(t *testing.T) {
	s := newToolFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	sentences := getToolSentences(listing, "recognition_create")
	if len(sentences) == 0 {
		t.Fatal("recognition_create lists no instructions")
	}
	if !strings.Contains(strings.Join(sentences, " "), "recognition_create") {
		t.Errorf("no instruction of recognition_create names it: %q", sentences)
	}
	for _, sentence := range sentences {
		if !isOneSentence(sentence) {
			t.Errorf("an instruction of recognition_create is not one sentence: %q", sentence)
		}
	}
}

// readCounts counts the reads the plugin makes of the scan, through the vault, and of version files,
// through its own folder or the host's texts.
type readCounts struct {
	mu       sync.Mutex
	scan     int
	versions int
}

func (c *readCounts) addScan() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scan++
}

func (c *readCounts) addVersion() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.versions++
}

func (c *readCounts) getCounts() (scan, versions int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scan, c.versions
}

// countingVault is the vault whose file reads are counted as reads of the scan.
type countingVault struct {
	*hosttest.Vault
	counts *readCounts
}

func (v countingVault) With(id domain.VaultID) countingVault {
	return countingVault{Vault: v.Vault.With(id), counts: v.counts}
}

func (v countingVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	v.counts.addScan()
	return v.Vault.Read(ctx, path)
}

// countingStore is the plugin's folder, which has its version files, with each read counted.
type countingStore struct {
	recognition.Folder
	counts *readCounts
}

func (s countingStore) Read(ctx context.Context, name string) ([]byte, error) {
	s.counts.addVersion()
	return s.Folder.Read(ctx, name)
}

func (s *toolFixture) countReads(t *testing.T) *readCounts {
	t.Helper()
	counts := &readCounts{}
	counted := countingVault{Vault: s.vault, counts: counts}
	recognitions, err := recognition.New(
		recognition.Host{
			Vaults: func(id domain.VaultID) recognition.Files { return counted.With(id) }, Store: func(id domain.VaultID) recognition.Folder {
				return countingStore{Folder: s.vault.Store().With(id), counts: counts}
			},
			Texts: textsOf(s.texts), Queue: s.queue,
		},
		func(context.Context) (recognition.Recogniser, error) {
			s.openCount.Add(1)
			return s.recogniser, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	s.recognitions = recognitions
	s.launcher = launch.New(recognitions, PluginID)
	t.Cleanup(s.launcher.Close)
	s.registry, err = newToolRegistry(s.launcher)
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestRecognitionCreateOfARecognisedScanReadsNoFile(t *testing.T) {
	s := newToolFixture(t)
	version := s.putRecognition(t)
	counts := s.countReads(t)

	out := s.recognize(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognize:Physics:" + scanPath, Path: scanPath, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
	if scan, versions := counts.getCounts(); scan != 0 || versions != 0 {
		t.Errorf("recognition_create read the scan %d times and version files %d times, want no read",
			scan, versions)
	}
}
