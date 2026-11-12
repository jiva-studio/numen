package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
)

func fakeProgram(t *testing.T, behavior string) (string, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "record")
	program := agenttest.WriteScript(t, "fake-agent",
		"exec "+agenttest.QuoteShell(os.Args[0])+" fake-acp "+behavior+" "+agenttest.QuoteShell(record)+"\n")
	return program, record
}

func programCall(program string) connection.Call {
	return connection.Call{Type: "acp", Fields: map[string]string{"program": program}}
}

func userMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

func TestCompleteReturnsTheAnswerOfTheProgram(t *testing.T) {
	program, record := fakeProgram(t, "plain")

	result, err := newTestServer(t).completion.Complete(t.Context(), programCall(program), &llm.CompleteRequest{
		Messages: []*llm.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "why?"}},
	})

	if err != nil || result.GetText() != "the answer" {
		t.Fatalf("Complete = %q, %v, want %q", result.GetText(), err, "the answer")
	}
	prompts := readRecord(t, record)["session/prompt"]
	if len(prompts) != 1 {
		t.Fatalf("the agent was prompted %d times, want 1", len(prompts))
	}
	var texts []string
	for _, block := range getField[[]any](t, getField[map[string]any](t, prompts[0], "params"), "prompt") {
		texts = append(texts, getField[string](t, block, "text"))
	}
	if strings.Join(texts, "|") != "be brief|why?" {
		t.Errorf("the prompt held %q, want the instructions and then the question", texts)
	}
}

func TestCompleteAsksTheProgramForTheModelAndTheLevelOfTheCall(t *testing.T) {
	program, record := fakeProgram(t, "thought")
	call := programCall(program)
	call.Options = map[string]string{llm.OptionModel: "m2", llm.OptionReasoning: "high"}

	if _, err := newTestServer(t).completion.Complete(t.Context(), call, userMessage("why?")); err != nil {
		t.Fatal(err)
	}

	var values []string
	for _, message := range readRecord(t, record)["session/set_config_option"] {
		values = append(values, getField[string](t, getField[map[string]any](t, message, "params"), "value"))
	}
	if !slices.Equal(values, []string{"m2", "high"}) {
		t.Errorf("the session was set to %v, want the model and then the level", values)
	}
}

// The session offers the program no file system and no terminal, no server of tools, and the client
// answers every permission request with a refusal, so a write the program attempts is refused.
func TestACompleteCallRefusesEveryWriteOfTheProgram(t *testing.T) {
	program, record := fakeProgram(t, "write")

	if _, err := newTestServer(t).completion.Complete(t.Context(), programCall(program), userMessage("write a note")); err != nil {
		t.Fatal(err)
	}

	observed := readRecord(t, record)
	capabilities := getField[map[string]any](t, getField[map[string]any](t, observed["initialize"][0], "params"), "clientCapabilities")
	raw, err := json.Marshal(capabilities)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false}`; string(raw) != want {
		t.Errorf("the client offered %s, want %s", raw, want)
	}
	servers := getField[[]any](t, getField[map[string]any](t, observed["session/new"][0], "params"), "mcpServers")
	if len(servers) != 0 {
		t.Errorf("the session was given the servers %v", servers)
	}
	replies := observed["reply"]
	if len(replies) < 2 {
		t.Fatalf("the agent had %d replies, want the answers to the permission and to the write", len(replies))
	}
	outcome, err := json.Marshal(replies[0]["result"])
	if err != nil || string(outcome) != `{"outcome":{"outcome":"canceled"}}` {
		t.Errorf("the permission to write was answered %s, %v, want a refusal", outcome, err)
	}
	if replies[1]["result"] != nil || replies[1]["error"] == nil {
		t.Errorf("the write was answered %v, want an error", replies[1])
	}
}

func TestCompleteRunsTheProgramInAnEmptyTemporaryFolder(t *testing.T) {
	program, record := fakeProgram(t, "plain")

	if _, err := newTestServer(t).completion.Complete(t.Context(), programCall(program), userMessage("why?")); err != nil {
		t.Fatal(err)
	}

	folders := readRecord(t, record)["folder"]
	if len(folders) != 1 {
		t.Fatalf("the program recorded %d folders", len(folders))
	}
	path := getField[string](t, folders[0], "path")
	if entries := getField[float64](t, folders[0], "entries"); entries != 0 {
		t.Errorf("the folder %q held %v entries, want it empty", path, entries)
	}
	if !strings.HasPrefix(filepath.Base(path), "numen-acp-") {
		t.Errorf("the program ran in %q, want a temporary folder of the plugin", path)
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("the folder %q is still there", path)
	}
}

func TestCompleteRefusesARequestWithNoProgramOrNoMessageToAnswer(t *testing.T) {
	program, _ := fakeProgram(t, "plain")
	svc := newTestServer(t)

	for name, ask := range map[string]func() error{
		"no program": func() error {
			_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "acp"}, userMessage("hi"))
			return err
		},
		"no message": func() error {
			_, err := svc.completion.Complete(t.Context(), programCall(program), &llm.CompleteRequest{})
			return err
		},
		"unknown role": func() error {
			_, err := svc.completion.Complete(t.Context(), programCall(program), &llm.CompleteRequest{
				Messages: []*llm.Message{{Role: "tool", Text: "x"}},
			})
			return err
		},
	} {
		if err := ask(); !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAProgramThatIsNotThereIsUnreachable(t *testing.T) {
	_, err := newTestServer(t).completion.Complete(t.Context(), programCall(filepath.Join(t.TempDir(), "gone")), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnavailable {
		t.Errorf("reason = %v, want unreachable", reason)
	}
}

func TestAFailedRunOfTheProgramIsAStopWithNoTextOfTheProgram(t *testing.T) {
	program, _ := fakeProgram(t, "fail")

	_, err := newTestServer(t).completion.Complete(t.Context(), programCall(program), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestATurnTheProgramRejectedIsAStop(t *testing.T) {
	program, _ := fakeProgram(t, "refuse")

	_, err := newTestServer(t).completion.Complete(t.Context(), programCall(program), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	program, _ := fakeProgram(t, "hang")
	svc := newTestServer(t)
	svc.settings.Apply(map[string]string{"requestTimeout": "0.3"})

	_, err := svc.completion.Complete(t.Context(), programCall(program), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnavailable {
		t.Errorf("reason = %v, want timeout", reason)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	program, _ := fakeProgram(t, "hang")
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(300*time.Millisecond, cancel)

	_, err := newTestServer(t).completion.Complete(ctx, programCall(program), userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

func TestTheModelsOfTheProgramAreListedForTheRow(t *testing.T) {
	program, _ := fakeProgram(t, "config")

	models, err := newTestServer(t).completion.ListModels(t.Context(), programCall(program), []string{llm.OperationComplete})

	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, one := range models {
		titles = append(titles, one.Value+"="+one.Title)
	}
	if !slices.Equal(titles, []string{"m1=Model One", "m2=Model Two"}) {
		t.Errorf("models = %v", titles)
	}
}

func TestTheLevelsOfTheCurrentModelOfTheProgramAreListed(t *testing.T) {
	program, _ := fakeProgram(t, "thought")
	svc := newTestServer(t)

	levels, err := svc.completion.ListReasoningLevels(t.Context(), programCall(program))
	if err != nil {
		t.Fatal(err)
	}
	var values []string
	for _, one := range levels {
		values = append(values, one.Value+"="+one.Title)
	}
	if !slices.Equal(values, []string{"low=Low", "medium=Medium", "high=High"}) {
		t.Errorf("levels = %v", values)
	}
}

func TestListingTheModelsOfAProgramThatIsNotThereIsUnreachable(t *testing.T) {
	_, err := newTestServer(t).completion.ListModels(t.Context(), programCall(filepath.Join(t.TempDir(), "gone")), []string{llm.OperationComplete})

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("ListModels = %v, want try the next", err)
	}
}

func TestTheProviderFillsTheRolesOfCompleteAndModels(t *testing.T) {
	provider := newTestServer(t).Provider()
	if provider.Completer == nil || provider.Models == nil || provider.Embedder != nil || provider.Profiles != nil {
		t.Errorf("provider = %+v, want complete and the models alone", provider)
	}
}

func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	program, _ := fakeProgram(t, "plain")
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	lifecycle := server.NewLifecycleClient(http.DefaultClient, web.URL, hostcall.WithBearer("tok"))
	if _, err := lifecycle.Configure(t.Context(), map[string]string{}); err != nil {
		t.Fatal(err)
	}

	body := `{"call":{"type":"acp","fields":{"program":` + quoteJSON(program) + `}},"messages":[{"role":"user","text":"why?"}]}`
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+"/numen.v1.ProviderService/Complete", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer tok")
	response, err := web.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()

	answer, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(answer) != `{"text":"the answer"}` {
		t.Errorf("the service answered %d %q", response.StatusCode, answer)
	}
}

func quoteJSON(s string) string {
	raw, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(raw)
}

func newTestServer(t *testing.T) *agentServer {
	t.Helper()
	svc, err := newAgentServer(process.LookPath, process.Launcher{}, clock.System{}, false)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}
