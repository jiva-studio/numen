package task_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/tempdir"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

type foundProgram struct{}

func (foundProgram) Find(string) string { return "/opt/codex" }

type plainCommands struct{}

func (plainCommands) MakeToolsCommand(spec task.Spec) task.Command {
	return task.Command{Path: spec.Program, Stdin: spec.Prompt}
}

func (plainCommands) MakeAnswerCommand(spec task.Spec) (task.Command, error) {
	return task.Command{Path: spec.Program, Stdin: spec.Prompt}, nil
}

// answeringLauncher starts a program that answers "Raised beds." and exits.
type answeringLauncher struct{}

func (answeringLauncher) Start(context.Context, task.Command) (task.Process, error) {
	return answeringProcess{}, nil
}

type answeringProcess struct{}

func (answeringProcess) Output() io.ReadCloser {
	return io.NopCloser(strings.NewReader(`{"type":"item.completed","item":{"type":"agent_message","text":"Raised beds."}}` + "\n"))
}

func (answeringProcess) Stderr() string { return "" }

func (answeringProcess) Wait() error { return nil }

type gardenTools struct{}

func (gardenTools) ListTools(context.Context) (tools.Listing, error) {
	return tools.Listing{Tools: []*tools.ToolInfo{{Name: "note_read", Title: "Read a note", Effect: tools.ReadOnly}}}, nil
}

func (gardenTools) InvokeTool(context.Context, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{}, nil
}

// stuckServer serves the tools at an address and fails to stop serving them.
type stuckServer struct{ err error }

func (s stuckServer) Serve(context.Context, []*tools.ToolInfo, task.Invoke) (task.ServedTools, error) {
	return stuckTools(s), nil
}

type stuckTools struct{ err error }

func (stuckTools) GetURL() string                { return "http://127.0.0.1:4100/mcp" }
func (stuckTools) GetToken() string              { return "orchard-token" }
func (s stuckTools) Close(context.Context) error { return s.err }

// A turn that the program answered ends with its answer though the tools could not stop being served; the
// failure is returned when the plugin closes.
func TestAFailureToStopServingTheToolsKeepsTheAnswerOfTheTurn(t *testing.T) {
	stuck := errors.New("the tool server did not stop")
	runner, err := task.New(task.Dependencies{
		Settings: &settings.Settings{}, Finder: foundProgram{}, Commands: plainCommands{}, Launcher: answeringLauncher{},
		Tools: gardenTools{}, ToolServer: stuckServer{err: stuck}, TempDirs: tempdir.Folders{Pattern: "numen-codex-tasks-*"},
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
	if err := runner.Close(); !errors.Is(err, stuck) {
		t.Errorf("Close = %v, want the failure to stop serving the tools", err)
	}
}
