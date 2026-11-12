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

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const writeTargetVariable = "NUMEN_TEST_WRITE_TARGET"

const resultEvent = `echo '{"event":"result","result":{"status":"SUCCESS","response":"fixed"}}'` + "\n"

// modelListing is the answer of the program to "models".
const modelListing = `if [ "$1" = models ]; then printf 'm-high\tModel (High)\nm-low\tModel (Low)\nplain\tPlain\n'; exit 0; fi` + "\n"

// answerScript is an Antigravity that says "fixed" and writes the file named by the variable when it was
// started to approve every permission request on its own.
const answerScript = modelListing + `for a in "$@"; do
  if [ "$a" = "--dangerously-skip-permissions" ]; then echo written > "$` + writeTargetVariable + `"; fi
done
` + resultEvent

func newAntigravity(t *testing.T, script string) *agentServer {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	svc := newAgentServer(false)
	svc.settings.Apply(map[string]string{"executablePath": agenttest.WriteScript(t, "agy", script)})
	return svc
}

func userMessage(text string) connection.ProviderCompleteRequest {
	return connection.ProviderCompleteRequest{Messages: []connection.Message{{Role: "user", Text: text}}}
}

func TestCompleteReturnsTheAnswerOfAntigravity(t *testing.T) {
	svc := newAntigravity(t, answerScript)
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("fix this"))

	if err != nil || result.Text != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.Text, err, "fixed")
	}
}

func TestCompleteStartsAntigravityWithTheModelOfTheLevelAndTheInstructions(t *testing.T) {
	recorded, read := agenttest.RecordingProgram(t, "agy")
	svc := newAntigravity(t, modelListing+agenttest.QuoteShell(recorded)+` "$@"`+"\n"+resultEvent)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity", Model: "m", Reasoning: "high"}, connection.ProviderCompleteRequest{
		Messages: []connection.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	_, lines := read()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "-p\nbe brief\n\nfix this") || !strings.Contains(joined, "--model\nm-high") {
		t.Errorf("the arguments are %v, want the instructions, the question and the model of the level", lines)
	}
}

func TestCompleteJoinsTheTurnsOfAConversationIntoOnePrompt(t *testing.T) {
	recorded, read := agenttest.RecordingProgram(t, "agy")
	svc := newAntigravity(t, agenttest.QuoteShell(recorded)+` "$@"`+"\n"+resultEvent)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, connection.ProviderCompleteRequest{
		Messages: []connection.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	_, lines := read()
	if joined := strings.Join(lines, "\n"); !strings.Contains(joined, "-p\nUser:\none\n\nAssistant:\ntwo\n\nUser:\nthree\n") {
		t.Errorf("the arguments are %v", lines)
	}
}

// The program is not started to approve its permission requests, so a write it attempts is refused.
// The script writes when it is, which the control run shows.
func TestACompleteCallLetsAntigravityWriteNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "written")
	t.Setenv(writeTargetVariable, target)
	svc := newAntigravity(t, answerScript)
	script := svc.settings.ExecutablePath()
	if err := exec.CommandContext(t.Context(), script, "--dangerously-skip-permissions").Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the control run wrote nothing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Antigravity wrote the file during complete: %v", err)
	}
}

// The program runs in an empty folder of its own, with a home that holds no server of tools.
func TestCompleteRunsAntigravityInAnEmptyFolderWithNoToolServer(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, "pwd > "+agenttest.QuoteShell(out+".dir")+"\n"+
		"ls -A > "+agenttest.QuoteShell(out+".files")+"\n"+
		"cat \"$HOME/.gemini/config/mcp_config.json\" > "+agenttest.QuoteShell(out+".mcp")+"\n"+resultEvent)

	if _, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("hi")); err != nil {
		t.Fatal(err)
	}

	files, err := os.ReadFile(out + ".files")
	if err != nil || strings.TrimSpace(string(files)) != "" {
		t.Errorf("the folder held %q, %v, want it empty", files, err)
	}
	mcp, err := os.ReadFile(out + ".mcp")
	if err != nil || strings.Contains(string(mcp), "url") || strings.Contains(string(mcp), "Url") {
		t.Errorf("the program was given the servers %s, %v", mcp, err)
	}
	dir, err := os.ReadFile(out + ".dir")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(dir))); err == nil {
		t.Errorf("the folder %q is still there", dir)
	}
}

func TestCompleteRefusesARequestWithNoMessageToAnswer(t *testing.T) {
	svc := newAntigravity(t, "exit 0\n")

	for name, request := range map[string]connection.ProviderCompleteRequest{
		"no message":   {},
		"system only":  {Messages: []connection.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []connection.Message{{Role: "tool", Text: "x"}}},
	} {
		_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, request)
		if !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAMissingAntigravityIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newAgentServer(false)

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonUnreachable {
		t.Errorf("reason = %v, want unreachable", reason)
	}
}

func TestAFailedRunOfAntigravityIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newAntigravity(t, "echo 'error: not signed in as sk-secret' >&2\nexit 2\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newAntigravity(t, "exit 0\n")

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("hi"))

	if !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newAntigravity(t, "sleep 30\n")
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := svc.completion.Complete(t.Context(), connection.Call{Type: "antigravity"}, userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete = %v, want try the next", err)
	}
	if reason, _ := connection.GetReason(err); reason != connection.ReasonTimeout {
		t.Errorf("reason = %v, want timeout", reason)
	}
}

func TestCompleteReturnsTheErrorOfACancelledContext(t *testing.T) {
	svc := newAntigravity(t, "sleep 30\n")
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := svc.completion.Complete(ctx, connection.Call{Type: "antigravity"}, userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

func TestTheModelsOfAntigravityAreListedForTheRow(t *testing.T) {
	svc := newAntigravity(t, modelListing)

	models, err := svc.completion.ListModels(t.Context(), connection.Call{Type: "antigravity"}, "complete")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range models {
		ids = append(ids, one.ID)
	}
	if !slices.Equal(ids, []string{"m", "plain"}) || models[0].Title != "Model" {
		t.Errorf("models = %+v", models)
	}
	levels, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "antigravity", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), connection.Call{Type: "antigravity", Model: "plain"})
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

func TestTheProviderFillsTheRolesOfCompleteAndModels(t *testing.T) {
	provider := newAgentServer(false).Provider()
	if provider.Completer == nil || provider.Models == nil || provider.Embedder != nil || provider.Profiles != nil {
		t.Errorf("provider = %+v, want complete and the models alone", provider)
	}
}

func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))
	script := agenttest.WriteScript(t, "agy", answerScript)
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

	body := `{"call":{"type":"antigravity"},"messages":[{"role":"user","text":"fix this"}]}`
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
