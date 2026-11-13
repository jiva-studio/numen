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

const resultEvent = `echo '{"event":"result","result":{"status":"SUCCESS","response":"fixed"}}'` + "\n"

// answerScript is an Antigravity that says "fixed" and writes the file named by the variable when it was
// started to approve every permission request on its own.
const answerScript = modelListing + `for a in "$@"; do
  if [ "$a" = "--dangerously-skip-permissions" ]; then echo written > "$` + writeTargetVariable + `"; fi
done
` + resultEvent

func userMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

func complete(t *testing.T, svc *agent, call connection.Call, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	t.Helper()
	return llmtest.Complete(t.Context(), svc.Provider(), call, request)
}

func TestCompleteReturnsTheAnswerOfAntigravity(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", answerScript))
	t.Setenv(writeTargetVariable, filepath.Join(t.TempDir(), "written"))

	result, err := complete(t, svc, agentCall(nil), userMessage("fix this"))

	if err != nil || result.GetText() != "fixed" {
		t.Errorf("Complete = %q, %v, want %q", result.GetText(), err, "fixed")
	}
}

func TestCompleteStartsAntigravityWithTheModelOfTheLevelAndTheInstructions(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", modelListing+recordRun(out)+resultEvent))

	_, err := complete(t, svc, agentCall(map[string]string{llm.OptionModel: "m", llm.OptionReasoning: "high"}), &llm.CompleteRequest{
		Messages: []*llm.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "fix this"}},
	})

	if err != nil {
		t.Fatal(err)
	}
	if argv := readFile(t, out+".argv"); argv != "--print=\n--input-format\nstream-json\n--output-format\nstream-json\n--model\nm-high\n" {
		t.Errorf("the arguments are %q, want the model of the level", argv)
	}
	if prompt := readPrompt(t, out); prompt != "be brief\n\nfix this" {
		t.Errorf("the prompt is %q, want the instructions and the question", prompt)
	}
}

func TestCompleteJoinsTheTurnsOfAConversationIntoOnePrompt(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", recordRun(out)+resultEvent))

	_, err := complete(t, svc, agentCall(nil), &llm.CompleteRequest{
		Messages: []*llm.Message{
			{Role: "user", Text: "one"}, {Role: "assistant", Text: "two"}, {Role: "user", Text: "three"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if prompt := readPrompt(t, out); !strings.HasPrefix(prompt, "User:\none\n\nAssistant:\ntwo\n\nUser:\nthree") {
		t.Errorf("the prompt is %q", prompt)
	}
}

// The program is not started to approve its permission requests, so a write it attempts is refused.
// The script writes when it is, which the control run shows.
func TestACompleteCallLetsAntigravityWriteNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "written")
	t.Setenv(writeTargetVariable, target)
	script := writeScript(t, "agy", answerScript)
	if err := exec.CommandContext(t.Context(), script, "--dangerously-skip-permissions").Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the control run wrote nothing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	svc := newAntigravity(t, nil, script)

	if _, err := complete(t, svc, agentCall(nil), userMessage("write a file")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Antigravity wrote the file during complete: %v", err)
	}
}

// The program runs in an empty folder of its own, with a home that holds no server of tools.
func TestCompleteRunsAntigravityInAnEmptyFolderWithNoToolServer(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", recordRun(out)+resultEvent))

	if _, err := complete(t, svc, agentCall(nil), userMessage("hi")); err != nil {
		t.Fatal(err)
	}

	if files := strings.TrimSpace(readFile(t, out+".files")); files != "" {
		t.Errorf("the folder held %q, want it empty", files)
	}
	if mcp := readFile(t, out+".mcp"); strings.Contains(mcp, "url") || strings.Contains(mcp, "Url") {
		t.Errorf("the program was given the servers %s", mcp)
	}
	if _, err := os.Stat(strings.TrimSpace(readFile(t, out+".dir"))); err == nil {
		t.Error("the folder of the run is still there")
	}
}

func TestCompleteRefusesARequestWithNoMessageToAnswer(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "exit 0\n"))

	for name, request := range map[string]*llm.CompleteRequest{
		"no message":   {},
		"system only":  {Messages: []*llm.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []*llm.Message{{Role: "tool", Text: "x"}}},
	} {
		if _, err := complete(t, svc, agentCall(nil), request); !connection.IsStop(err) {
			t.Errorf("%s: Complete = %v, want a stop", name, err)
		}
	}
}

func TestAMissingAntigravityIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newTestAgent(t, nil)

	_, err := complete(t, svc, agentCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestAFailedRunOfAntigravityIsAStopWithNoTextOfTheProgram(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "echo 'error: not signed in as sk-secret' >&2\nexit 2\n"))

	_, err := complete(t, svc, agentCall(nil), userMessage("hi"))

	if !connection.IsStop(err) {
		t.Fatalf("Complete = %v, want a stop", err)
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "signed in") {
		t.Errorf("the error shows the text of the program: %q", err)
	}
}

func TestAnEmptyAnswerIsAStop(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "exit 0\n"))

	if _, err := complete(t, svc, agentCall(nil), userMessage("hi")); !connection.IsStop(err) {
		t.Errorf("Complete = %v, want a stop", err)
	}
}

func TestCompleteEndsAtTheRequestTimeoutAsATimeout(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "sleep 30\n"))
	svc.settings.Apply(map[string]string{"requestTimeout": "0.2"})

	_, err := complete(t, svc, agentCall(nil), userMessage("hi"))

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) {
		t.Fatalf("Complete = %v, want try the next as unavailable", err)
	}
}

func TestCompleteReturnsTheErrorOfACanceledContext(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "sleep 30\n"))
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := llmtest.Complete(ctx, svc.Provider(), agentCall(nil), userMessage("hi"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

// The models are listed with "models", each family of levels as one model whose levels run from the
// least to the most.
func TestTheModelsOfAntigravityAreListedForTheRow(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", `if [ "$1" = models ]; then printf 'Fetching available models...\nm-high\tModel (High)\nm-low\tModel (Low)\nplain\tPlain\n'; fi`+"\n"))

	models, err := svc.completion.ListModels(t.Context(), agentCall(nil), []string{llm.OperationComplete})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, one := range models {
		ids = append(ids, one.Value)
	}
	if !slices.Equal(ids, []string{"m", "plain"}) || models[0].Title != "Model" {
		t.Errorf("models = %+v", models)
	}
	levels, err := svc.completion.ListReasoningLevels(t.Context(), agentCall(map[string]string{llm.OptionModel: "m"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0].Value != "low" || levels[1].Value != "high" || levels[1].Title != "High" {
		t.Errorf("levels = %+v", levels)
	}
	none, err := svc.completion.ListReasoningLevels(t.Context(), agentCall(map[string]string{llm.OptionModel: "plain"}))
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model with none = %+v, %v", none, err)
	}
}

func TestAProgramWhoseListFailsListsNoModels(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "exit 1\n"))

	models, err := svc.completion.ListModels(t.Context(), agentCall(nil), nil)

	if err != nil || len(models) != 0 {
		t.Errorf("ListModels = %+v, %v", models, err)
	}
}

func TestListingTheModelsOfAMissingAntigravityIsUnreachable(t *testing.T) {
	withoutAgentsAnywhere(t)

	_, err := newTestAgent(t, nil).completion.ListModels(t.Context(), agentCall(nil), nil)

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("ListModels = %v, want try the next", err)
	}
}

func TestTheProviderServesCompleteTheListsOfModelsAndReasoningLevelsAndTasks(t *testing.T) {
	served := slices.Sorted(maps.Keys(newTestAgent(t, nil).Provider()))
	want := []string{acp.OperationTask, llm.OperationComplete, llm.OperationModels, llm.OperationReasoning}
	if !slices.Equal(served, want) {
		t.Errorf("operations = %v, want %v", served, want)
	}
}

// The plugin linked into a host reads the models of the program in the background and lists none until
// they are read.
func TestTheLinkedHandlerServesTheProviderService(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	program := writeScript(t, "agy", modelListing)
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
		if bytes.Contains(readDoneFrame(answer), []byte("Plain")) {
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
	body, err := json.Marshal(map[string]any{"operation": llm.OperationModels, "call": map[string]any{"type": "antigravity"}})
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

func readDoneFrame(answer []byte) []byte {
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
