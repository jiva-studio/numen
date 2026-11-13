package task

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// Tools lists the tools of the host the plugin may offer and runs a call of one.
type Tools interface {
	ListTools(ctx context.Context) (tools.Listing, error)
	InvokeTool(ctx context.Context, call tools.Call) (*tools.CallResponse, error)
}

// Invoke runs one call of a served tool with the JSON arguments of the agent.
type Invoke func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error)

// ToolServer serves tools to an agent program.
type ToolServer interface {
	Serve(ctx context.Context, listed []*tools.ToolInfo, invoke Invoke) (ServedTools, error)
}

// ServedTools is where tools are served, the token presented there, and how the serving ends.
type ServedTools interface {
	GetURL() string
	GetToken() string
	Close(ctx context.Context) error
}

// Run is what one run of the program is started with. ToolsURL is empty for a run that only answers.
type Run struct {
	Prompt     string
	Model      string
	Session    string
	ToolsURL   string
	ToolsToken string
}

// Command is the program, the folder and the environment of a run, what the program reads on its
// standard input, and what releases the files made for it.
type Command struct {
	Path    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   string
	Release func() error
}

// Program is the Antigravity program on the machine.
type Program interface {
	// Find returns the program at path when it is runnable, else the program found on the machine, else
	// none.
	Find(path string) string
	MakeTaskCommand(path string, run Run) (Command, error)
	MakeAnswerCommand(path string, run Run) (Command, error)
}

// Process is a started program of a run.
type Process interface {
	Output() io.ReadCloser
	Stderr() string
	Wait() error
}

// Launcher starts a command and ends its process group when its context is canceled.
type Launcher interface {
	Start(ctx context.Context, cmd Command) (Process, error)
}

// Models chooses the listed model that a model and a reasoning level of a row name.
type Models interface {
	ResolveModel(ctx context.Context, program, model, level string) string
}
