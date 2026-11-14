package task

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
)

func TestConstantsAndFailureMessages(t *testing.T) {
	if listLimit != 10*time.Second || maxLine != 4<<20 || maxSessions != 64 {
		t.Fatalf("constants = (%v, %d, %d)", listLimit, maxLine, maxSessions)
	}
	baseErr := errors.New("exit 1")
	if got := getFailureMessage(baseErr, "  \n  "); got != "exit 1" {
		t.Errorf("getFailureMessage empty output = %q, want exit 1", got)
	}
	if got := getFailureMessage(baseErr, "info: starting\n  Error:  bad token \ntrailing"); got != "bad token" {
		t.Errorf("getFailureMessage error line = %q, want bad token", got)
	}
	if got := getFailureMessage(baseErr, "error:\nlast line of stderr\n"); got != "last line of stderr" {
		t.Errorf("getFailureMessage fallback line = %q, want last line of stderr", got)
	}
}

func TestSessionsKeepMostRecentThreadsUpToLimit(t *testing.T) {
	var store sessions
	store.set("", "t-empty")
	store.set("k-empty", "")
	if got := store.get(""); got != "" {
		t.Errorf("get empty = %q, want empty", got)
	}
	if got := store.get("k-empty"); got != "" || len(store.order) != 0 {
		t.Errorf("after ignored sets: get=%q order=%v", got, store.order)
	}

	for i := range maxSessions {
		store.set("k"+strconv.Itoa(i), "t"+strconv.Itoa(i))
	}
	// Refresh k0 so it moves to the end of the order and k1 becomes the oldest.
	store.set("k0", "t0-updated")
	store.set("k-extra", "t-extra")

	if len(store.order) != maxSessions {
		t.Fatalf("len(order) = %d, want %d", len(store.order), maxSessions)
	}
	if got := store.get("k0"); got != "t0-updated" {
		t.Errorf("get(k0) = %q, want t0-updated", got)
	}
	if got := store.get("k1"); got != "" {
		t.Errorf("get(k1) = %q, want evicted empty", got)
	}
	if got := store.get("k-extra"); got != "t-extra" {
		t.Errorf("get(k-extra) = %q, want t-extra", got)
	}
}

func TestTurnFramesTrackMessagesThoughtsAndToolCalls(t *testing.T) {
	var sent []string
	var sendErr error
	listed := indexTools([]*tools.ToolInfo{
		{Name: "note_read", Title: "Read a note", Effect: tools.ReadOnly},
		{Name: "note_write", Title: "", Effect: tools.ReadWrite},
	})
	frames := newTurn(listed, func(frame *contract.TaskFrame) error {
		if sendErr != nil {
			return sendErr
		}
		switch u := frame.GetUpdate().(type) {
		case *contract.TaskFrame_AgentMessageChunk:
			sent = append(sent, fmt.Sprintf("msg %s:%s", u.AgentMessageChunk.GetMessageId(), u.AgentMessageChunk.GetText()))
		case *contract.TaskFrame_AgentThoughtChunk:
			sent = append(sent, fmt.Sprintf("thought %s:%s", u.AgentThoughtChunk.GetMessageId(), u.AgentThoughtChunk.GetText()))
		case *contract.TaskFrame_ToolCall:
			tc := u.ToolCall
			var locs []string
			for _, l := range tc.GetLocations() {
				locs = append(locs, l.GetPath())
			}
			sent = append(sent, fmt.Sprintf("call %s:%s:%s:%s:%v:%v", tc.GetToolCallId(), tc.GetTitle(), tc.GetKind(), tc.GetStatus(), tc.GetContent(), locs))
		case *contract.TaskFrame_ToolCallUpdate:
			tu := u.ToolCallUpdate
			sent = append(sent, fmt.Sprintf("update %s:%s:%v", tu.GetToolCallId(), tu.GetStatus(), tu.GetContent()))
		}
		return nil
	})

	frames.RecordThread("thread-1")
	if frames.thread != "thread-1" || frames.isSent {
		t.Fatalf("thread=%q isSent=%v", frames.thread, frames.isSent)
	}

	if err := frames.SendText("hello "); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendText("world"); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendThought("thinking 1"); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendThought("thinking 2"); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendMessage("standalone"); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendText("after standalone"); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolCall("c1", "note_read", ""); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolCall("c1", "note_read", `{"path":"a.md","query":"beds"}`); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolResult("c1", false); err != nil {
		t.Fatal(err)
	}
	// Repeat calls after finish and unknown tool results are ignored.
	if err := frames.SendToolCall("c1", "note_read", `{}`); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolResult("c1", true); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolResult("unknown", false); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolCall("c2", "note_write", `{"count":1}`); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolResult("c2", true); err != nil {
		t.Fatal(err)
	}
	if err := frames.SendToolCall("c3", "custom_tool", `{"path":"b.md"}`); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"msg m1:hello ",
		"msg m1:world",
		"thought t2:thinking 1",
		"thought t2:thinking 2",
		"msg m2:standalone",
		"msg m3:after standalone",
		"call c1:Read a note:TOOL_KIND_READ:TOOL_CALL_STATUS_PENDING:[]:[]",
		"update c1:TOOL_CALL_STATUS_IN_PROGRESS:[beds]",
		"update c1:TOOL_CALL_STATUS_COMPLETED:[]",
		"call c2:note_write:TOOL_KIND_EDIT:TOOL_CALL_STATUS_IN_PROGRESS:[]:[]",
		"update c2:TOOL_CALL_STATUS_FAILED:[]",
		"call c3:custom_tool:TOOL_KIND_OTHER:TOOL_CALL_STATUS_IN_PROGRESS:[b.md]:[]",
	}
	if !slices.Equal(sent, want) {
		t.Errorf("sent =\n%v\nwant\n%v", sent, want)
	}

	refused := errors.New("sink closed")
	sendErr = refused
	if err := frames.SendMessage("fail"); !errors.Is(err, refused) || !errors.Is(frames.sendErr, refused) {
		t.Errorf("SendMessage error = %v, sendErr = %v, want %v", err, frames.sendErr, refused)
	}
}

func TestScanOutputCountsReportsAndHandlesLargeLines(t *testing.T) {
	largeText := strings.Repeat("x", 70000)
	input := strings.Join([]string{
		`{"type":"thread.started","thread_id":"th-1"}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"` + largeText + `"}}`,
		`{"type":"item.completed","item":{"type":"reasoning","text":"thought"}}`,
		`{"type":"item.started","item":{"id":"i1","type":"mcp_tool_call","name":"t","arguments":{}}}`,
		`{"type":"item.completed","item":{"id":"i1","type":"mcp_tool_call","status":"completed"}}`,
		"   ",
		"plain line",
	}, "\n") + "\n"

	var col collector
	res := scanOutput(strings.NewReader(input), &col)
	if res.sendErr != nil || res.scanErr != nil || res.reports != 5 {
		t.Fatalf("scanOutput = %+v, want 5 reports and no errors", res)
	}
	if got := col.text.String(); got != largeText+"plain line\n" {
		t.Errorf("collector text len = %d, want %d", len(got), len(largeText+"plain line\n"))
	}

	tooLong := strings.Repeat("y", maxLine+10) + "\n"
	longRes := scanOutput(strings.NewReader(tooLong), &col)
	if !errors.Is(longRes.scanErr, bufio.ErrTooLong) {
		t.Errorf("scanOutput too long = %+v, want bufio.ErrTooLong", longRes)
	}
}

func TestWriteInstructionsAndWritePromptFormatting(t *testing.T) {
	listed := []*tools.ToolInfo{
		{Name: "t1", Instructions: []string{"First.", "Second."}},
		{Name: "t2", Instructions: []string{"Second.", "Third."}},
	}
	instructions := writeInstructions(listed)
	if instructions != "First. Second. Third." {
		t.Fatalf("writeInstructions = %q", instructions)
	}

	prompt := writePrompt(instructions, &contract.TaskRequest{
		Prompt: "Question?",
		ContextNotes: []*contract.ContextNote{
			{Path: "Garden/Beds.md", Title: "Beds", Text: "Oak."},
			{Path: "Garden/Soil.md", Title: "", Text: "Clay."},
		},
	})
	want := "First. Second. Third.\n\nThe note Garden/Beds.md (Beds):\n\nOak.\n\nThe note Garden/Soil.md:\n\nClay.\n\nQuestion?"
	if prompt != want {
		t.Errorf("writePrompt = %q, want %q", prompt, want)
	}

	if got := joinParagraphs("  ", "one", "\n\t", "two", ""); got != "one\n\ntwo" {
		t.Errorf("joinParagraphs = %q, want one\\n\\ntwo", got)
	}
}

type noopFinder struct{}

func (noopFinder) Find(string) string { return "/opt/codex" }

type noopCommands struct{}

func (noopCommands) MakeToolsCommand(spec Spec) Command { return Command{Path: spec.Program} }
func (noopCommands) MakeAnswerCommand(spec Spec) (Command, error) {
	return Command{Path: spec.Program}, nil
}

type noopLauncher struct{}

func (noopLauncher) Start(context.Context, Command) (Process, error) { return nil, nil }

type noopServer struct{}

func (noopServer) Serve(context.Context, []*tools.ToolInfo, Invoke) (ServedTools, error) {
	return nil, nil
}

func TestRunnerBeginCloseAndDirLifecycle(t *testing.T) {
	validDeps := Dependencies{
		Settings: &settings.Settings{}, Finder: noopFinder{}, Commands: noopCommands{},
		Launcher: noopLauncher{}, ToolServer: noopServer{},
	}
	for name, mutate := range map[string]func(*Dependencies){
		"no settings":    func(d *Dependencies) { d.Settings = nil },
		"no finder":      func(d *Dependencies) { d.Finder = nil },
		"no commands":    func(d *Dependencies) { d.Commands = nil },
		"no launcher":    func(d *Dependencies) { d.Launcher = nil },
		"no tool server": func(d *Dependencies) { d.ToolServer = nil },
	} {
		deps := validDeps
		mutate(&deps)
		if _, err := New(deps); err == nil {
			t.Errorf("%s: New accepted missing dependency", name)
		}
	}

	r, err := New(validDeps)
	if err != nil || r.nextRun != 0 {
		t.Fatalf("New = (%v, %v), nextRun = %d", r, err, r.nextRun)
	}

	ctx1, end1, err := r.begin(t.Context())
	if err != nil || r.nextRun != 1 || len(r.cancels) != 1 {
		t.Fatalf("begin 1: err=%v nextRun=%d cancels=%d", err, r.nextRun, len(r.cancels))
	}
	end1()
	if ctx1.Err() == nil || len(r.cancels) != 0 {
		t.Errorf("after end1: ctxErr=%v cancels=%d", ctx1.Err(), len(r.cancels))
	}

	// getDir creates and reuses one temp directory, and Close removes it.
	dir1, err := r.getDir()
	if err != nil || dir1 == "" {
		t.Fatalf("getDir 1 = %q, %v", dir1, err)
	}
	dir2, err := r.getDir()
	if err != nil || dir2 != dir1 {
		t.Fatalf("getDir 2 = %q, %v, want %q", dir2, err, dir1)
	}

	// Start two concurrent runs and verify Close cancels both and waits for them.
	runA, endA, err := r.begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	runB, endB, err := r.begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-runA.Done()
		endA()
	}()
	go func() {
		<-runB.Done()
		endB()
	}()

	cleanSentinel := errors.New("clean failed")
	r.cleanErrs = cleanSentinel
	if err := r.Close(); !errors.Is(err, cleanSentinel) {
		t.Errorf("first Close = %v, want %v", err, cleanSentinel)
	}
	if _, err := os.Stat(dir1); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dir %q still exists after Close: %v", dir1, err)
	}
	// Second Close returns nil immediately.
	if err := r.Close(); err != nil {
		t.Errorf("second Close = %v, want nil", err)
	}
	if _, _, err := r.begin(t.Context()); !errors.Is(err, errClosed) {
		t.Errorf("begin after Close = %v, want errClosed", err)
	}

	// Close reports an error when os.RemoveAll fails (e.g. "." cannot be removed).
	r2, err := New(validDeps)
	if err != nil {
		t.Fatal(err)
	}
	r2.dir = "."
	if err := r2.Close(); err == nil {
		t.Error("Close with dir='.' succeeded, want error")
	}

	// getDir reports an error when TMPDIR does not exist.
	r3, err := New(validDeps)
	if err != nil {
		t.Fatal(err)
	}
	for _, env := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(env, "/dev/null/nonexistent")
	}
	if _, err := r3.getDir(); err == nil {
		t.Error("getDir with invalid TMPDIR succeeded, want error")
	}

	// readFailure preserves TryNext and Stop errors and wraps others into StopError.
	tryNext := connection.NewTryNextError(connection.ReasonUnavailable, 0)
	if got := readFailure(t.Context(), tryNext); !errors.Is(got, tryNext) {
		t.Errorf("readFailure(tryNext) = %v", got)
	}
	if got := readFailure(t.Context(), errors.New("other")); !connection.IsStop(got) {
		t.Errorf("readFailure(other) = %v, want StopError", got)
	}
}
