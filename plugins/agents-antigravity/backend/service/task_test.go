package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Antigravity calls the tools of the host on the server its home names, and each call runs on the host
// with the tool context and the vault of the task. What the program reports reaches the chat as the
// frames of the task, and its calls end with the run.
func TestATaskStreamsTheAntigravityTextAndToolCalls(t *testing.T) {
	host, client := serveGardenHost(t)
	svc := newAntigravity(t, client, fakeAntigravity(t))

	frames, done, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{
		Prompt: "How are the beds?", ToolContext: map[string]string{"md.numen.views.agent.chat/note": "Garden/Beds.md"}, VaultId: "garden",
	})

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("RunTask = %v, %v, want the end of the turn", done, err)
	}
	want := []string{
		`call c1 "Read a note" TOOL_KIND_READ TOOL_CALL_STATUS_IN_PROGRESS [Garden/Beds.md] [Garden/Beds.md]`,
		`chunk m1 "Raised beds."`,
		"update c1 TOOL_CALL_STATUS_COMPLETED",
	}
	if !slices.Equal(frames, want) {
		t.Errorf("frames = %q, want %q", frames, want)
	}
	calls := host.listCalls()
	if len(calls) != 1 || calls[0].Name != "note_read" || calls[0].VaultID != "garden" ||
		calls[0].Context["md.numen.views.agent.chat/note"] != "Garden/Beds.md" || string(calls[0].Arguments) != `{"path":"Garden/Beds.md"}` {
		t.Errorf("the host ran %+v", calls)
	}
}

// A task starts Antigravity to approve its permission requests on its own, with the prompt after the
// instruction sentences of the tools, in an empty folder of its own and a home whose MCP configuration
// names the tools of the vault beside the servers of the person, and reads the configuration of the person.
func TestATaskStartsAntigravityWithTheToolsOfTheVault(t *testing.T) {
	_, client := serveGardenHost(t)
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, client, writeScript(t, "agy", recordRun(out)))
	home := os.Getenv("HOME")
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"), []byte(`{"mcpServers":{"mine":{"serverUrl":"http://127.0.0.1:2/mcp"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "config", "config.json"), []byte(`{"auth":"ok"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "Plan the beds"}); err != nil {
		t.Fatal(err)
	}

	argv := readFile(t, out+".argv")
	if !strings.HasPrefix(argv, "-p\nCite the note you read. Ask before you plan.\n\nPlan the beds\n--output-format\nstream-json\n--dangerously-skip-permissions\n") {
		t.Errorf("the arguments are %q", argv)
	}
	mcp := readFile(t, out+".mcp")
	if !strings.Contains(mcp, `"serverUrl": "http://127.0.0.1:`) || !strings.Contains(mcp, `"Authorization": "Bearer `) || !strings.Contains(mcp, `"mine"`) {
		t.Errorf("the program was given the servers %s", mcp)
	}
	if strings.Contains(argv, "Bearer") {
		t.Errorf("the token is on the command line: %q", argv)
	}
	if cfg := readFile(t, out+".cfg"); cfg != `{"auth":"ok"}` {
		t.Errorf("the program read config.json %q", cfg)
	}
	if files := strings.TrimSpace(readFile(t, out+".files")); files != "" {
		t.Errorf("the folder held %q, want it empty", files)
	}
	if _, err := os.Stat(strings.TrimSpace(readFile(t, out+".dir"))); err == nil {
		t.Error("the folder of the run is still there")
	}
}

// Without a host to list the tools on, Antigravity answers: it approves nothing on its own and its home
// names no server of tools.
func TestWithoutAHostAntigravityAnswersWithNoTool(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", recordRun(out)))

	if _, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}

	if argv := readFile(t, out+".argv"); argv != "-p\nhi\n--output-format\nstream-json\n" {
		t.Errorf("the arguments are %q", argv)
	}
	if mcp := readFile(t, out+".mcp"); strings.Contains(mcp, "serverUrl") {
		t.Errorf("the program was given the servers %s", mcp)
	}
}

// modelListing is the answer of the program to "models".
const modelListing = `if [ "$1" = models ]; then printf 'm-high\tModel (High)\nm-low\tModel (Low)\nplain\tPlain\n'; exit 0; fi` + "\n"

// The level of the row chooses the listed model of its family; the model of the program's own choice is
// not named.
func TestTheModelOfTheLevelIsGivenToAntigravity(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", modelListing+recordRun(out)))
	cases := []struct{ model, level, want string }{
		{"m", "high", "--model\nm-high\n"},
		{"m", "", "--model\nm-low\n"},
		{"m-high", "low", "--model\nm-high\n"},
		{"plain", "high", "--model\nplain\n"},
		{"default", "high", ""},
	}
	for _, one := range cases {
		call := agentCall(map[string]string{llm.OptionModel: one.model, llm.OptionReasoning: one.level})
		if _, _, err := runTask(t.Context(), svc, call, &acp.TaskRequest{Prompt: "hi"}); err != nil {
			t.Fatal(err)
		}
		argv := readFile(t, out+".argv")
		if (one.want == "" && strings.Contains(argv, "--model")) || !strings.Contains(argv, one.want) {
			t.Errorf("%s at %q: the arguments are %q, want %q", one.model, one.level, argv, one.want)
		}
	}
}

func TestTheNotesOfTheTaskComeBeforeThePrompt(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	svc := newAntigravity(t, nil, writeScript(t, "agy", recordRun(out)))

	_, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{
		Prompt:       "What grows here?",
		ContextNotes: []*acp.ContextNote{{Path: "Garden/Beds.md", Title: "Beds", Text: "Raised beds."}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if argv := readFile(t, out+".argv"); !strings.HasPrefix(argv, "-p\nThe note Garden/Beds.md (Beds):\n\nRaised beds.\n\nWhat grows here?\n") {
		t.Errorf("the arguments are %q", argv)
	}
}

func echoConversation(t *testing.T, dir string) string {
	t.Helper()
	return writeScript(t, "agy",
		"asked=$2\nprev=\n"+
			"for a in \"$@\"; do\n"+
			"  if [ \"$prev\" = \"--conversation\" ]; then echo \"$a\" >> "+dir+"/resumed-$asked; fi\n"+
			"  prev=\"$a\"\n"+
			"done\n"+
			"printf '{\"event\":\"init\",\"conversation_id\":\"s-%s\"}\\n' \"$asked\"\n")
}

func readResumeLog(t *testing.T, dir, prompt string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "resumed-"+prompt))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

// Each conversation continues the conversation of the program it started.
func TestEachConversationGoesOnInItsOwn(t *testing.T) {
	dir := t.TempDir()
	svc := newAntigravity(t, nil, echoConversation(t, dir))

	for _, one := range []struct{ conversation, prompt string }{{"one", "left"}, {"two", "right"}, {"one", "left"}, {"", "alone"}, {"", "alone"}} {
		if _, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: one.prompt, Conversation: one.conversation}); err != nil {
			t.Fatal(err)
		}
	}

	if got := readResumeLog(t, dir, "left"); !slices.Equal(got, []string{"s-left"}) {
		t.Errorf("the first conversation resumed %v", got)
	}
	if got := readResumeLog(t, dir, "right"); got != nil {
		t.Errorf("the second conversation resumed %v", got)
	}
	if got := readResumeLog(t, dir, "alone"); got != nil {
		t.Errorf("a task in no conversation resumed %v", got)
	}
}

func TestAMissingAntigravityIsAFailureToTryTheNextConnectionAfter(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newTestAgent(t, nil)

	frames, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) || len(frames) != 0 {
		t.Errorf("RunTask = %q, %v, want try the next as unavailable", frames, err)
	}
}

// A run that fails says what the program said of it as a message of its own, and stops.
func TestAFailedRunReportsTheReasonAndStops(t *testing.T) {
	cases := map[string]struct{ script, want string }{
		"error result": {
			`echo '{"event":"result","result":{"status":"ERROR","response":"","error":"Eligibility check failed: 503"}}'` + "\n" +
				`echo 'error: Eligibility check failed: 503' >&2` + "\nexit 1\n",
			"Eligibility check failed: 503",
		},
		"unknown model": {
			`echo 'error: invalid model selection (--model "m-old"): not recognized' >&2` + "\n" +
				`printf 'Available models:\n  Model One\n  Model Two\n' >&2` + "\nexit 1\n",
			`invalid model selection (--model "m-old"): not recognized`,
		},
		"stopped": {`echo '{"stopped":"the quota is used up"}'` + "\n", "the quota is used up"},
	}
	for name, one := range cases {
		svc := newAntigravity(t, nil, writeScript(t, "agy", one.script))

		frames, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

		if !connection.IsStop(err) || !slices.Equal(frames, []string{`chunk m1 "` + strings.ReplaceAll(one.want, `"`, `\"`) + `"`}) {
			t.Errorf("%s: RunTask = %q, %v, want %q and a stop", name, frames, err, one.want)
		}
	}
}

// The answer arrives as the text deltas of the response steps, and the result that repeats it is not
// sent again. A line that is not an event is text.
func TestTheAnswerOfAntigravityIsSentOnce(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy",
		`echo '{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":"ok"}}'`+"\n"+
			`echo 'plain line'`+"\n"+
			`echo '{"event":"result","result":{"status":"SUCCESS","response":"ok"}}'`+"\n"))

	frames, done, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN ||
		!slices.Equal(frames, []string{`chunk m1 "ok"`, `chunk m1 "plain line\n"`}) {
		t.Errorf("RunTask = %q, %v, %v", frames, done, err)
	}
}

// A result event ends the run even when a child process keeps the output pipe open.
func TestAResultEndsTheRunWithAnOpenChildOutput(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy",
		"(sleep 60) &\n"+
			`echo '{"event":"result","result":{"status":"SUCCESS","response":"done"}}'`+"\n"+
			"wait\n"))
	// The child outlives the deadline, so a run that waits for its output to close ends with the deadline.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	frames, _, err := runTask(ctx, svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if err != nil || !slices.Equal(frames, []string{`chunk m1 "done"`}) {
		t.Errorf("RunTask = %q, %v", frames, err)
	}
}

func TestAHomeThatCannotBeSetUpStopsTheTask(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "exit 0\n"))
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, filepath.Join(t.TempDir(), "absent"))
	}

	frames, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if !connection.IsStop(err) || len(frames) != 0 {
		t.Errorf("RunTask = %q, %v, want a stop", frames, err)
	}
}

func TestACanceledTaskEndsTheProgram(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "sleep 30\n"))
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, _, err := runTask(ctx, svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunTask = %v, want the cancellation", err)
	}
}

func TestClosingThePluginEndsATaskThatRuns(t *testing.T) {
	svc := newAntigravity(t, nil, writeScript(t, "agy", "sleep 30\n"))
	ended := make(chan error, 1)
	go func() {
		_, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"})
		ended <- err
	}()
	time.Sleep(200 * time.Millisecond)

	if err := svc.tasks.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-ended:
		if !connection.IsStop(err) {
			t.Errorf("RunTask = %v, want a stop", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the task still runs")
	}
	if _, _, err := runTask(t.Context(), svc, agentCall(nil), &acp.TaskRequest{Prompt: "hi"}); !connection.IsStop(err) {
		t.Errorf("a task after the close = %v, want a stop", err)
	}
}
