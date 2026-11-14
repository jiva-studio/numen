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

// Finder finds the Codex program on the machine.
type Finder interface {
	// Find returns the program: the custom path when it is runnable, else the program found on the
	// machine, else none.
	Find(customPath string) string
}

// Spec is what a run of the program is built from. ToolsURL is the server of the tools a run that may
// call them is given. Thread is the thread of the program the run continues, empty for a new one; a run
// that IsKept keeps its thread for a later run to continue.
type Spec struct {
	Program   string
	Prompt    string
	Model     string
	Reasoning string
	ToolsURL  string
	Thread    string
	IsKept    bool
}

// Command is the program and environment of a run, and what the program reads on its standard input.
// Release frees what the command was built with.
type Command struct {
	Path    string
	Args    []string
	Dir     string
	Env     []string
	Token   string
	Stdin   string
	Release func() error
}

// Commands builds the command lines of the runs of the program.
type Commands interface {
	// MakeToolsCommand builds the command of a run that may call the tools of the server of the spec.
	MakeToolsCommand(spec Spec) Command
	// MakeAnswerCommand builds the command of a run that only answers.
	MakeAnswerCommand(spec Spec) (Command, error)
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

// TempDirs creates the folder the runs start in and removes it.
type TempDirs interface {
	Create() (string, error)
	Remove(dir string) error
}
