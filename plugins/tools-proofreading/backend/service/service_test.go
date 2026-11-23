package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/model"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/usecase/proofread"
)

const (
	ocrID    = "md.test.alpha"
	scanPath = "books/flywheel.pdf"
)

// scene is a vault with a scan the OCR plugin made a text of, and the proofreading over it as the
// plugin mounts it.
type scene struct {
	texts    *hosttest.Texts
	queue    *hosttest.Queue
	host     proofread.Host
	settings *settings.Store
	requests []*llm.CompleteRequest
	script   func(prompt string) string
	run      *proofread.Proofreader
}

func newScene(t *testing.T) *scene {
	t.Helper()
	vault := hosttest.New("Physics")
	s := &scene{
		texts:    hosttest.NewTexts("Physics", PluginID),
		queue:    hosttest.NewQueue(),
		settings: settings.NewStore(),
		script:   func(string) string { return "" },
	}
	s.texts.WriteFile(scanPath, []byte("the scan"))
	store := vault.Store()
	s.texts.SetStore(PluginID, store)
	complete := func(_ context.Context, request *llm.CompleteRequest, _ connection.Priority) (*llm.CompleteResult, error) {
		s.requests = append(s.requests, request)
		return &llm.CompleteResult{Text: s.script(request.GetMessages()[1].GetText())}, nil
	}
	s.host = proofread.Host{
		Store:    func(id domain.VaultID) host.StoreWriter { return store.With(id) },
		GetTexts: func(id domain.VaultID) proofread.Texts { return s.texts.With(id) },
		Queue:    s.queue,
	}
	s.run = newProofreading(s.host, complete, s.settings)
	return s
}

func (s *scene) holdScan(t *testing.T) domain.TextVersion {
	t.Helper()
	parts := []domain.TextSegment{
		{Text: "A flywheel keeps energy ", Selector: `{"page":1}`},
		{Text: "in a rotor spinnig in a vacum.\n\n", Selector: `{"page":1}`},
		{Text: "Pumped storage lifts water.\n\n", Selector: `{"page":2}`},
	}
	if err := host.WriteText(t.Context(), s.texts.GetStore(ocrID), "recognition.jsonl", parts); err != nil {
		t.Fatal(err)
	}
	version, err := s.texts.UsePlugin(ocrID).PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: "recognition", Kind: "recognition", InputHash: domain.HashContent([]byte("the scan")), File: "recognition.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) readLiveSegments(t *testing.T) ([]domain.TextSegment, domain.TextVersion) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			parts, err := proofread.ReadParts(t.Context(), s.texts, version)
			if err != nil {
				t.Fatal(err)
			}
			return parts, version
		}
	}
	t.Fatalf("%s has no live version: %+v", scanPath, listing)
	return nil, domain.TextVersion{}
}

func getRows(text string, replace func(text string) string) string {
	var rows []string
	for _, segment := range strings.Split(text, numbered.Opens)[1:] {
		number, line, _ := strings.Cut(segment, numbered.Closes)
		line = strings.TrimSpace(line)
		if fixed := replace(line); fixed != line {
			rows = append(rows, number+"|"+fixed)
		}
	}
	return strings.Join(rows, "\n")
}

func TestProofreadWithoutAVersionUsesTheLiveOne(t *testing.T) {
	s := newScene(t)
	s.holdScan(t)
	s.script = func(text string) string {
		return getRows(text, func(line string) string { return strings.ReplaceAll(line, "vacum", "vacuum") })
	}

	res, err := newProofreadCommand(s.run)(t.Context(), command.Request{Command: "proofreading.create", Path: scanPath})
	if err != nil {
		t.Fatal(err)
	}
	if want := "proofread 3 lines of books/flywheel.pdf and corrected 1"; res.Message != want {
		t.Errorf("message = %q, want %q", res.Message, want)
	}

	if _, live := s.readLiveSegments(t); live.PluginID != PluginID {
		t.Errorf("the live version is %+v", live)
	}

	s.script = func(string) string { return "999|invented" }
	uncorrectedRes, err := newProofreadCommand(s.run)(t.Context(), command.Request{Command: "proofreading.create", Path: scanPath})
	if err != nil {
		t.Fatal(err)
	}
	if want := "proofread 3 lines of books/flywheel.pdf and corrected 0; the model returned no reply for 2 batches"; uncorrectedRes.Message != want {
		t.Errorf("uncorrected message = %q, want %q", uncorrectedRes.Message, want)
	}

	if failedRes, err := newProofreadCommand(s.run)(t.Context(), command.Request{Command: "proofreading.create", Path: "books/missing.pdf"}); err == nil || failedRes != (command.Result{}) {
		t.Errorf("missing file gave %+v, %v", failedRes, err)
	}
}

func TestThePluginRunsNeitherDeleteTextNorTheCommandOfBefore(t *testing.T) {
	s := newScene(t)
	commands, err := newCommands(s.run)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "proofreading.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}

func TestTheMockMarksTheLinesTheGatesLetThrough(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run = newProofreading(s.host, model.WithMock(failWith(connection.ErrNoConnection)), s.settings)

	if _, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLiveSegments(t)
	if parts[0].Text != "A flywheel keeps energy " {
		t.Errorf("the gate let a mark on a short line through: %q", parts[0].Text)
	}
	for _, part := range parts[1:] {
		if !strings.HasSuffix(strings.TrimSpace(part.Text), " "+model.MockMark) {
			t.Errorf("the mock left %q", part.Text)
		}
	}
}

func failWith(err error) model.Completer {
	return func(context.Context, *llm.CompleteRequest, connection.Priority) (*llm.CompleteResult, error) {
		return nil, err
	}
}

func TestProofreadWithNoConnectionIsOffAndPointsToTheConnectionsPage(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run = newProofreading(s.host, failWith(connection.ErrNoConnection), s.settings)

	_, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()})

	if !errors.Is(err, model.ErrNoConnection) || !strings.Contains(err.Error(), "Connections") {
		t.Fatalf("got %v, want %v", err, model.ErrNoConnection)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "Connections") {
		t.Errorf("tasks %+v", tasks)
	}
	if _, live := s.readLiveSegments(t); live.PluginID != ocrID {
		t.Errorf("a proofreading that is off left the live version %+v", live)
	}
}

func TestProofreadNamesEveryConnectionThatFailedInTheTask(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	failed := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "rate limited"},
		{ConnectionName: "Local", Reason: "unreachable"},
	}}
	s.run = newProofreading(s.host, failWith(failed), s.settings)

	_, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()})

	all, isAllFailed := errors.AsType[*connection.AllFailedError](err)
	if !isAllFailed || len(all.Failures) != 2 {
		t.Fatalf("got %v, want every connection failed", err)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "OpenRouter: rate limited; Local: unreachable") {
		t.Errorf("tasks %+v", tasks)
	}
}

func TestAMistakeInTheRequestEndsTheProofreadingWithTheConnectionThatFoundIt(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run = newProofreading(s.host, failWith(&connection.StoppedError{ConnectionName: "OpenRouter", Reason: "bad request"}), s.settings)

	_, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()})

	var rejected *connection.StoppedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "OpenRouter" {
		t.Fatalf("got %v, want a rejection by OpenRouter", err)
	}
}

func TestMountAndHostValidation(t *testing.T) {
	s := newScene(t)
	for _, incomplete := range []proofread.Host{
		{GetTexts: s.host.GetTexts, Queue: s.host.Queue},
		{Store: s.host.Store, Queue: s.host.Queue},
		{Store: s.host.Store, GetTexts: s.host.GetTexts},
	} {
		if err := mount(server.New(), incomplete, failWith(connection.ErrNoConnection)); !errors.Is(err, errIncompleteHost) {
			t.Errorf("incomplete host got %v, want %v", err, errIncompleteHost)
		}
	}

	if err := mount(server.New(), s.host, failWith(connection.ErrNoConnection)); err != nil {
		t.Errorf("valid mount failed: %v", err)
	}

	opened := openHostAt(hostcall.Endpoint{BaseURL: "http://127.0.0.1:1", Token: "tok"})
	if opened.Store("v") == nil || opened.GetTexts("v") == nil || opened.Queue == nil {
		t.Errorf("openHostAt returned nil field: %+v", opened)
	}

	t.Setenv("NUMEN_HOST_PORT", "")
	t.Setenv("NUMEN_PLUGIN_TOKEN", "")
	if err := Mount(server.New()); err == nil {
		t.Error("Mount without env succeeded")
	}

	t.Setenv("NUMEN_HOST_PORT", "12345")
	t.Setenv("NUMEN_PLUGIN_TOKEN", "token")
	t.Setenv(model.EnvMockML, "false")
	if err := Mount(server.New()); err != nil {
		t.Errorf("Mount with env failed: %v", err)
	}
	t.Setenv(model.EnvMockML, "true")
	if err := Mount(server.New()); err != nil {
		t.Errorf("Mount with mock env failed: %v", err)
	}
}
