// Package command lets a plugin run the commands its manifest declares. A Handler runs them; Mount
// serves one to the host as PluginCommandService.
//
// The host checks every call against the manifest before it reaches a Handler: the command is
// declared, every required argument is given, every argument is declared and holds a value of its
// type, and a command over files is given a file of a declared extension.
package command

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

var (
	// ErrUnknownCommand is a command the handler does not run.
	ErrUnknownCommand = errors.New("no command of that id")
	// ErrBadArguments is a call whose arguments the command cannot run with. A handler wraps it with
	// what is wrong.
	ErrBadArguments = errors.New("the arguments do not fit the command")
	// ErrDuplicate is a second command added under an id.
	ErrDuplicate = errors.New("a command of that id is already added")
)

// ArgumentVersion is the argument the host gives a command whose manifest declares a condition on
// the text versions of a file: the id of the version that meets it.
const ArgumentVersion = "version-id"

// Arguments are the values of a call's arguments by their declared names, each as text: a number in
// decimal, a boolean as true or false. An optional argument left out is absent.
type Arguments map[string]string

// GetString returns the argument, empty where it was not given.
func (a Arguments) GetString(name string) string {
	return a[name]
}

// Request is one run of a command.
type Request struct {
	// Command is the id the manifest declares the command under.
	Command string
	// Path is the file the command runs over, relative to the vault folder. Empty for a command that
	// declares no file extensions.
	Path      string
	Arguments Arguments
}

// Result is what a command that ran answers.
type Result struct {
	// Message is what a person reads about the outcome.
	Message string
	// OpenedPath is the file a window opens once the command ran, relative to the vault folder. Empty
	// for a command that leaves nothing to open.
	OpenedPath string
}

// Handler runs the commands a plugin declares. On failure it returns an error whose text a person
// reads: ErrUnknownCommand for a command it does not run and ErrBadArguments for arguments it cannot
// run with.
type Handler interface {
	RunCommand(ctx context.Context, request Request) (Result, error)
}

// Run is a function that runs one command.
type Run func(ctx context.Context, request Request) (Result, error)

// Registry is a Handler of the commands added to it by id.
type Registry struct {
	mu   sync.RWMutex
	runs map[string]Run
}

// New makes a registry of no commands.
func New() *Registry {
	return &Registry{runs: map[string]Run{}}
}

// Add adds the command run answers for.
func (r *Registry) Add(id string, run Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, isAdded := r.runs[id]; isAdded {
		return fmt.Errorf("%w: %s", ErrDuplicate, id)
	}
	r.runs[id] = run
	return nil
}

// RunCommand runs the command added under the request's id.
func (r *Registry) RunCommand(ctx context.Context, request Request) (Result, error) {
	r.mu.RLock()
	run, isAdded := r.runs[request.Command]
	r.mu.RUnlock()
	if !isAdded {
		return Result{}, fmt.Errorf("%w: %s", ErrUnknownCommand, request.Command)
	}
	return run(ctx, request)
}

// Mount registers the handler as PluginCommandService on s.
func Mount(s *server.Server, handler Handler) {
	s.Register(numenv1connect.NewPluginCommandServiceHandler(Service{Handler: handler}, s.HandlerOptions()...))
}

// Service answers PluginCommandService with a Handler.
type Service struct {
	numenv1connect.UnimplementedPluginCommandServiceHandler

	Handler Handler
}

// RunCommand runs the command and answers what the handler said. A failure is an error code with the
// handler's words as the message.
func (s Service) RunCommand(
	ctx context.Context, req *connect.Request[numenv1.RunCommandRequest],
) (*connect.Response[numenv1.RunCommandResponse], error) {
	result, err := s.Handler.RunCommand(ctx, Request{
		Command:   req.Msg.GetCommand(),
		Path:      req.Msg.GetPath(),
		Arguments: Arguments(req.Msg.GetArguments()),
	})
	if err == nil {
		return connect.NewResponse(&numenv1.RunCommandResponse{
			Message: result.Message, OpenedPath: result.OpenedPath,
		}), nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	code := numenv1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED
	switch {
	case errors.Is(err, ErrUnknownCommand):
		code = numenv1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND
	case errors.Is(err, ErrBadArguments):
		code = numenv1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS
	}
	return connect.NewResponse(&numenv1.RunCommandResponse{Error: code, Message: err.Error()}), nil
}
