// Package ports declares what the use cases of the plugin need from the machine.
package ports

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

// ProgramFinder finds the Claude Code program on the machine.
type ProgramFinder interface {
	// Find returns the command of the program, or none when it is not installed. A runnable custom
	// path takes precedence over the places the program is looked for.
	Find(customPath string) string
}

// ModelLister asks the program for the models it accepts.
type ModelLister interface {
	ListModels(ctx context.Context, command, env []string) ([]model.Model, error)
}

// Environment gives the variables a program is started with.
type Environment interface {
	GetVariables() []string
}

// ToolConfig keeps the file that names the vault's tool server and the person's own MCP servers for a
// run.
type ToolConfig interface {
	// Write stores the file readable by this user alone and returns its path and the names of the
	// person's servers in it, or no path when the address or the token is empty.
	Write(url, token string) (string, []string, error)
	// Remove deletes the file; a path of none or of a file that is gone is not an error.
	Remove(path string) error
}

// Launch is the program to start for a run.
type Launch struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
	Input   string
}

// Process is a program that is running.
type Process interface {
	// Output is the standard output of the program.
	Output() io.ReadCloser
	// Stderr is what the program wrote to its standard error; complete once Wait has returned.
	Stderr() string
	// Wait returns when the program has exited.
	Wait() error
}

// Launcher starts programs; a canceled context ends the program and everything it started.
type Launcher interface {
	Launch(ctx context.Context, launch Launch) (Process, error)
}
