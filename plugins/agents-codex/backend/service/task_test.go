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

// Codex calls the tools of the host on the server the plugin serves it, and each call runs on the host
// with the tool context and the vault of the task. What the program reports reaches the chat as the
// frames of the task.
func TestATaskStreamsTheCodexTextAndToolCalls(t *testing.T) {
	host, client := serveGardenHost(t)
	svc := newCodex(t, client, fakeCodex(t))

	frames, done, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{
		Prompt: "How are the beds?", ToolContext: map[string]string{"md.numen.views.agent.chat/note": "Garden/Beds.md"}, VaultId: "garden",
	})

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("RunTask = %v, %v, want the end of the turn", done, err)
	}
	want := []string{
		`call i1 "Read a note" TOOL_KIND_READ TOOL_CALL_STATUS_IN_PROGRESS [Garden/Beds.md] [Garden/Beds.md]`,
		"update i1 TOOL_CALL_STATUS_COMPLETED",
		`chunk m1 "Raised beds."`,
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

// A task with tools runs `codex exec --json` without approvals or a sandbox, in the folder of the
// plugin, with the server of the tools named on the command line and its token in NUMEN_TOKEN alone.
// The instruction sentences of the tools come before the prompt, which the program reads on its input.
func TestATaskPassesCodexTheToolServerAndItsTokenInTheEnvironment(t *testing.T) {
	_, client := serveGardenHost(t)
	dir := t.TempDir()
	svc := newCodex(t, client, writeScript(t, "codex", recordArgs(dir)))

	call := taskCall(map[string]string{llm.OptionModel: "gpt-x", llm.OptionReasoning: "high"})
	if _, _, err := runTask(t.Context(), svc, call, &acp.TaskRequest{Prompt: "Plan the beds"}); err != nil {
		t.Fatal(err)
	}

	argv := readArgv(t, dir)
	if argv[0] != "exec" || argv[1] != "--json" || argv[len(argv)-1] != "-" {
		t.Errorf("the arguments are %q", argv)
	}
	if stdin := readFile(t, filepath.Join(dir, "stdin")); stdin != "Cite the note you read. Ask before you plan.\n\nPlan the beds" {
		t.Errorf("the input is %q", stdin)
	}
	joined := strings.Join(argv, "\n")
	for _, want := range []string{
		"--dangerously-bypass-approvals-and-sandbox", "--model\ngpt-x", "-c\nmodel_reasoning_effort=\"high\"",
		"-c\nmcp_servers.numen.url=\"http://127.0.0.1:", "-c\nmcp_servers.numen.bearer_token_env_var=\"NUMEN_TOKEN\"",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q absent from %q", want, argv)
		}
	}
	token := readFile(t, filepath.Join(dir, "token"))
	if len(token) < 32 {
		t.Fatalf("the program has NUMEN_TOKEN=%q", token)
	}
	if strings.Contains(joined, token) {
		t.Errorf("the token is on the command line: %q", argv)
	}
	if pwd := strings.TrimSpace(readFile(t, filepath.Join(dir, "pwd"))); !strings.Contains(filepath.Base(pwd), "numen-codex-tasks-") {
		t.Errorf("the task ran in %q, want the folder of the plugin", pwd)
	}
}

// Without a host to list the tools on, Codex answers with no tool, in a read-only sandbox and an empty
// folder that is removed.
func TestWithoutAHostCodexAnswersWithNoTool(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, writeScript(t, "codex", recordArgs(dir)))

	if _, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "the question"}); err != nil {
		t.Fatal(err)
	}

	argv := readArgv(t, dir)
	joined := strings.Join(argv, "\n")
	if argv[0] != "exec" || argv[len(argv)-1] != "-" || readFile(t, filepath.Join(dir, "stdin")) != "the question" {
		t.Errorf("the arguments are %q", argv)
	}
	for _, want := range []string{"--ignore-user-config", "--ephemeral", "--sandbox\nread-only", "--disable\nshell_tool", "web_search=\"disabled\""} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q absent from %q", want, argv)
		}
	}
	for _, forbidden := range []string{"--dangerously-bypass-approvals-and-sandbox", "mcp_servers"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("%q is among %q", forbidden, argv)
		}
	}
	if _, err := os.Stat(strings.TrimSpace(readFile(t, filepath.Join(dir, "pwd")))); err == nil {
		t.Error("the folder of the answer is still there")
	}
}

// The notes the person attached come before the prompt, each under the path and the title of the note.
func TestTheNotesOfTheTaskComeBeforeThePrompt(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, writeScript(t, "codex", recordArgs(dir)))

	_, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{
		Prompt:       "What grows here?",
		ContextNotes: []*acp.ContextNote{{Path: "Garden/Beds.md", Title: "Beds", Text: "Raised beds."}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if stdin := readFile(t, filepath.Join(dir, "stdin")); stdin != "The note Garden/Beds.md (Beds):\n\nRaised beds.\n\nWhat grows here?" {
		t.Errorf("the input is %q", stdin)
	}
}

// Every line Codex writes that is no event is shown as text, in order with the events, and the reasoning
// of the model as a thought.
func TestALineThatIsNoEventIsShownAsText(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex",
		`echo '{"type":"item.started","item":{"id":"i1","type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"id":"i1","type":"mcp_tool_call","status":"failed"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"reasoning","text":"hidden"}}'`+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n"+
			"echo plain text\n"))

	frames, done, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})

	want := []string{
		`call i1 "read_note" TOOL_KIND_OTHER TOOL_CALL_STATUS_IN_PROGRESS [a.md] []`,
		"update i1 TOOL_CALL_STATUS_FAILED",
		`thought t1 "hidden"`,
		`chunk m1 "plain text\n"`,
	}
	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN || !slices.Equal(frames, want) {
		t.Errorf("RunTask = %q, %v, %v, want %q", frames, done, err, want)
	}
}

func threadScript(t *testing.T, dir string) string {
	t.Helper()
	return writeScript(t, "codex",
		"asked=$(cat)\n"+
			"printf '%s\\n' \"$*\" >> "+quoteShell(dir)+"/\"$asked\"\n"+
			"for a in \"$@\"; do if [ \"$a\" = thread-gone ]; then echo 'error: thread/resume: no rollout found' >&2; exit 1; fi; done\n"+
			"printf '{\"type\":\"thread.started\",\"thread_id\":\"thread-%s\"}\\n' \"$asked\"\n"+
			"echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"ok\"}}'\n")
}

func readRuns(t *testing.T, dir, prompt string) []string {
	t.Helper()
	return strings.Split(strings.TrimSpace(readFile(t, filepath.Join(dir, prompt))), "\n")
}

// The first turn of a conversation starts a thread that is kept, and the next turn continues it; a turn
// of no conversation keeps nothing.
func TestTheTurnsOfAConversationContinueItsThread(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, threadScript(t, dir))

	for _, one := range []struct{ conversation, prompt string }{{"one", "beds"}, {"one", "beds"}, {"", "alone"}} {
		if _, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: one.prompt, Conversation: one.conversation}); err != nil {
			t.Fatal(err)
		}
	}

	runs := readRuns(t, dir, "beds")
	if len(runs) != 2 || strings.Contains(runs[0], "resume") || strings.Contains(runs[0], "--ephemeral") ||
		!strings.HasSuffix(runs[1], " resume thread-beds -") {
		t.Errorf("the runs of the conversation were %q", runs)
	}
	if alone := readRuns(t, dir, "alone"); len(alone) != 1 || strings.Contains(alone[0], "resume") || !strings.Contains(alone[0], "--ephemeral") {
		t.Errorf("the run of no conversation was %q", alone)
	}
}

// A thread the program no longer has is started anew in the same turn, which answers as a first turn does.
// The script fails, as Codex does, when it is asked to resume the thread of the prompt "gone".
func TestAThreadTheProgramNoLongerHasIsStartedAnew(t *testing.T) {
	dir := t.TempDir()
	svc := newCodex(t, nil, threadScript(t, dir))
	request := &acp.TaskRequest{Prompt: "gone", Conversation: "one"}
	if _, _, err := runTask(t.Context(), svc, taskCall(nil), request); err != nil {
		t.Fatal(err)
	}

	frames, done, err := runTask(t.Context(), svc, taskCall(nil), request)

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN || !slices.Equal(frames, []string{`chunk m1 "ok"`}) {
		t.Errorf("RunTask = %q, %v, %v, want the answer of a new thread", frames, done, err)
	}
	runs := readRuns(t, dir, "gone")
	if len(runs) != 3 || strings.Contains(runs[0], "resume") || !strings.HasSuffix(runs[1], " resume thread-gone -") ||
		strings.Contains(runs[2], "resume") {
		t.Errorf("the runs were %q, want a new thread, the failed resume and a new thread", runs)
	}
}

func TestAMissingCodexIsAFailureToTryTheNextConnectionAfter(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := newTestServer(t, nil)

	frames, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if _, isTryNext := connection.IsTryNext(err); !isTryNext || !hasReason(err, connection.ReasonUnavailable) || len(frames) != 0 {
		t.Errorf("RunTask = %q, %v, want try the next as unavailable", frames, err)
	}
}

// A run that fails says what the program said of it as a message of its own, and stops.
func TestAFailedRunReportsTheReasonAndStops(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "echo 'not signed in' >&2\nexit 2\n"))

	frames, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if !connection.IsStop(err) || !slices.Equal(frames, []string{`chunk m1 "not signed in"`}) {
		t.Errorf("RunTask = %q, %v, want the reason and a stop", frames, err)
	}
}

// A run that writes nothing but an error says the error, though the program exits well.
func TestARunThatWritesOnlyAnErrorReportsIt(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "echo 'error: the model is overloaded' >&2\n"))

	frames, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if !connection.IsStop(err) || !slices.Equal(frames, []string{`chunk m1 "the model is overloaded"`}) {
		t.Errorf("RunTask = %q, %v", frames, err)
	}
}

func TestTheFirstFrameTheChatRefusesEndsTheTask(t *testing.T) {
	svc := newCodex(t, nil, fakeCodex(t))
	refused := errors.New("the chat is gone")

	_, err := svc.tasks.RunTask(t.Context(), taskCall(nil), &acp.TaskRequest{Prompt: "hi"}, func(*acp.TaskFrame) error {
		return refused
	})

	if !errors.Is(err, refused) {
		t.Errorf("RunTask = %v, want the error of the send", err)
	}
}

func TestACanceledTaskEndsTheProgram(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "sleep 30\n"))
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, _, err := runTask(ctx, svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunTask = %v, want the cancellation", err)
	}
}

func TestClosingThePluginEndsATaskThatRuns(t *testing.T) {
	svc := newCodex(t, nil, writeScript(t, "codex", "sleep 30\n"))
	ended := make(chan error, 1)
	go func() {
		_, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"})
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
	if _, _, err := runTask(t.Context(), svc, taskCall(nil), &acp.TaskRequest{Prompt: "hi"}); !connection.IsStop(err) {
		t.Errorf("a task after the close = %v, want a stop", err)
	}
}
