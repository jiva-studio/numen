package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/task"
)

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
// A request that offers an option to reject is answered with it, and one that offers none with an error.
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
	if len(replies) < 3 {
		t.Fatalf("the agent had %d replies, want the answers to the permissions and to the write", len(replies))
	}
	outcome, err := json.Marshal(replies[0]["result"])
	if err != nil || string(outcome) != `{"outcome":{"optionId":"no","outcome":"selected"}}` {
		t.Errorf("the permission to write was answered %s, %v, want the option that rejects", outcome, err)
	}
	if replies[1]["result"] != nil || replies[1]["error"] == nil {
		t.Errorf("the write was answered %v, want an error", replies[1])
	}
	if replies[2]["result"] != nil || replies[2]["error"] == nil {
		t.Errorf("a permission without an option to reject was answered %v, want an error", replies[2])
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

func TestTheProviderServesCompleteTheListsOfModelsAndReasoningLevelsAndTasks(t *testing.T) {
	provider := newTestServer(t).Provider()
	served := slices.Sorted(maps.Keys(provider))
	want := []string{acp.OperationTask, llm.OperationComplete, llm.OperationModels, llm.OperationReasoning}
	if !slices.Equal(served, want) {
		t.Errorf("operations = %v, want %v", served, want)
	}
}

func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	program, _ := fakeProgram(t, "config")
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

	body, err := json.Marshal(map[string]any{
		"operation": llm.OperationModels,
		"call":      map[string]any{"type": "acp", "fields": map[string]string{"program": program}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+"/numen.v1.ProviderService/Call", bytes.NewReader(writeEnvelope(0, body)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/connect+json")
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
	// The done frame is the encoded list of options, which holds the titles as they are.
	if done := readDone(t, answer); response.StatusCode != http.StatusOK || !bytes.Contains(done, []byte("Model Two")) {
		t.Errorf("the service answered %d %q", response.StatusCode, answer)
	}
}

// writeEnvelope frames a message of a streaming Connect call.
func writeEnvelope(flags byte, message []byte) []byte {
	framed := make([]byte, 5, 5+len(message))
	framed[0] = flags
	binary.BigEndian.PutUint32(framed[1:], uint32(len(message)))
	return append(framed, message...)
}

// readDone returns the done frame of the streaming Connect answer, or nil where it holds none.
func readDone(t *testing.T, answer []byte) []byte {
	t.Helper()
	for len(answer) >= 5 {
		size := binary.BigEndian.Uint32(answer[1:5])
		if uint32(len(answer)-5) < size {
			return nil
		}
		message := answer[5 : 5+size]
		answer = answer[5+size:]
		var frame struct {
			Done []byte `json:"done"`
		}
		if json.Unmarshal(message, &frame) == nil && frame.Done != nil {
			return frame.Done
		}
	}
	return nil
}

func newTestServer(t *testing.T) *agentServer {
	t.Helper()
	return newTestServerWith(t, nil, clock.System{})
}

// newTestServerWith builds the plugin over the programs of the machine, with the tools of host and the
// clock.
func newTestServerWith(t *testing.T, host task.Tools, source cache.Clock) *agentServer {
	t.Helper()
	svc, err := newAgentServer(machine{
		lookPath: process.LookPath, launcher: process.Launcher{}, tools: host, clock: source, isAsync: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := svc.tasks.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Log(err)
		}
	})
	return svc
}
