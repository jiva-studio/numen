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

// finishingQueue is the queue in memory that sends the id of every task that ends on finished, and
// counts the tasks reported and the turns asked for in calls.
type finishingQueue struct {
	*hosttest.Queue
	finished chan string
	calls    *queueCalls
}

func (q finishingQueue) FinishTask(ctx context.Context, id string) error {
	err := q.Queue.FinishTask(ctx, id)
	q.finished <- id
	return err
}

// queueCalls counts what the runs asked of the queue.
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

func (q finishingQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	q.calls.mu.Lock()
	q.calls.turns++
	q.calls.mu.Unlock()
	return q.Queue.EnqueueWork(ctx, work)
}

func (q finishingQueue) ReportTask(ctx context.Context, task host.Task) error {
	q.calls.mu.Lock()
	q.calls.reported++
	q.calls.mu.Unlock()
	return q.Queue.ReportTask(ctx, task)
}

// toolScene is the recognition tool over the vault Physics, shown, holding the scan, and the vault
// Chemistry beside it, holding a scan of its own.
type toolScene struct {
	vault        *hosttest.Vault
	texts        *hosttest.Texts
	chemistry    *hosttest.Vault
	chemTexts    *hosttest.Texts
	queue        finishingQueue
	recogniser   blockingRecogniser
	openCount    atomic.Int32
	recognitions *recognition.Recognition
	tools        *Tools
	registry     *tools.Registry
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
	texts.File(scanPath, raw)

	chemistry := hosttest.New("Chemistry")
	vault.Beside(chemistry)
	chemistry.File(chemistryScan, string(raw))
	chemTexts, isTexts := texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of Chemistry are not hosttest texts")
	}
	chemTexts.SetStore(PluginID, chemistry.Store())
	chemTexts.File(chemistryScan, raw)

	s := &toolScene{
		vault: vault, texts: texts, chemistry: chemistry, chemTexts: chemTexts,
		queue: finishingQueue{Queue: hosttest.NewQueue(), finished: make(chan string, 64), calls: &queueCalls{}},
		recogniser: blockingRecogniser{
			entered: make(chan struct{}), enterOnce: &sync.Once{}, release: make(chan struct{}),
		},
	}
	recognitions, err := recognition.New(
		recognition.Host{Vault: vault, Store: vault.Store(), Texts: texts, Queue: s.queue},
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
	if res.GetResult().GetIsError() {
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

func TestTheInstructionsAreOneParagraphNamingRecognitionCreate(t *testing.T) {
	if !strings.Contains(instructions, "recognition_create") {
		t.Errorf("the instructions do not name recognition_create: %q", instructions)
	}
	if strings.Contains(strings.TrimSpace(instructions), "\n\n") {
		t.Errorf("the instructions are more than one paragraph: %q", instructions)
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

	if !res.GetResult().GetIsError() {
		t.Errorf("a file that is not there returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if text := res.GetResult().GetText(); !strings.Contains(text, "not found") || !strings.Contains(text, "books/missing.pdf") {
		t.Errorf("the refusal reads %q, want it to say books/missing.pdf is not found", text)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a refused call shows the tasks %+v", tasks)
	}
}

// putRecognition keeps a recognised text of the scan in the plugin's folder and records it as the
// scan's version under recognition.VersionKey, as a finished recognition does. It fails the test
// unless the version is live.
func (s *toolScene) putRecognition(t *testing.T) domain.TextVersion {
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

// checkNoRun fails the test where a run of the scan reported a task, asked the queue for a turn,
// ended a task or opened the recogniser. The tools are closed first, so a run that was started has
// ended.
func (s *toolScene) checkNoRun(t *testing.T) {
	t.Helper()
	close(s.recogniser.release)
	s.tools.Close()
	if reported, turns := s.queue.calls.getCounts(); reported != 0 || turns != 0 {
		t.Errorf("a run reported %d tasks and asked for %d turns, want none", reported, turns)
	}
	if ended := len(s.queue.finished); ended != 0 {
		t.Errorf("%d tasks ended, want none", ended)
	}
	if count := s.openCount.Load(); count != 0 {
		t.Errorf("the recogniser was opened %d times, want never", count)
	}
}

func TestRecognitionCreateOfARecognisedScanReturnsItsVersionWithoutARun(t *testing.T) {
	s := newToolScene(t)
	version := s.putRecognition(t)

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognise:" + scanPath, Path: scanPath, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestRecognitionCreateAfterAFinishedRecognitionReturnsItsVersion(t *testing.T) {
	s := newToolScene(t)
	close(s.recogniser.release)
	if _, err := s.recognitions.Recognise(t.Context(), recognition.Request{
		Vault: "Physics", Path: scanPath, IsUserStarted: true,
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

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognise:" + scanPath, Path: scanPath, State: "done", Version: string(listing.Live),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
}

// proofreadingID is the plugin that makes versions from other versions in the chain tests.
const proofreadingID = "md.numen.tools.proofreading"

// putProofread records a proofread text of the scan, made by the plugin proofreadingID from the
// source version, or from the scan itself where source is the zero version. It fails the test unless
// the proofread version is live with the intended source.
func (s *toolScene) putProofread(t *testing.T, source domain.TextVersion) domain.TextVersion {
	t.Helper()
	raw, _, err := s.vault.Read(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	const file = "proofread-1.jsonl"
	segments := []domain.TextSegment{{Text: "Ohm measured the current through a copper wire, twice.\n"}}
	if err := host.WriteText(t.Context(), s.texts.GetStore(proofreadingID), file, segments); err != nil {
		t.Fatal(err)
	}
	params := host.TextPut{
		Path: scanPath, Key: "proofread-1", Source: "", Kind: "recognition",
		InputHash: domain.HashContent(raw), File: file,
	}
	if source != (domain.TextVersion{}) {
		params.Source, params.InputHash = source.GetID(), source.TextHash
	}
	version, err := s.texts.As(proofreadingID).PutVersion(t.Context(), params)
	if err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == version.GetID() })
	if i < 0 || listing.Live != version.GetID() || listing.Versions[i].Source != params.Source ||
		listing.Versions[i].PluginID != proofreadingID {
		t.Fatalf("the scan lists %+v, want %s live, made from %q", listing, version.GetID(), params.Source)
	}
	return version
}

func TestRecognitionCreateOfALiveProofreadOfTheRecognitionReturnsTheProofreadWithoutARun(t *testing.T) {
	s := newToolScene(t)
	recognised := s.putRecognition(t)
	proofread := s.putProofread(t, recognised)

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognise:" + scanPath, Path: scanPath, State: "done", Version: string(proofread.GetID()),
	}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestRecognitionCreateOfALiveProofreadOfTheScanStartsTheRun(t *testing.T) {
	s := newToolScene(t)
	s.putRecognition(t)
	s.putProofread(t, domain.TextVersion{})

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{TaskID: "recognise:" + scanPath, Path: scanPath, State: "started"}
	if out != want {
		t.Errorf("recognition_create returned %+v, want %+v", out, want)
	}
	close(s.recogniser.release)
	s.tools.Close()
}

func TestRecognitionCreateRefusesACallThatNamesNoVault(t *testing.T) {
	s := newToolScene(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "", Name: "recognition_create", Arguments: []byte(`{"path":"` + scanPath + `"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a call with no vault returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestRecognitionCreateRecognisesInTheVaultTheCallNames(t *testing.T) {
	s := newToolScene(t)

	inShown := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"` + chemistryScan + `"}`),
	})
	if !inShown.GetResult().GetIsError() {
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

func TestRegistryListsTheRecognitionInstructions(t *testing.T) {
	s := newToolScene(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions == "" || listing.Instructions != instructions {
		t.Errorf("the registry lists the instructions %q, want %q", listing.Instructions, instructions)
	}
	if !strings.Contains(listing.Instructions, "recognition_create") {
		t.Errorf("the listed instructions do not name recognition_create: %q", listing.Instructions)
	}
	if strings.Contains(strings.TrimSpace(listing.Instructions), "\n\n") {
		t.Errorf("the listed instructions are more than one paragraph: %q", listing.Instructions)
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
	host.Vault
	counts *readCounts
}

func (v countingVault) With(id domain.VaultID) host.Vault {
	return countingVault{Vault: v.Vault.With(id), counts: v.counts}
}

func (v countingVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	v.counts.addScan()
	return v.Vault.Read(ctx, path)
}

// countingStore is the plugin's folder, which holds its version files, with each read counted.
type countingStore struct {
	host.Store
	counts *readCounts
}

func (s countingStore) With(id domain.VaultID) host.Store {
	return countingStore{Store: s.Store.With(id), counts: s.counts}
}

func (s countingStore) Read(ctx context.Context, name string) ([]byte, error) {
	s.counts.addVersion()
	return s.Store.Read(ctx, name)
}

// countingTexts is the host's texts with each read of a version's contents counted.
type countingTexts struct {
	host.Texts
	counts *readCounts
}

func (t countingTexts) With(id domain.VaultID) host.Texts {
	return countingTexts{Texts: t.Texts.With(id), counts: t.counts}
}

func (t countingTexts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error) {
	t.counts.addVersion()
	return t.Texts.ReadText(ctx, path, id, cursor)
}

// countReads gives the scene a recognition and a tool whose every read of the scan and of version
// files is counted. The host's texts keep reading version files to pick the live version; those reads
// are the host's and are not counted.
func (s *toolScene) countReads(t *testing.T) *readCounts {
	t.Helper()
	counts := &readCounts{}
	recognitions, err := recognition.New(
		recognition.Host{
			Vault: countingVault{Vault: s.vault, counts: counts}, Store: countingStore{Store: s.vault.Store(), counts: counts},
			Texts: countingTexts{Texts: s.texts, counts: counts}, Queue: s.queue,
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
	s.tools = NewTools(recognitions)
	t.Cleanup(s.tools.Close)
	s.registry, err = s.tools.Registry()
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestRecognitionCreateOfARecognisedScanReadsNoFile(t *testing.T) {
	s := newToolScene(t)
	version := s.putRecognition(t)
	counts := s.countReads(t)

	out := s.recognise(t, "Physics", scanPath)

	want := RecognitionCreateOutput{
		TaskID: "recognise:" + scanPath, Path: scanPath, State: "done", Version: string(version.GetID()),
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
