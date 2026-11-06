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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

const (
	// callDeadline is how long a tool call may take before the test fails it as waiting for the work.
	callDeadline = 30 * time.Second
	// workDeadline is how long a transcription may run before the test fails.
	workDeadline = time.Minute
	// chemistryLecture is the recording held only by the vault Chemistry.
	chemistryLecture = "assets/titration.wav"
	chemistrySound   = "RIFF the bytes of a titration lecture"
)

// blockingModel transcribes recordings as the heard model does, but opening one waits until release is
// closed. The first recording it opens closes entered.
type blockingModel struct {
	*heardModel
	entered   chan struct{}
	enterOnce sync.Once
	release   chan struct{}
}

func (m *blockingModel) Open(ctx context.Context, raw []byte) (engine.Recording, error) {
	m.enterOnce.Do(func() { close(m.entered) })
	select {
	case <-m.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return m.heardModel.Open(ctx, raw)
}

// finishingQueue is the queue in memory that sends the id of every task that ends on finished.
type finishingQueue struct {
	*hosttest.Queue
	finished chan string
	failed   chan host.Task
}

func (q finishingQueue) FinishTask(ctx context.Context, id string) error {
	err := q.Queue.FinishTask(ctx, id)
	q.finished <- id
	return err
}

// ReportTask also sends every task reported with a failure on failed.
func (q finishingQueue) ReportTask(ctx context.Context, task host.Task) error {
	err := q.Queue.ReportTask(ctx, task)
	if task.FailureReason != "" {
		q.failed <- task
	}
	return err
}

// blockingFiles reads recordings through files, but each read waits until release is closed. The
// first read closes entered.
type blockingFiles struct {
	files     Files
	entered   chan struct{}
	enterOnce *sync.Once
	release   chan struct{}
}

func (f blockingFiles) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	f.enterOnce.Do(func() { close(f.entered) })
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return f.files.ReadFile(ctx, vault, path)
}

// failingFiles is the files of a vault whose every read fails, while the vault still lists them.
type failingFiles struct{}

var errUnreadable = errors.New("the disk returned an input/output error")

func (failingFiles) ReadFile(context.Context, domain.VaultID, string) ([]byte, error) {
	return nil, errUnreadable
}

// toolScene is the transcription tool over the scene's vault Lectures, shown, and the vault
// Chemistry beside it, holding a recording of its own.
type toolScene struct {
	*scene
	chemTexts *hosttest.Texts
	finishing finishingQueue
	blocking  *blockingModel
	registry  *tools.Registry
}

func newToolScene(t *testing.T) *toolScene {
	t.Helper()
	return newToolSceneOver(t, func(files Files) Files { return files })
}

// newToolSceneOver is the scene whose transcriber reads recordings through what wrap makes of the
// files of the vault.
func newToolSceneOver(t *testing.T, wrap func(Files) Files) *toolScene {
	t.Helper()
	s := &toolScene{scene: newScene(t)}
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	chemistry.File(chemistryLecture, chemistrySound)
	chemTexts, isTexts := s.texts.With("Chemistry").(*hosttest.Texts)
	if !isTexts {
		t.Fatal("the texts of Chemistry are not hosttest texts")
	}
	chemTexts.SetStore(PluginID, chemistry.Store())
	chemTexts.File(chemistryLecture, []byte(chemistrySound))
	s.chemTexts = chemTexts

	s.finishing = finishingQueue{Queue: s.queue, finished: make(chan string, 64), failed: make(chan host.Task, 64)}
	s.blocking = &blockingModel{heardModel: s.model, entered: make(chan struct{}), release: make(chan struct{})}
	s.transcriber = NewTranscriber(wrap(vaultFiles{vault: s.vault}), s.store, s.texts, s.finishing,
		func(context.Context) (engine.TranscriberPort, error) { return s.blocking, nil })
	t.Cleanup(s.transcriber.Close)
	registry, err := NewTools(s.transcriber, s.vault).Registry()
	if err != nil {
		t.Fatal(err)
	}
	s.registry = registry
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
		t.Fatalf("%s did not answer within %v: the call waits for the transcription", call.Name, callDeadline)
		return nil
	}
}

func (s *toolScene) createTranscript(t *testing.T, vault, path string) TranscriptCreateOutput {
	t.Helper()
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: vault, Name: "transcript_create", Arguments: []byte(`{"path":"` + path + `"}`),
	})
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("transcript_create of %s in %s was refused: %s", path, vault, res.GetResult().GetText())
	}
	var out TranscriptCreateOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("decode the result %q: %v", res.GetResult().GetStructured(), err)
	}
	return out
}

func (s *toolScene) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-s.blocking.entered:
	case <-time.After(workDeadline):
		t.Fatalf("no recording was opened within %v; tasks %+v", workDeadline, s.queue.ListTasks())
	}
}

// waitFinished waits for the task of the id to end.
func (s *toolScene) waitFinished(t *testing.T, id string) {
	t.Helper()
	deadline := time.After(workDeadline)
	for {
		select {
		case got := <-s.finishing.finished:
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
	s := newToolScene(t)

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

func TestTheInstructionsAreOneParagraphNamingTranscriptCreate(t *testing.T) {
	if !strings.Contains(instructions, "transcript_create") {
		t.Errorf("the instructions do not name transcript_create: %q", instructions)
	}
	if strings.Contains(strings.TrimSpace(instructions), "\n\n") {
		t.Errorf("the instructions are more than one paragraph: %q", instructions)
	}
}

func TestTranscriptCreateReturnsBeforeTheTranscriptionEnds(t *testing.T) {
	s := newToolScene(t)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	s.waitEntered(t)
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

func TestTranscriptCreateOfARecordingBeingTranscribedReportsItRunning(t *testing.T) {
	s := newToolScene(t)
	first := s.createTranscript(t, "Lectures", lecture)
	s.waitEntered(t)

	second := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: first.TaskID, Path: lecture, State: "running"}
	if second != want {
		t.Errorf("the second call returned %+v, want %+v", second, want)
	}
	close(s.blocking.release)
	s.waitFinished(t, first.TaskID)
}

func TestTranscriptCreateRefusesARecordingNotInTheVault(t *testing.T) {
	s := newToolScene(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"assets/missing.wav"}`),
	})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a recording that is not there returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestTranscriptCreateRefusesACallThatNamesNoVault(t *testing.T) {
	s := newToolScene(t)

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "", Name: "transcript_create", Arguments: []byte(`{"path":"` + lecture + `"}`),
	})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a call with no vault returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestTranscriptCreateTranscribesInTheVaultTheCallNames(t *testing.T) {
	s := newToolScene(t)

	inShown := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"` + chemistryLecture + `"}`),
	})
	if !inShown.GetResult().GetIsRefusal() {
		t.Errorf("a recording only Chemistry holds was accepted in Lectures: %s", inShown.GetResult().GetStructured())
	}

	out := s.createTranscript(t, "Chemistry", chemistryLecture)
	if out.State != "started" || out.TaskID != "transcription:"+chemistryLecture {
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
		t.Errorf("the shown vault holds %+v, want no version", inLectures.Versions)
	}
}

func TestTranscriptCreateAnswersBeforeTheRecordingIsRead(t *testing.T) {
	reader := blockingFiles{entered: make(chan struct{}), enterOnce: &sync.Once{}, release: make(chan struct{})}
	s := newToolSceneOver(t, func(files Files) Files {
		reader.files = files
		return reader
	})
	close(s.blocking.release)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	select {
	case <-reader.entered:
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
	s := newToolSceneOver(t, func(Files) Files { return failingFiles{} })
	close(s.blocking.release)

	out := s.createTranscript(t, "Lectures", lecture)

	want := TranscriptCreateOutput{TaskID: "transcription:" + lecture, Path: lecture, State: "started"}
	if out != want {
		t.Fatalf("transcript_create returned %+v, want %+v", out, want)
	}
	deadline := time.After(workDeadline)
	for {
		select {
		case task := <-s.finishing.failed:
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

func TestTranscriptCreateRefusesAFileThatIsNoRecording(t *testing.T) {
	s := newToolScene(t)
	s.vault.File("notes/plan.md", "# Plan\nRecord the lecture on tides.\n")

	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"notes/plan.md"}`),
	})

	if !res.GetResult().GetIsRefusal() {
		t.Errorf("a note returned %q %s, want a refusal", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a refused call shows the tasks %+v", tasks)
	}
}

func TestRegistryListsTheTranscriptionInstructions(t *testing.T) {
	s := newToolScene(t)

	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions == "" || listing.Instructions != instructions {
		t.Errorf("the registry lists the instructions %q, want %q", listing.Instructions, instructions)
	}
	if !strings.Contains(listing.Instructions, "transcript_create") {
		t.Errorf("the listed instructions do not name transcript_create: %q", listing.Instructions)
	}
	if strings.Contains(strings.TrimSpace(listing.Instructions), "\n\n") {
		t.Errorf("the listed instructions are more than one paragraph: %q", listing.Instructions)
	}
}
