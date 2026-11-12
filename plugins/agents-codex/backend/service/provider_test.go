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
)

const writeTargetVariable = "NUMEN_TEST_WRITE_TARGET"

// answerScript is a Codex that says "fixed" and writes the file named by the variable unless it was
// started with the read-only sandbox.
const answerScript = `sandbox=
prev=
for a in "$@"; do
  if [ "$prev" = "--sandbox" ]; then sandbox=$a; fi
  if [ "$a" = "--dangerously-bypass-approvals-and-sandbox" ]; then sandbox=none; fi
  prev=$a
done
if [ "$sandbox" != "read-only" ]; then echo written > "$` + writeTargetVariable + `"; fi
echo '{"type":"item.completed","item":{"type":"agent_message","text":"fixed"}}'
`

// answerLine is the event of a Codex that answers "ok".
const answerLine = `echo '{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}'` + "\n"

func userMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

func completeCall(options map[string]string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: "codex", Options: options}
}

func TestCompleteReturnsTheAnswerOfCodex(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", answerScript))
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("fix this"))

	if err != nil || result.GetText() != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.GetText(), err, "fixed")
	}
}

func TestCompleteStartsCodexWithTheModelTheInstructionsAndNoTool(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, writeScript(t, "codex", recordArgs(dir)+answerLine))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(map[string]string{llm.OptionModel: "gpt-x", llm.OptionReasoning: "high"}), &llm.CompleteRequest{
		Messages: []*llm.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	lines := readArgv(t, dir)
	joined := strings.Join(lines, "\n")
	if !strings.HasSuffix(joined, "\n--\nbe brief\n\nfix this") {
		t.Errorf("the arguments are %v, want the instructions and the question last", lines)
	}
	for _, want := range []string{"--model\ngpt-x", `model_reasoning_effort="high"`, "--sandbox\nread-only"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q absent from %v", want, lines)
		}
	}
	for _, forbidden := range []string{"--dangerously-bypass-approvals-and-sandbox", "mcp_servers"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("%q is among %v", forbidden, lines)
		}
	}
}

func TestCompleteJoinsTheTurnsOfAConversationIntoOnePrompt(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, writeScript(t, "codex", recordArgs(dir)+answerLine))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), &llm.CompleteRequest{
		Messages: []*llm.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	lines := readArgv(t, dir)
	if got := strings.Join(lines[len(lines)-8:], "\n"); got != "User:\none\n\nAssistant:\ntwo\n\nUser:\nthree" {
		t.Errorf("the prompt is %q", got)
	}
}

// The program is given a read-only sandbox, so a write it attempts is refused. The script writes
// whenever the sandbox is not read-only, which the control run shows.
func TestACompleteCallLetsCodexWriteNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "written")
	t.Setenv(writeTargetVariable, target)
	script := writeScript(t, "codex", answerScript)
	if err := exec.CommandContext(t.Context(), script).Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the control run wrote nothing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	svc := newCodex(t, nil, script)

	if _, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Codex wrote the file during complete: %v", err)
	}
}

func TestCompleteRunsCodexInAnEmptyFolderThatIsRemoved(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, writeScript(t, "codex", recordArgs(dir)+"ls -A > "+dir+"/listing\n"+answerLine))

	if _, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi")); err != nil {
		t.Fatal(err)
	}

	if listing := readFile(t, filepath.Join(dir, "listing")); listing != "" {
		t.Errorf("the folder held %q", listing)
	}
	if _, err := os.Stat(strings.TrimSpace(readFile(t, filepath.Join(dir, "pwd")))); err == nil {
		t.Error("the folder is still there")
	}
}

func TestCompleteRefusesARequestWithNoUserMessage(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "exit 0\n"))

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

func TestAMissingCodexIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newTestServer(t, nil)

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestAFailedRunOfCodexIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "echo 'not signed in as sk-secret' >&2\nexit 2\n"))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "exit 0\n"))

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "sleep 30\n"))
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := llmtest.Complete(t.Context(), svc.Provider(), completeCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "sleep 30\n"))
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := llmtest.Complete(ctx, svc.Provider(), completeCall(nil), userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

// appServerScript is a Codex whose app-server lists two models.
const appServerScript = `echo '{"id":1,"result":{}}'` + "\n" +
	`echo '{"id":2,"result":{"data":[{"model":"gpt-x","displayName":"GPT X","isDefault":true,` +
	`"supportedReasoningEfforts":[{"reasoningEffort":"low"},{"reasoningEffort":"high"}]},{"model":"gpt-y"},{"model":"hid","hidden":true}]}}'` + "\n" +
	"while read -r _; do :; done\n"

func TestTheModelsOfCodexAreListedForTheRow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	svc := newCodex(t, nil, writeScript(t, "codex", appServerScript))

	models, err := svc.completion.ListModels(t.Context(), completeCall(nil), []string{llm.OperationComplete})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range models {
		ids = append(ids, one.Value)
	}
	if !slices.Equal(ids, []string{"gpt-x", "gpt-y"}) || models[0].Title != "GPT X" {
		t.Errorf("models = %+v", models)
	}
	levels, err := svc.completion.ListReasoningLevels(t.Context(), completeCall(map[string]string{llm.OptionModel: "gpt-x"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), completeCall(map[string]string{llm.OptionModel: "gpt-y"}))
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

func TestListingTheModelsOfAMissingCodexIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newTestServer(t, nil)

	_, err := svc.completion.ListModels(t.Context(), completeCall(nil), nil)

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("ListModels = %v, want try the next", err)
	}
}

func TestTheProviderServesCompleteTheListsOfModelsAndReasoningLevelsAndTasks(t *testing.T) {
	served := slices.Sorted(maps.Keys(newTestServer(t, nil).Provider()))
	want := []string{acp.OperationTask, llm.OperationComplete, llm.OperationModels, llm.OperationReasoning}
	if !slices.Equal(served, want) {
		t.Errorf("operations = %v, want %v", served, want)
	}
}

// The plugin linked into a host reads the models of the program in the background and lists the
// stored ones until they are read.
func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	program := writeScript(t, "codex", appServerScript)
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
		if bytes.Contains(readDoneFrame(t, answer), []byte("GPT X")) {
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
	body, err := json.Marshal(map[string]any{"operation": llm.OperationModels, "call": map[string]any{"type": "codex"}})
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
