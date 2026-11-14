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
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

const writeTargetVariable = "NUMEN_TEST_WRITE_TARGET"

// textEvent is the line Claude Code writes for a piece of its answer.
const textEvent = `printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"fixed"}}}'` + "\n" +
	`printf '%s\n' '{"type":"result","is_error":false}'` + "\n"

// answerScript is a Claude Code that says "fixed" and writes the file named by the variable unless it
// was started with its tools turned off.
const answerScript = `cat > /dev/null
tools=on
prev=
for a in "$@"; do
  if [ "$prev" = "--tools" ]; then tools=$a; fi
  prev=$a
done
if [ "$tools" != "" ]; then echo written > "$` + writeTargetVariable + `"; fi
` + textEvent

func userMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

func completeCall(options map[string]string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: "claude-code", Options: options}
}

func TestCompleteReturnsTheAnswerOfClaude(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", answerScript))
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("fix this"))

	if err != nil || result.GetText() != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.GetText(), err, "fixed")
	}
}

func TestCompleteStartsClaudeWithTheModelTheInstructionsAndNoTool(t *testing.T) {
	dir := t.TempDir()
	svc := newClaude(t, nil, writeScript(t, "claude", recordArgs(dir)+textEvent))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(map[string]string{llm.OptionModel: "opus"}), &llm.CompleteRequest{
		Messages: []*llm.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	argv := readArgv(t, filepath.Join(dir, "argv"))
	assertPair(t, argv, "--tools", "")
	assertPair(t, argv, "--append-system-prompt", "be brief")
	assertPair(t, argv, "--model", "opus")
	assertPair(t, argv, "--permission-mode", "dontAsk")
	for _, forbidden := range []string{"--mcp-config", "--allowedTools"} {
		if slices.Contains(argv, forbidden) {
			t.Errorf("%q is among %v", forbidden, argv)
		}
	}
	stdin, err := os.ReadFile(filepath.Join(dir, "stdin"))
	if err != nil || string(stdin) != "fix this" {
		t.Errorf("the program was asked %q, %v, want %q", stdin, err, "fix this")
	}
}

func TestCompleteJoinsTheTurnsOfAConversationIntoOnePrompt(t *testing.T) {
	dir := t.TempDir()
	svc := newClaude(t, nil, writeScript(t, "claude", recordArgs(dir)+textEvent))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), &llm.CompleteRequest{
		Messages: []*llm.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	stdin, err := os.ReadFile(filepath.Join(dir, "stdin"))
	if err != nil || string(stdin) != "User:\none\n\nAssistant:\ntwo\n\nUser:\nthree" {
		t.Errorf("the program was asked %q, %v", stdin, err)
	}
}

// The program is started with its tools turned off, so a write it attempts is refused. The script writes
// whenever its tools are not off, which the control run shows.
func TestACompleteCallLetsClaudeWriteNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "written")
	t.Setenv(writeTargetVariable, target)
	script := writeScript(t, "claude", answerScript)
	if err := exec.CommandContext(t.Context(), script).Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the control run wrote nothing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	svc := newClaude(t, nil, script)

	if _, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Claude Code wrote the file during complete: %v", err)
	}
}

func TestCompleteRefusesARequestWithNoMessageToAnswer(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "exit 0\n"))

	for name, request := range map[string]*llm.CompleteRequest{
		"no message":   {},
		"system only":  {Messages: []*llm.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []*llm.Message{{Role: "tool", Text: "x"}}},
	} {
		_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), request)
		if !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAMissingClaudeIsUnreachable(t *testing.T) {
	withoutClaudeAnywhere(t)
	svc := newTestAgent(t, nil)

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestAFailedRunOfClaudeIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "cat > /dev/null\necho 'not signed in as sk-secret' >&2\nexit 2\n"))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "cat > /dev/null\nprintf '%s\\n' '{\"type\":\"result\",\"is_error\":false}'\n"))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "sleep 30\n"))
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "sleep 30\n"))
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := llmtest.Complete(ctx, svc.Provider(), completeCall(nil), userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

// listingAnswer is the answer of a Claude Code that lists two models.
const listingAnswer = `{"type":"control_response","response":{"subtype":"success","request_id":"numen-models","response":{"models":[{"value":"opus","displayName":"Opus","supportsEffort":true,"supportedEffortLevels":["high","low"]},{"value":"haiku","displayName":"Haiku","supportsEffort":false,"supportedEffortLevels":[]}]}}}`

func newListingClaude(t *testing.T, request string) *agent {
	t.Helper()
	return newClaude(t, nil, writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\nprintf '%s\\n' '"+listingAnswer+"'\nsleep 30\n"))
}

// The program is asked for its models with the initialize request of its control protocol.
func TestTheModelsOfClaudeAreListedForTheRow(t *testing.T) {
	request := filepath.Join(t.TempDir(), "asked")
	svc := newListingClaude(t, request)

	listed, err := svc.completion.ListModels(t.Context(), completeCall(nil), connection.OptionsQuery{Operations: []string{llm.OperationComplete}})
	models := listed.Options

	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range models {
		ids = append(ids, one.Value)
	}
	if !slices.Equal(ids, []string{"opus", "haiku"}) || models[0].Title != "Opus" {
		t.Errorf("models = %+v", models)
	}
	asked, err := os.ReadFile(request)
	if err != nil || !strings.Contains(string(asked), `"subtype":"initialize"`) {
		t.Errorf("the program was asked %s, %v", asked, err)
	}
}

func TestTheLevelsOfTheModelOfTheCallAreListedFromTheLeastToTheMost(t *testing.T) {
	svc := newListingClaude(t, filepath.Join(t.TempDir(), "asked"))

	levels, err := svc.completion.ListReasoningLevels(t.Context(), completeCall(map[string]string{llm.OptionModel: "opus"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), completeCall(map[string]string{llm.OptionModel: "haiku"}))
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

// A level the listed model does not offer is not passed to the program.
func TestATaskPassesNoReasoningLevelTheModelDoesNotOffer(t *testing.T) {
	svc := newListingClaude(t, filepath.Join(t.TempDir(), "asked"))
	if _, err := svc.completion.ListModels(t.Context(), completeCall(nil), connection.OptionsQuery{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	svc.settings.Apply(map[string]string{"executablePath": writeScript(t, "claude", recordArgs(dir))})

	call := completeCall(map[string]string{llm.OptionModel: "haiku", llm.OptionReasoning: "high"})
	if _, _, err := runTask(t.Context(), svc, call, &acp.TaskRequest{Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}
	if argv := readArgv(t, filepath.Join(dir, "argv")); slices.Contains(argv, "--effort") {
		t.Errorf("a level haiku does not take was given: %q", argv)
	}
}

func TestAClaudeThatListsNoModelListsTheFallbackModels(t *testing.T) {
	svc := newClaude(t, nil, writeScript(t, "claude", "exit 1\n"))

	listed, err := svc.completion.ListModels(t.Context(), completeCall(nil), connection.OptionsQuery{})
	models := listed.Options

	if err != nil || len(models) != len(model.GetFallback()) {
		t.Fatalf("ListModels = %+v, %v", models, err)
	}
	for place, one := range models {
		if one.Value != model.GetFallback()[place].Value {
			t.Errorf("the model %v is not a fallback alias", one)
		}
	}
}

func TestListingTheModelsOfAMissingClaudeIsUnreachable(t *testing.T) {
	withoutClaudeAnywhere(t)
	svc := newTestAgent(t, nil)

	_, err := svc.completion.ListModels(t.Context(), completeCall(nil), connection.OptionsQuery{Operations: []string{llm.OperationComplete}})

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("ListModels = %v, want try the next", err)
	}
}

func TestTheProviderServesCompleteTheListsOfModelsAndReasoningLevelsAndTasks(t *testing.T) {
	served := slices.Sorted(maps.Keys(newTestAgent(t, nil).Provider()))
	want := []string{acp.OperationTask, connection.OperationDetected, llm.OperationComplete, llm.OperationModels, llm.OperationReasoning}
	if !slices.Equal(served, want) {
		t.Errorf("operations = %v, want %v", served, want)
	}
}

// The plugin linked into a host reads the models of the program in the background and lists none until
// they are read.
func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	program := writeScript(t, "claude", "read line\nprintf '%s\\n' '"+listingAnswer+"'\nsleep 30\n")
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	lifecycle := servertest.NewLifecycleClient(http.DefaultClient, web.URL, "tok")
	if _, err := lifecycle.Configure(t.Context(), map[string]string{"executablePath": program}); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(5 * time.Second)
	for {
		answer := callModels(t, web.URL)
		if bytes.Contains(readDoneFrame(t, answer), []byte("Opus")) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("the service answered %q", answer)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func callModels(t *testing.T, url string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"operation": llm.OperationModels, "call": map[string]any{"type": "claude-code"}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url+"/numen.v1.ProviderService/Call", bytes.NewReader(writeEnvelope(body)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/connect+json")
	request.Header.Set("Authorization", "Bearer tok")
	response, err := http.DefaultClient.Do(request)
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
	if response.StatusCode != http.StatusOK {
		t.Fatalf("the service answered %d %q", response.StatusCode, answer)
	}
	return answer
}

func writeEnvelope(message []byte) []byte {
	framed := make([]byte, 5, 5+len(message))
	binary.BigEndian.PutUint32(framed[1:], uint32(len(message)))
	return append(framed, message...)
}

func readDoneFrame(t *testing.T, answer []byte) []byte {
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

func TestTheProgramOnThePathIsDetected(t *testing.T) {
	withoutClaudeAnywhere(t)
	t.Setenv("PATH", filepath.Dir(writeScript(t, "claude", "exit 0\n")))
	svc := newTestAgent(t, nil)

	found, err := svc.tasks.DetectConnections(t.Context(), connection.Call{Type: "claude-code"})

	if err != nil || len(found) != 1 || found[0].Name != "Claude Code" {
		t.Errorf("DetectConnections = %+v, %v, want Claude Code", found, err)
	}
}

func TestTheProgramAtTheExecutablePathIsDetected(t *testing.T) {
	withoutClaudeAnywhere(t)
	program := writeScript(t, "claude", "exit 0\n")
	svc := newClaude(t, nil, program)

	found, err := svc.tasks.DetectConnections(t.Context(), connection.Call{Type: "claude-code"})

	if err != nil || len(found) != 1 || found[0].Name != "Claude Code" || found[0].Path != program {
		t.Errorf("DetectConnections = %+v, %v, want Claude Code at %s", found, err, program)
	}
}

func TestNothingIsDetectedWithoutTheProgram(t *testing.T) {
	withoutClaudeAnywhere(t)
	svc := newTestAgent(t, nil)

	found, err := svc.tasks.DetectConnections(t.Context(), connection.Call{Type: "claude-code"})

	if err != nil || len(found) != 0 {
		t.Errorf("DetectConnections = %+v, %v, want none", found, err)
	}
}

func TestCompleteReturnsAnErrorWhenThePluginStopsAfterSomeText(t *testing.T) {
	started := filepath.Join(t.TempDir(), "started")
	// The marker is created a second after the text is written, so the plugin has read the text by then.
	script := `printf '%s\n' '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"half an ans"}}}'` +
		"\nsleep 1\ntouch '" + started + "'\nsleep 30\n"
	svc := newClaude(t, nil, writeScript(t, "claude", script))
	type outcome struct {
		result *llm.CompleteResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))
		done <- outcome{result: result, err: err}
	}()
	deadline := time.After(20 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		select {
		case got := <-done:
			t.Fatalf("Complete ended before the text was written: %q, %v", got.result.GetText(), got.err)
		case <-deadline:
			t.Fatal("the program did not write the text")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		case <-time.After(10 * time.Millisecond):
		}
	}

	if err := svc.tasks.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		if got.err == nil || got.result.GetText() != "" {
			t.Errorf("Complete = %q, %v, want an error and no text", got.result.GetText(), got.err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Complete did not end after Close")
	}
}
