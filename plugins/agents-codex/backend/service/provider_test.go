package service

import (
	"context"
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

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
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

func newCodex(t *testing.T, script string) *agentServer {
	t.Helper()
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": agenttest.WriteScript(t, "codex", script)})
	return svc
}

func recordingCodex(t *testing.T) (string, func() (string, []string)) {
	t.Helper()
	recorded, read := agenttest.RecordingProgram(t, "codex")
	script := agenttest.WriteScript(t, "codex", agenttest.QuoteShell(recorded)+` "$@"`+"\n"+
		`echo '{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}'`+"\n")
	return script, read
}

func userMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

func TestCompleteReturnsTheAnswerOfCodex(t *testing.T) {
	svc := newCodex(t, answerScript)
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("fix this"))

	if err != nil || result.GetText() != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.GetText(), err, "fixed")
	}
}

func TestCompleteStartsCodexWithTheModelTheInstructionsAndNoTool(t *testing.T) {
	script, read := recordingCodex(t)
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": script})

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex", Options: map[string]string{llm.OptionModel: "gpt-x", llm.OptionReasoning: "high"}}, &llm.CompleteRequest{
		Messages: []*llm.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	_, lines := read()
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
	script, read := recordingCodex(t)
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": script})

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, &llm.CompleteRequest{
		Messages: []*llm.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	_, lines := read()
	if got := strings.Join(lines[len(lines)-8:], "\n"); got != "User:\none\n\nAssistant:\ntwo\n\nUser:\nthree" {
		t.Errorf("the prompt is %q", got)
	}
}

// The program is given a read-only sandbox, so a write it attempts is refused. The script writes
// whenever the sandbox is not read-only, which the control run shows.
func TestACompleteCallLetsCodexWriteNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "written")
	t.Setenv(writeTargetVariable, target)
	script := agenttest.WriteScript(t, "codex", answerScript)
	if err := exec.CommandContext(t.Context(), script).Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the control run wrote nothing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": script})

	if _, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Codex wrote the file during complete: %v", err)
	}
}

func TestCompleteRunsCodexInAnEmptyFolderThatIsRemoved(t *testing.T) {
	script, read := recordingCodex(t)
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": script})

	if _, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("hi")); err != nil {
		t.Fatal(err)
	}

	dir, _ := read()
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("the folder %q is still there", dir)
	}
}

func TestCompleteRefusesARequestWithNoUserMessage(t *testing.T) {
	svc := newCodex(t, "exit 0\n")

	for name, request := range map[string]*llm.CompleteRequest{
		"no message":   {},
		"system only":  {Messages: []*llm.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []*llm.Message{{Role: "tool", Text: "x"}}},
	} {
		_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, request)
		if !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAMissingCodexIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newAgentServer(false)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnavailable {
		t.Errorf("reason = %v, want unreachable", reason)
	}
}

func TestAFailedRunOfCodexIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newCodex(t, "echo 'not signed in as sk-secret' >&2\nexit 2\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newCodex(t, "exit 0\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newCodex(t, "sleep 30\n")
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "codex"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnavailable {
		t.Errorf("reason = %v, want timeout", reason)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	svc := newCodex(t, "sleep 30\n")
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := svc.completion.Complete(ctx, connection.Call{Type: "codex"}, userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

func TestTheModelsOfCodexAreListedForTheRow(t *testing.T) {
	svc := newCodex(t, `echo '{"id":1,"result":{}}'`+"\n"+
		`echo '{"id":2,"result":{"data":[{"model":"gpt-x","displayName":"GPT X","isDefault":true,`+
		`"supportedReasoningEfforts":[{"reasoningEffort":"low"},{"reasoningEffort":"high"}]},{"model":"gpt-y"}]}}'`+"\n"+
		"while read -r _; do :; done\n")
	t.Setenv("HOME", t.TempDir())

	models, err := svc.completion.ListModels(t.Context(), connection.Call{Type: "codex"}, []string{llm.OperationComplete})
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
	levels, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "codex", Options: map[string]string{llm.OptionModel: "gpt-x"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "codex", Options: map[string]string{llm.OptionModel: "gpt-y"}})
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

func TestTheProviderServesCompleteAndTheListsOfModelsAndReasoningLevels(t *testing.T) {
	provider := newAgentServer(false).Provider()
	served := slices.Sorted(maps.Keys(provider))
	want := []string{llm.OperationComplete, llm.OperationModels, llm.OperationReasoning}
	if !slices.Equal(served, want) {
		t.Errorf("operations = %v, want %v", served, want)
	}
}

func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))
	script := agenttest.WriteScript(t, "codex", answerScript)
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	lifecycle := server.NewLifecycleClient(http.DefaultClient, web.URL, hostcall.WithBearer("tok"))
	if _, err := lifecycle.Configure(t.Context(), map[string]string{"executablePath": script}); err != nil {
		t.Fatal(err)
	}

	body := `{"call":{"type":"codex","model":"gpt-x"},"messages":[{"role":"user","text":"fix this"}]}`
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
	if response.StatusCode != http.StatusOK || string(answer) != `{"text":"fixed"}` {
		t.Errorf("the service answered %d %q", response.StatusCode, answer)
	}
}
