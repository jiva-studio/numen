package service

import (
	"context"
	"errors"
	"io"
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
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
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

func newClaude(t *testing.T, script string) *agent {
	t.Helper()
	svc := newTestAgent(t)
	svc.settings.Apply(map[string]string{"executablePath": writeScript(t, "claude", script)})
	return svc
}

func userMessage(text string) connection.ProviderCompleteRequest {
	return connection.ProviderCompleteRequest{Messages: []connection.Message{{Role: "user", Text: text}}}
}

func TestCompleteReturnsTheAnswerOfClaude(t *testing.T) {
	svc := newClaude(t, answerScript)
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("fix this"))

	if err != nil || result.Text != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.Text, err, "fixed")
	}
}

func TestCompleteStartsClaudeWithTheModelTheInstructionsAndNoTool(t *testing.T) {
	out := filepath.Join(t.TempDir(), "argv")
	svc := newClaude(t, "cat > "+out+".stdin\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+"\n"+textEvent)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code", Model: "opus"}, connection.ProviderCompleteRequest{
		Messages: []connection.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	argv := readArgv(t, out)
	assertPair(t, argv, "--tools", "")
	assertPair(t, argv, "--append-system-prompt", "be brief")
	assertPair(t, argv, "--model", "opus")
	assertPair(t, argv, "--permission-mode", "dontAsk")
	for _, forbidden := range []string{"--mcp-config", "--allowedTools"} {
		if slices.Contains(argv, forbidden) {
			t.Errorf("%q is among %v", forbidden, argv)
		}
	}
	stdin, err := os.ReadFile(out + ".stdin")
	if err != nil || string(stdin) != "fix this" {
		t.Errorf("the program was asked %q, %v, want %q", stdin, err, "fix this")
	}
}

func TestCompleteJoinsTheTurnsOfAConversationIntoOnePrompt(t *testing.T) {
	out := filepath.Join(t.TempDir(), "stdin")
	svc := newClaude(t, "cat > "+out+"\n"+textEvent)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, connection.ProviderCompleteRequest{
		Messages: []connection.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	stdin, err := os.ReadFile(out)
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
	svc := newTestAgent(t)
	svc.settings.Apply(map[string]string{"executablePath": script})

	if _, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Claude Code wrote the file during complete: %v", err)
	}
}

func TestCompleteRefusesARequestWithNoMessageToAnswer(t *testing.T) {
	svc := newClaude(t, "exit 0\n")

	for name, request := range map[string]connection.ProviderCompleteRequest{
		"no message":   {},
		"system only":  {Messages: []connection.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []connection.Message{{Role: "tool", Text: "x"}}},
	} {
		_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, request)
		if !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAMissingClaudeIsUnreachable(t *testing.T) {
	withoutClaudeAnywhere(t)
	svc := newTestAgent(t)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnreachable {
		t.Errorf("reason = %v, want unreachable", reason)
	}
}

func TestAFailedRunOfClaudeIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newClaude(t, "cat > /dev/null\necho 'not signed in as sk-secret' >&2\nexit 2\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newClaude(t, "cat > /dev/null\nprintf '%s\\n' '{\"type\":\"result\",\"is_error\":false}'\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newClaude(t, "sleep 30\n")
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "claude-code"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonTimeout {
		t.Errorf("reason = %v, want timeout", reason)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	svc := newClaude(t, "sleep 30\n")
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := svc.completion.Complete(ctx, connection.Call{Type: "claude-code"}, userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

// listingAnswer is the answer of a Claude Code that lists two models.
const listingAnswer = `{"type":"control_response","response":{"subtype":"success","request_id":"numen-models","response":{"models":[{"value":"opus","displayName":"Opus","supportsEffort":true,"supportedEffortLevels":["high","low"]},{"value":"haiku","displayName":"Haiku","supportsEffort":false,"supportedEffortLevels":[]}]}}}`

func newListingClaude(t *testing.T) *agent {
	t.Helper()
	return newClaude(t, "read line\nprintf '%s\\n' '"+listingAnswer+"'\nsleep 30\n")
}

func TestTheModelsOfClaudeAreListedForTheRow(t *testing.T) {
	svc := newListingClaude(t)

	models, err := svc.completion.ListModels(t.Context(), connection.Call{Type: "claude-code"}, "complete")

	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range models {
		ids = append(ids, one.ID)
	}
	if !slices.Equal(ids, []string{"opus", "haiku"}) || models[0].Title != "Opus" {
		t.Errorf("models = %+v", models)
	}
}

func TestTheLevelsOfTheModelOfTheCallAreListedFromTheLeastToTheMost(t *testing.T) {
	svc := newListingClaude(t)

	levels, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "claude-code", Model: "opus"})
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "claude-code", Model: "haiku"})
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

func TestListingTheModelsOfAMissingClaudeIsUnreachable(t *testing.T) {
	withoutClaudeAnywhere(t)
	svc := newTestAgent(t)

	_, err := svc.completion.ListModels(t.Context(), connection.Call{Type: "claude-code"}, "complete")

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("ListModels = %v, want try the next", err)
	}
}

func TestTheProviderFillsTheRolesOfCompleteAndModels(t *testing.T) {
	provider := newTestAgent(t).Provider()
	if provider.Completer == nil || provider.Models == nil || provider.Embedder != nil || provider.Profiles != nil {
		t.Errorf("provider = %+v, want complete and the models alone", provider)
	}
}

func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))
	script := writeScript(t, "claude", answerScript)
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

	body := `{"call":{"type":"claude-code"},"messages":[{"role":"user","text":"fix this"}]}`
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
