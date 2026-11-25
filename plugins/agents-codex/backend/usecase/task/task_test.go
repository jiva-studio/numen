package task_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/tempdir"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

func newTempDirs() tempdir.Folders { return tempdir.Folders{Pattern: "numen-codex-tasks-*"} }

type stubFinder struct{ path string }

func (f stubFinder) Find(string) string { return f.path }

type recordingCommands struct {
	mu        sync.Mutex
	specs     []task.Spec
	answerErr error
	onRelease func() error
}

func (c *recordingCommands) MakeToolsCommand(spec task.Spec) task.Command {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.specs = append(c.specs, spec)
	return task.Command{Path: spec.Program, Stdin: spec.Prompt, Release: c.onRelease}
}

func (c *recordingCommands) MakeAnswerCommand(spec task.Spec) (task.Command, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.specs = append(c.specs, spec)
	if c.answerErr != nil {
		return task.Command{}, c.answerErr
	}
	return task.Command{Path: spec.Program, Stdin: spec.Prompt, Release: c.onRelease}, nil
}

type scriptedLauncher struct {
	mu       sync.Mutex
	commands []task.Command
	runs     []func(context.Context, task.Command) (task.Process, error)
}

func (l *scriptedLauncher) Start(ctx context.Context, cmd task.Command) (task.Process, error) {
	l.mu.Lock()
	l.commands = append(l.commands, cmd)
	idx := len(l.commands) - 1
	var fn func(context.Context, task.Command) (task.Process, error)
	if idx < len(l.runs) {
		fn = l.runs[idx]
	} else if len(l.runs) > 0 {
		fn = l.runs[len(l.runs)-1]
	}
	l.mu.Unlock()
	if fn != nil {
		return fn(ctx, cmd)
	}
	return &stubProcess{stdout: `{"type":"item.completed","item":{"type":"agent_message","text":"Raised beds."}}` + "\n"}, nil
}

type stubProcess struct {
	stdout  string
	stderr  string
	waitErr error
	blockOn <-chan struct{}
}

func (p *stubProcess) Output() io.ReadCloser {
	if p.blockOn != nil {
		return &blockingReader{done: p.blockOn}
	}
	return io.NopCloser(strings.NewReader(p.stdout))
}

func (p *stubProcess) Stderr() string { return p.stderr }
func (p *stubProcess) Wait() error    { return p.waitErr }

type blockingReader struct{ done <-chan struct{} }

func (b *blockingReader) Read([]byte) (int, error) {
	<-b.done
	return 0, io.EOF
}

func (b *blockingReader) Close() error { return nil }

type gardenTools struct {
	listed  []*tools.ToolInfo
	listErr error
}

func (g gardenTools) ListTools(context.Context) (tools.ToolList, error) {
	if g.listErr != nil {
		return tools.ToolList{}, g.listErr
	}
	if g.listed != nil {
		return tools.ToolList{Tools: g.listed}, nil
	}
	return tools.ToolList{Tools: []*tools.ToolInfo{{Name: "note_read", Title: "Read a note", Effect: tools.ReadOnly}}}, nil
}

func (gardenTools) InvokeTool(context.Context, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{}, nil
}

// stuckServer serves the tools at an address and fails to stop serving them.
type stuckServer struct {
	serveErr error
	closeErr error
}

func (s stuckServer) Serve(context.Context, []*tools.ToolInfo, task.Invoke) (task.ServedTools, error) {
	if s.serveErr != nil {
		return nil, s.serveErr
	}
	return stuckTools{err: s.closeErr}, nil
}

type stuckTools struct{ err error }

func (stuckTools) GetURL() string                { return "http://127.0.0.1:4100/mcp" }
func (stuckTools) GetToken() string              { return "orchard-token" }
func (s stuckTools) Close(context.Context) error { return s.err }

// A turn that the program answered ends with its answer though the tools could not stop being served; the
// failure is returned when the plugin closes.
func TestAFailureToStopServingTheToolsKeepsTheAnswerOfTheTurn(t *testing.T) {
	stuck := errors.New("the tool server did not stop")
	cmds := &recordingCommands{}
	launcher := &scriptedLauncher{}
	runner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"}, Commands: cmds, Launcher: launcher,
		Tools: gardenTools{}, ToolServer: stuckServer{closeErr: stuck}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var texts []string

	done, err := runner.RunTask(t.Context(), connection.Call{ConnectionID: "conn-orchard"}, &contract.TaskRequest{Prompt: "How are the beds?"},
		func(frame *contract.TaskFrame) error {
			texts = append(texts, frame.GetAgentMessageChunk().GetText())
			return nil
		})

	if err != nil || done.GetStopReason() != contract.StopReason_STOP_REASON_END_TURN || len(texts) != 1 || texts[0] != "Raised beds." {
		t.Errorf("RunTask = %v, %v after %q, want the end of the turn with its answer", done, err, texts)
	}
	if len(cmds.specs) != 1 || cmds.specs[0].ToolsURL != "http://127.0.0.1:4100/mcp" {
		t.Errorf("tools spec = %+v", cmds.specs)
	}
	if len(launcher.commands) != 1 || launcher.commands[0].Token != "orchard-token" || launcher.commands[0].Dir == "" || launcher.commands[0].Path != "/opt/codex" {
		t.Errorf("launched command = %+v", launcher.commands)
	}
	if err := runner.Close(); !errors.Is(err, stuck) {
		t.Errorf("Close = %v, want the failure to stop serving the tools", err)
	}
}

func TestRunTaskThreadsRetriesAndErrorPaths(t *testing.T) {
	// 1. Missing program returns TryNextError and DetectConnections returns empty.
	missingRunner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: ""}, Commands: &recordingCommands{},
		Launcher: &scriptedLauncher{}, ToolServer: stuckServer{}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if found, err := missingRunner.DetectConnections(t.Context(), connection.Call{}); err != nil || len(found) != 0 {
		t.Errorf("DetectConnections missing = (%v, %v)", found, err)
	}
	if _, err := missingRunner.RunTask(t.Context(), connection.Call{}, &contract.TaskRequest{Prompt: "hi"}, func(*contract.TaskFrame) error { return nil }); err == nil {
		t.Error("RunTask with missing program succeeded")
	} else if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("RunTask missing program = %v, want TryNextError", err)
	}

	// 2. ListTools error and Serve error return StopError.
	listFailRunner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"}, Commands: &recordingCommands{},
		Launcher: &scriptedLauncher{}, Tools: gardenTools{listErr: errors.New("list failed")}, ToolServer: stuckServer{}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := listFailRunner.RunTask(t.Context(), connection.Call{}, &contract.TaskRequest{Prompt: "hi"}, func(*contract.TaskFrame) error { return nil }); !connection.IsStop(err) {
		t.Errorf("RunTask listTools error = %v, want StopError", err)
	}

	serveFailRunner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"}, Commands: &recordingCommands{},
		Launcher: &scriptedLauncher{}, Tools: gardenTools{}, ToolServer: stuckServer{serveErr: errors.New("serve failed")}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serveFailRunner.RunTask(t.Context(), connection.Call{}, &contract.TaskRequest{Prompt: "hi"}, func(*contract.TaskFrame) error { return nil }); !connection.IsStop(err) {
		t.Errorf("RunTask serveTools error = %v, want StopError", err)
	}

	// 3. MakeAnswerCommand error when Tools is nil returns StopError.
	cmdFailRunner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"},
		Commands:   &recordingCommands{answerErr: errors.New("mkcmd failed")},
		Launcher:   &scriptedLauncher{},
		ToolServer: stuckServer{},
		TempDirs:   newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cmdFailRunner.RunTask(t.Context(), connection.Call{}, &contract.TaskRequest{Prompt: "hi"}, func(*contract.TaskFrame) error { return nil }); !connection.IsStop(err) {
		t.Errorf("RunTask makeTaskCommand error = %v, want StopError", err)
	}

	// 4. Thread continuation, retry on gone thread, and no retry when no thread was resumed.
	cmds := &recordingCommands{}
	launcher := &scriptedLauncher{
		runs: []func(context.Context, task.Command) (task.Process, error){
			// Run 0: first turn of conv-1 starts thread-1 (with benign stderr warning while reports > 0).
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{
					stdout: `{"type":"thread.started","thread_id":"thread-1"}` + "\n" +
						`{"type":"item.completed","item":{"type":"agent_message","text":"First."}}` + "\n",
					stderr: "warning: benign stderr with report\n",
				}, nil
			},
			// Run 1: second turn of conv-1 fails before any output (thread gone).
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{stderr: "error: thread gone\n", waitErr: errors.New("exit 1")}, nil
			},
			// Run 2: retry of second turn of conv-1 with empty thread succeeds.
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{
					stdout: `{"type":"thread.started","thread_id":"thread-2"}` + "\n" +
						`{"type":"item.completed","item":{"type":"agent_message","text":"Second."}}` + "\n",
				}, nil
			},
			// Run 3: turn without conversation that fails with stderr is NOT retried and reports reason as frame.
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{stderr: "error: no output failure\n"}, nil
			},
		},
	}
	runner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"}, Commands: cmds,
		Launcher: launcher, Tools: gardenTools{listed: []*tools.ToolInfo{}}, ToolServer: stuckServer{}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	call := connection.Call{ConnectionID: "conn-1"}
	if _, err := runner.RunTask(t.Context(), call, &contract.TaskRequest{Prompt: "p1", Conversation: "conv-1"}, func(*contract.TaskFrame) error { return nil }); err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	if _, err := runner.RunTask(t.Context(), call, &contract.TaskRequest{Prompt: "p2", Conversation: "conv-1"}, func(*contract.TaskFrame) error { return nil }); err != nil {
		t.Fatalf("turn 2 (retry): %v", err)
	}
	var failureFrames []string
	if _, err := runner.RunTask(t.Context(), call, &contract.TaskRequest{Prompt: "p3"}, func(f *contract.TaskFrame) error {
		failureFrames = append(failureFrames, f.GetAgentMessageChunk().GetText())
		return nil
	}); !connection.IsStop(err) || !slices.Equal(failureFrames, []string{"no output failure"}) {
		t.Fatalf("turn 3 (no retry): frames=%q err=%v", failureFrames, err)
	}
	if len(cmds.specs) != 4 ||
		cmds.specs[0].Thread != "" || !cmds.specs[0].IsKept ||
		cmds.specs[1].Thread != "thread-1" || !cmds.specs[1].IsKept ||
		cmds.specs[2].Thread != "" || !cmds.specs[2].IsKept ||
		cmds.specs[3].Thread != "" || cmds.specs[3].IsKept {
		t.Errorf("recorded specs = %+v", cmds.specs)
	}

	// 5. Sink error.
	sinkErr := errors.New("chat closed")
	if _, err := runner.RunTask(t.Context(), call, &contract.TaskRequest{Prompt: "p"}, func(*contract.TaskFrame) error {
		return sinkErr
	}); !errors.Is(err, sinkErr) {
		t.Errorf("RunTask sink error = %v, want %v", err, sinkErr)
	}
}

func TestAnswerAndRunProgramBranches(t *testing.T) {
	// Missing program in Answer returns error.
	missingRunner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: ""}, Commands: &recordingCommands{},
		Launcher: &scriptedLauncher{}, ToolServer: stuckServer{}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := missingRunner.Answer(t.Context(), connection.Call{}, "", "hi"); err == nil {
		t.Error("Answer with missing program succeeded")
	}

	released := 0
	cmds := &recordingCommands{
		onRelease: func() error {
			released++
			return nil
		},
	}
	blockCh := make(chan struct{})
	launcher := &scriptedLauncher{
		runs: []func(context.Context, task.Command) (task.Process, error){
			// 0: Normal answer.
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{stdout: "  direct answer \n", stderr: "   "}, nil
			},
			// 1: Empty output returns errEmptyResponse.
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{stdout: "   \n"}, nil
			},
			// 2: Wait error returns failure reason.
			func(context.Context, task.Command) (task.Process, error) {
				return &stubProcess{stderr: "error: rate limited\n", waitErr: errors.New("exit 1")}, nil
			},
			// 3: Launcher.Start error returns TryNextError.
			func(context.Context, task.Command) (task.Process, error) {
				return nil, errors.New("spawn failed")
			},
			// 4: Blocked process canceled when runner.Close() is called.
			func(ctx context.Context, _ task.Command) (task.Process, error) {
				close(blockCh)
				return &stubProcess{blockOn: ctx.Done()}, nil
			},
		},
	}
	runner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: stubFinder{path: "/opt/codex"}, Commands: cmds,
		Launcher: launcher, ToolServer: stuckServer{}, TempDirs: newTempDirs(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, err := runner.Answer(t.Context(), connection.Call{}, "inst", "prompt"); err != nil || got != "direct answer\n" {
		t.Errorf("Answer 0 = (%q, %v), want direct answer", got, err)
	}
	if _, err := runner.Answer(t.Context(), connection.Call{}, "", "prompt"); err == nil {
		t.Error("Answer 1 (empty) succeeded, want error")
	}
	if _, err := runner.Answer(t.Context(), connection.Call{}, "", "prompt"); err == nil || err.Error() != "rate limited" {
		t.Errorf("Answer 2 (waitErr) = %v, want rate limited", err)
	}
	if _, err := runner.Answer(t.Context(), connection.Call{}, "", "prompt"); err == nil {
		t.Error("Answer 3 (startErr) succeeded, want error")
	}
	if released != 4 {
		t.Errorf("released = %d, want 4", released)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := runner.Answer(t.Context(), connection.Call{}, "", "prompt")
		errCh <- err
	}()
	<-blockCh
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err == nil {
		t.Error("Answer 4 interrupted by Close succeeded, want error")
	}
	if _, err := runner.Answer(t.Context(), connection.Call{}, "", "prompt"); err == nil {
		t.Error("Answer after Close succeeded, want error")
	}
	if _, err := runner.RunTask(t.Context(), connection.Call{}, &contract.TaskRequest{Prompt: "hi"}, func(*contract.TaskFrame) error { return nil }); !connection.IsStop(err) {
		t.Errorf("RunTask after Close = %v, want StopError", err)
	}
}
