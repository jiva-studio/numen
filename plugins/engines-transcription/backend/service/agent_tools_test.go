package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

const (
	// callDeadline is how long a tool call may take before the test fails it as waiting for the work.
	callDeadline = 30 * time.Second
	// workDeadline is how long a transcription may run before the test fails.
	workDeadline = time.Minute
	// chemistryLecture is the recording stored only in the vault Chemistry.
	chemistryLecture = "assets/titration.wav"
	chemistrySound   = "RIFF the bytes of a titration lecture"
)

// blockingModel transcribes recordings as the heard model does, but opening one waits until release is
// closed. The first recording it opens closes startCh.
type blockingModel struct {
	*heardModel
	startCh      chan struct{}
	startOnce    sync.Once
	release      chan struct{}
	onTranscribe func()
}

func (m *blockingModel) Open(ctx context.Context, raw []byte) (engine.Recording, error) {
	m.startOnce.Do(func() { close(m.startCh) })
	select {
	case <-m.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return m.heardModel.Open(ctx, raw)
}

func (m *blockingModel) Transcribe(ctx context.Context, audio engine.Audio) (string, error) {
	if m.onTranscribe != nil {
		m.onTranscribe()
	}
	return m.heardModel.Transcribe(ctx, audio)
}

// recordingQueue is the queue in memory that sends the id of every task that ends on doneCh, and
// counts the tasks reported and the turns requested in calls.
type recordingQueue struct {
	*hosttest.Queue
	doneCh chan string
	failCh chan host.Task
	calls  *queueCalls
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

func (q recordingQueue) FinishTask(ctx context.Context, id string) error {
	err := q.Queue.FinishTask(ctx, id)
	q.doneCh <- id
	return err
}

func (q recordingQueue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	q.calls.mu.Lock()
	q.calls.turns++
	q.calls.mu.Unlock()
	return q.Queue.EnqueueWork(ctx, work)
}

// ReportTask also sends every task reported with a failure on failCh.
func (q recordingQueue) ReportTask(ctx context.Context, task host.Task) error {
	q.calls.mu.Lock()
	q.calls.reported++
	q.calls.mu.Unlock()
	err := q.Queue.ReportTask(ctx, task)
	if task.FailureReason != "" {
		q.failCh <- task
	}
	return err
}

// blockingFiles reads recordings through files, but each read waits until release is closed. The
// first read closes startCh.
type blockingFiles struct {
	files     Files
	startCh   chan struct{}
	startOnce *sync.Once
	release   chan struct{}
}

func (f blockingFiles) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	f.startOnce.Do(func() { close(f.startCh) })
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return f.files.ReadFile(ctx, vault, path)
}

func (f blockingFiles) GetActiveVaultID(ctx context.Context) (domain.VaultID, error) {
	if p, ok := f.files.(activeVaultProvider); ok {
		return p.GetActiveVaultID(ctx)
	}
	return "", nil
}

// failingFiles is the files of a vault whose every read fails, while the vault still lists them.
type failingFiles struct{}

var errUnreadable = errors.New("the disk returned an input/output error")

func (failingFiles) ReadFile(context.Context, domain.VaultID, string) ([]byte, error) {
	return nil, errUnreadable
}

// toolFixture is the transcription tool over the open vault Lectures of the scene, and the vault
// Chemistry with a recording of its own.
type toolFixture struct {
	*scene
	chemistry *hosttest.Vault
	chemTexts *hosttest.Texts
	finishing recordingQueue
	blocking  *blockingModel
	registry  *tools.Registry
}

func newToolFixture(t *testing.T) *toolFixture {
	t.Helper()
	return newToolFixtureWithFiles(t, func(files Files) Files { return files })
}

func newToolFixtureWithFiles(t *testing.T, wrap func(Files) Files) *toolFixture {
	t.Helper()
	s := &toolFixture{scene: newScene(t)}
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	chemistry.File(chemistryLecture, chemistrySound)
	chemTexts := s.texts.With("Chemistry")
	chemTexts.SetStore(PluginID, chemistry.Store())
	chemTexts.File(chemistryLecture, []byte(chemistrySound))
	s.chemistry = chemistry
	s.chemTexts = chemTexts

	s.finishing = recordingQueue{
		Queue: s.queue, doneCh: make(chan string, 64), failCh: make(chan host.Task, 64), calls: &queueCalls{},
	}
	s.blocking = &blockingModel{heardModel: s.model, startCh: make(chan struct{}), release: make(chan struct{})}
	s.transcriber = NewTranscriber(wrap(vaultFiles{vaults: func(id domain.VaultID) VaultReader { return s.vault.With(id) }}), storesOf(s.store), textsOf(s.texts), s.finishing,
		func(context.Context) (engine.TranscriberPort, error) { return s.blocking, nil })
	t.Cleanup(s.transcriber.Close)
	registry, err := NewTools(s.transcriber, func(id domain.VaultID) VaultReader { return s.vault.With(id) }).Registry()
	if err != nil {
		t.Fatal(err)
	}
	s.registry = registry
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
		t.Fatalf("%s did not return within %v: the call waits for the transcription", call.Name, callDeadline)
		return nil
	}
}

func (s *toolFixture) createTranscript(t *testing.T, vault, path string) TranscriptCreateOutput {
	t.Helper()
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: vault, Name: "transcript_create", Arguments: []byte(`{"path":"` + path + `"}`),
	})
	if res.GetResult().GetIsError() {
		t.Fatalf("transcript_create of %s in %s was rejected: %s", path, vault, res.GetResult().GetText())
	}
	var out TranscriptCreateOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func (s *toolFixture) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.blocking.startCh:
	case <-time.After(workDeadline):
		t.Fatalf("no recording was opened within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}
}

func (s *toolFixture) waitFinished(t *testing.T, id string) {
	t.Helper()
	deadline := time.After(workDeadline)
	for {
		select {
		case got := <-s.finishing.doneCh:
			if got == id {
				return
			}
		case <-deadline:
			t.Fatalf("the task %s did not end within %v; tasks %+v", id, workDeadline, s.queue.ListTasks())
		}
	}
}

func isListed(tasks []host.Task, id string) bool {
	return slices.ContainsFunc(tasks, func(task host.Task) bool { return task.ID == id })
}

func TestRegistryListsTranscriptCreateOnTheFullServerOnly(t *testing.T) {
	s := newToolFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	infos := listing.Tools
	if len(infos) != 1 || infos[0].GetName() != "transcript_create" {
		t.Fatalf("the registry lists %+v, want transcript_create alone", infos)
	}
	if infos[0].GetEffect() != tools.ReadWrite {
		t.Errorf("transcript_create has effect %v, want ReadWrite", infos[0].GetEffect())
	}
	if !slices.Equal(infos[0].GetServers(), []tools.Server{tools.Full}) {
		t.Errorf("transcript_create is listed on %v, want the full server alone", infos[0].GetServers())
	}
}

func TestTranscriptCreateReturnsBeforeTheTranscriptionEnds(t *testing.T) {
	s := newToolFixture(t)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	s.waitStarted(t)
	if tasks := s.queue.ListTasks(); !isListed(tasks, want.TaskID) {
		t.Errorf("the tasks %+v do not list %s while the transcription runs", tasks, want.TaskID)
	}

	close(s.blocking.release)
	s.waitFinished(t, want.TaskID)
	s.transcriber.Close()

	if _, text := s.readLive(t); text != "first words\nsecond words\n" {
		t.Errorf("the transcript reads %q", text)
	}
}

func TestTranscriptCreateDuringTranscriptionReportsItRunning(t *testing.T) {
	s := newToolFixture(t)
	first := s.createTranscript(t, "Lectures", lecture)
	s.waitStarted(t)

	second := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: first.TaskID, Path: lecture, State: "running"}
	if second != want {
		t.Errorf("the second call returned %+v, want %+v", second, want)
	}
	close(s.blocking.release)
	s.waitFinished(t, first.TaskID)
}

func TestTranscriptCreateRejectsARecordingNotInTheVault(t *testing.T) {
	s := newToolFixture(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"assets/missing.wav"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a recording that is not there returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if text := res.GetResult().GetText(); !strings.Contains(text, "not found") || !strings.Contains(text, "assets/missing.wav") {
		t.Errorf("the error reads %q, want it to state that assets/missing.wav is not found", text)
	}
}

func TestTranscriptCreateRejectsAMissingNoteAsNotFound(t *testing.T) {
	s := newToolFixture(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"notes.md"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Fatalf("a note that is not there returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	text := res.GetResult().GetText()
	if !strings.Contains(text, "not found") || !strings.Contains(text, "notes.md") {
		t.Errorf("the error reads %q, want it to state that notes.md is not found", text)
	}
	if strings.Contains(text, "not an audio or video recording") {
		t.Errorf("the error reads %q, want no word on the kind of a file that is not there", text)
	}
}

func (s *toolFixture) putTranscript(t *testing.T, isWhole bool) domain.TextVersion {
	t.Helper()
	fileHash := domain.HashContent([]byte(lectureSound))
	key := getTranscriptionKey(fileHash)
	transcriptStore := transcripts{store: s.store}
	cues := []transcript.Cue{{Text: "heard before", From: 0, To: 1000}}
	if err := transcriptStore.appendBatch(t.Context(), key, cues, 1000); err != nil {
		t.Fatal(err)
	}
	file := getPartialName(key)
	if isWhole {
		if err := transcriptStore.finish(t.Context(), key); err != nil {
			t.Fatal(err)
		}
		file = getFileName(key)
	}
	j := &job{
		vault: "Lectures", path: lecture, key: key, fileHash: fileHash,
		store: transcriptStore, texts: s.texts, turn: nil,
	}
	if err := s.transcriber.putVersion(t.Context(), j, file); err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.Key == key && v.PluginID == PluginID })
	if i < 0 || listing.Live != listing.Versions[i].GetID() || listing.Versions[i].File != file {
		t.Fatalf("the lecture lists %+v, want the transcript in %s live", listing, file)
	}
	return listing.Versions[i]
}

func (s *toolFixture) checkNoRun(t *testing.T) {
	t.Helper()
	close(s.blocking.release)
	s.transcriber.Close()
	if reported, turns := s.finishing.calls.getCounts(); reported != 0 || turns != 0 {
		t.Errorf("a run reported %d tasks and requested %d turns, want none", reported, turns)
	}
	if ended := len(s.finishing.doneCh); ended != 0 {
		t.Errorf("%d tasks ended, want none", ended)
	}
	select {
	case <-s.blocking.startCh:
		t.Error("the model opened the recording")
	default:
	}
}

func TestTranscriptCreateOfTranscribedRecordingReturnsVersion(t *testing.T) {
	s := newToolFixture(t)
	version := s.putTranscript(t, true)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{
		TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestTranscriptCreateOfPartlyTranscribedRecordingStartsRun(t *testing.T) {
	s := newToolFixture(t)
	s.putTranscript(t, false)
	close(s.blocking.release)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	s.waitFinished(t, want.TaskID)
	s.transcriber.Close()
	if _, text := s.readLive(t); text != "heard before\nsecond words\n" {
		t.Errorf("the transcript reads %q, want the partial transcript had on", text)
	}
}

func TestTranscriptCreateAfterFinishedRunReturnsItsVersion(t *testing.T) {
	s := newToolFixture(t)
	close(s.blocking.release)
	if err := s.transcriber.Transcribe(t.Context(), "Lectures", lecture); err != nil {
		t.Fatal(err)
	}
	version, _ := s.readLive(t)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{
		TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	if version.File != getFileName(version.Key) {
		t.Errorf("the live version is in %s, want the whole transcript", version.File)
	}
}

// proofreaderPluginID is the plugin that makes versions from other versions in the chain tests.
const proofreaderPluginID = "md.numen.tools.proofreading"

func (s *toolFixture) putProofread(t *testing.T, source domain.TextVersion) domain.TextVersion {
	t.Helper()
	const file = "proofread-1.jsonl"
	segments := []domain.TextSegment{{Text: "Heard before, with the commas put back.\n"}}
	if err := host.WriteText(t.Context(), s.texts.GetStore(proofreaderPluginID), file, segments); err != nil {
		t.Fatal(err)
	}
	params := host.TextPut{
		Path: lecture, Key: "proofread-1", Source: "", Kind: textKind,
		InputHash: domain.HashContent([]byte(lectureSound)), File: file,
	}
	if source != (domain.TextVersion{}) {
		params.Source, params.InputHash = source.GetID(), source.TextHash
	}
	version, err := s.texts.As(proofreaderPluginID).PutVersion(t.Context(), params)
	if err != nil {
		t.Fatal(err)
	}
	listing, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == version.GetID() })
	if i < 0 || listing.Live != version.GetID() || listing.Versions[i].Source != params.Source ||
		listing.Versions[i].PluginID != proofreaderPluginID {
		t.Fatalf("the lecture lists %+v, want %s live, made from %q", listing, version.GetID(), params.Source)
	}
	return version
}

func TestTranscriptCreateOfTranscriptProofreadReturnsItWithoutRun(t *testing.T) {
	s := newToolFixture(t)
	whole := s.putTranscript(t, true)
	proofread := s.putProofread(t, whole)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{
		TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "done", Version: string(proofread.GetID()),
	}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
}

func TestTranscriptCreateOfRecordingProofreadStartsTheRun(t *testing.T) {
	s := newToolFixture(t)
	s.putTranscript(t, true)
	s.putProofread(t, domain.TextVersion{})

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	close(s.blocking.release)
	s.transcriber.Close()
}

func TestTranscriptCreateOfPartialTranscriptProofreadStartsTheRun(t *testing.T) {
	s := newToolFixture(t)
	partial := s.putTranscript(t, false)
	s.putProofread(t, partial)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	close(s.blocking.release)
	s.transcriber.Close()
}

func TestTranscriptCreateRejectsACallWithoutAVault(t *testing.T) {
	s := newToolFixture(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "", Name: "transcript_create", Arguments: []byte(`{"path":"` + lecture + `"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a call with no vault returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestTranscriptCreateTranscribesInTheNamedVault(t *testing.T) {
	s := newToolFixture(t)

	inShown := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"` + chemistryLecture + `"}`),
	})
	if !inShown.GetResult().GetIsError() {
		t.Errorf("a recording only Chemistry has was accepted in Lectures: %s", inShown.GetResult().GetStructured())
	}

	out := s.createTranscript(t, "Chemistry", chemistryLecture)
	if out.State != "started" || out.TaskID != "transcription:Chemistry:"+chemistryLecture {
		t.Fatalf("transcript_create in Chemistry returned %+v", out)
	}
	close(s.blocking.release)
	s.waitFinished(t, out.TaskID)
	s.transcriber.Close()

	inChemistry, err := s.chemTexts.ListVersions(t.Context(), chemistryLecture)
	if err != nil || inChemistry.Live == "" {
		t.Errorf("Chemistry lists %+v (%v), want the transcript live", inChemistry, err)
	}
	inLectures, err := s.texts.ListVersions(t.Context(), chemistryLecture)
	if err != nil {
		t.Fatal(err)
	}
	if len(inLectures.Versions) != 0 {
		t.Errorf("the shown vault has %+v, want no version", inLectures.Versions)
	}
}

func TestTranscriptCreateStartsRunsInTwoVaultsOnTheSamePath(t *testing.T) {
	s := newToolFixture(t)
	s.chemistry.File(lecture, lectureSound)
	s.chemTexts.File(lecture, []byte(lectureSound))
	s.putTranscript(t, false)
	close(s.blocking.release)

	inBatch := make(chan struct{})
	releaseBatch := make(chan struct{})
	var once sync.Once
	s.blocking.onTranscribe = func() {
		once.Do(func() {
			close(inBatch)
			<-releaseBatch
		})
	}

	first := s.createTranscript(t, "Lectures", lecture)
	if want := (TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}); first != want {
		t.Fatalf("first call returned %+v, want %+v", first, want)
	}
	select {
	case <-inBatch:
	case <-time.After(workDeadline):
		t.Fatalf("Lectures did not start transcribing within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}

	again := s.createTranscript(t, "Lectures", lecture)
	if want := (TranscriptCreateOutput{TaskID: first.TaskID, Path: lecture, State: "running"}); again != want {
		t.Errorf("second call in Lectures returned %+v, want %+v", again, want)
	}

	second := s.createTranscript(t, "Chemistry", lecture)
	if want := (TranscriptCreateOutput{TaskID: "transcription:Chemistry:" + lecture, Path: lecture, State: "started"}); second != want {
		t.Fatalf("call in Chemistry returned %+v, want %+v", second, want)
	}

	deadline := time.After(workDeadline)
	for {
		byID := map[string]host.Task{}
		for _, task := range s.queue.ListTasks() {
			byID[task.ID] = task
		}
		if lTask, okL := byID[first.TaskID]; okL {
			if cTask, okC := byID[second.TaskID]; okC && cTask.Title == taskWaiting {
				if lTask.CompletedCount != 1 || lTask.TotalCount != 2 || cTask.CompletedCount != 0 || cTask.TotalCount != 0 {
					t.Errorf("tasks %+v and %+v, want 1/2 in Lectures and 0/0 in Chemistry", lTask, cTask)
				}
				break
			}
		}
		select {
		case <-deadline:
			t.Fatalf("tasks %+v, want both %s and %s", s.queue.ListTasks(), first.TaskID, second.TaskID)
		case <-time.After(10 * time.Millisecond):
		}
	}

	close(releaseBatch)
	s.waitFinished(t, first.TaskID)
	s.waitFinished(t, second.TaskID)
	s.transcriber.Close()

	inLectures, err := s.texts.ListVersions(t.Context(), lecture)
	if err != nil || inLectures.Live == "" {
		t.Errorf("Lectures lists %+v (%v), want the transcript live", inLectures, err)
	}
	inChemistry, err := s.chemTexts.ListVersions(t.Context(), lecture)
	if err != nil || inChemistry.Live == "" {
		t.Errorf("Chemistry lists %+v (%v), want the transcript live", inChemistry, err)
	}
}

func TestTranscriptCreateReturnsBeforeTheRecordingIsRead(t *testing.T) {
	reader := blockingFiles{startCh: make(chan struct{}), startOnce: &sync.Once{}, release: make(chan struct{})}
	s := newToolFixtureWithFiles(t, func(files Files) Files {
		reader.files = files
		return reader
	})
	close(s.blocking.release)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	select {
	case <-reader.startCh:
	case <-time.After(workDeadline):
		t.Fatalf("the recording was not read within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}
	close(reader.release)
	s.waitFinished(t, want.TaskID)
	s.transcriber.Close()

	if _, text := s.readLive(t); text != "first words\nsecond words\n" {
		t.Errorf("the transcript reads %q", text)
	}
}

func TestTranscriptCreateReportsAFailedReadInTheTask(t *testing.T) {
	s := newToolFixtureWithFiles(t, func(Files) Files { return failingFiles{} })
	close(s.blocking.release)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	deadline := time.After(workDeadline)
	for {
		select {
		case task := <-s.finishing.failCh:
			if task.ID != want.TaskID {
				continue
			}
			tasks := s.queue.ListTasks()
			i := slices.IndexFunc(tasks, func(one host.Task) bool { return one.ID == want.TaskID })
			if i < 0 || tasks[i].FailureReason == "" {
				t.Errorf("the tasks %+v do not list %s with its failure", tasks, want.TaskID)
			}
			return
		case <-deadline:
			t.Fatalf("no failure of %s was reported within %v; tasks %+v", want.TaskID, workDeadline, s.queue.ListTasks())
		}
	}
}

func TestTranscriptCreateRejectsNonRecording(t *testing.T) {
	s := newToolFixture(t)
	s.vault.File("notes/plan.md", "# Plan\nRecord the lecture on tides.\n")

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"notes/plan.md"}`),
	})

	if !res.GetResult().GetIsError() {
		t.Errorf("a note returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if text := res.GetResult().GetText(); !strings.Contains(text, "not an audio or video recording") {
		t.Errorf("the error reads %q, want it to state that the note is not an audio or video recording", text)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a rejected call shows the tasks %+v", tasks)
	}
}

func TestTheTranscriptionToolHasItsInstructions(t *testing.T) {
	s := newToolFixture(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	sentences := getToolSentences(listing, "transcript_create")
	if len(sentences) == 0 {
		t.Fatal("transcript_create lists no instructions")
	}
	if !strings.Contains(strings.Join(sentences, " "), "transcript_create") {
		t.Errorf("no instruction of transcript_create names it: %q", sentences)
	}
	for _, sentence := range sentences {
		if !isOneSentence(sentence) {
			t.Errorf("an instruction of transcript_create is not one sentence: %q", sentence)
		}
	}
}

// readCounts counts the reads the plugin makes of the recording, through its files or the vault, and
// of version files, through its own folder or the host's texts.
type readCounts struct {
	mu        sync.Mutex
	recording int
	versions  int
}

func (c *readCounts) addRecording() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recording++
}

func (c *readCounts) addVersion() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.versions++
}

func (c *readCounts) getCounts() (recording, versions int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.recording, c.versions
}

// countingFiles reads recordings through files and counts each read.
type countingFiles struct {
	files  Files
	counts *readCounts
}

func (f countingFiles) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	f.counts.addRecording()
	return f.files.ReadFile(ctx, vault, path)
}

// countingVault is the vault whose file reads are counted as reads of the recording.
type countingVault struct {
	*hosttest.Vault
	counts *readCounts
}

func (v countingVault) With(id domain.VaultID) countingVault {
	return countingVault{Vault: v.Vault.With(id), counts: v.counts}
}

func (v countingVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	v.counts.addRecording()
	return v.Vault.Read(ctx, path)
}

// countingStore is the plugin's folder, which has its version files, with each read counted.
type countingStore struct {
	Folder
	counts *readCounts
}

func (s countingStore) Read(ctx context.Context, name string) ([]byte, error) {
	s.counts.addVersion()
	return s.Folder.Read(ctx, name)
}

func (s *toolFixture) countReads(t *testing.T) *readCounts {
	t.Helper()
	counts := &readCounts{}
	s.transcriber = NewTranscriber(countingFiles{files: vaultFiles{vaults: func(id domain.VaultID) VaultReader { return s.vault.With(id) }}, counts: counts},
		func(id domain.VaultID) Folder { return countingStore{Folder: s.store.With(id), counts: counts} }, textsOf(s.texts), s.finishing,
		func(context.Context) (engine.TranscriberPort, error) { return s.blocking, nil })
	t.Cleanup(s.transcriber.Close)
	counted := countingVault{Vault: s.vault, counts: counts}
	registry, err := NewTools(s.transcriber, func(id domain.VaultID) VaultReader { return counted.With(id) }).Registry()
	if err != nil {
		t.Fatal(err)
	}
	s.registry = registry
	return counts
}

func TestTranscriptCreateOfATranscribedRecordingReadsNoFile(t *testing.T) {
	s := newToolFixture(t)
	version := s.putTranscript(t, true)
	counts := s.countReads(t)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{
		TaskID: "transcription:Lectures:" + lecture, Path: lecture, State: "done", Version: string(version.GetID()),
	}
	if out != want {
		t.Errorf("transcript_create returned %+v, want %+v", out, want)
	}
	s.checkNoRun(t)
	if recording, versions := counts.getCounts(); recording != 0 || versions != 0 {
		t.Errorf("transcript_create read the recording %d times and version files %d times, want no read",
			recording, versions)
	}
}
