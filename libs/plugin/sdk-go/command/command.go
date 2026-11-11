// Package command lets a plugin run the commands its manifest declares and its handlers of the shared
// commands. A Handler runs them; Mount serves one to the host as PluginCommandService.
//
// The host checks every call against the manifest before it is passed to a Handler: the command is
// declared, its when condition holds over the file, every required argument is given, and every
// argument is declared and holds a value of its type.
package command

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
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
//
//numen:api
type Request struct {
	// Command is the id the manifest declares the command under.
	Command string
	// Path is the file the command runs over, relative to the vault folder. Empty for a run with no
	// target, such as one from the palette.
	Path      string
	Arguments Arguments
	// ItemID is the id of the item a person picked, for a command that offers its own items. Empty for
	// a run without a pick, such as one from numen-cli or a trigger.
	ItemID string
}

// Result is what a command that ran answers.
//
//numen:api
type Result struct {
	// Message is the outcome text shown to the user.
	Message string
	// OpenedPath is the file a window opens once the command ran, relative to the vault folder. Empty
	// for a command that leaves nothing to open.
	OpenedPath string
}

// Handler runs the commands a plugin declares. On failure it returns an error
// whose text is shown to the user: ErrUnknownCommand for a command it does not
// run and ErrBadArguments for arguments it cannot run with.
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

func New() *Registry {
	return &Registry{runs: map[string]Run{}}
}

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
	s.Register(numenv1connect.NewPluginCommandServiceHandler(service{handler: handler}, s.HandlerOptions()...))
}

// service answers PluginCommandService with a Handler.
type service struct {
	numenv1connect.UnimplementedPluginCommandServiceHandler

	handler Handler
}

// RunCommand runs the command and returns the handler's result. A failure is
// returned as an error code with the handler's message.
func (s service) RunCommand(
	ctx context.Context, req *connect.Request[v1.RunCommandRequest],
) (*connect.Response[v1.RunCommandResponse], error) {
	result, err := s.handler.RunCommand(ctx, readRequest(req.Msg))
	if err == nil {
		return connect.NewResponse(writeResult(result)), nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	return connect.NewResponse(writeFailure(err)), nil
}

func readRequest(msg *v1.RunCommandRequest) Request {
	return Request{
		Command:   msg.GetCommand(),
		Path:      msg.GetPath(),
		Arguments: Arguments(msg.GetArguments()),
		ItemID:    msg.GetItemId(),
	}
}

func writeResult(result Result) *v1.RunCommandResponse {
	return &v1.RunCommandResponse{Message: result.Message, OpenedPath: result.OpenedPath}
}

func writeFailure(err error) *v1.RunCommandResponse {
	code := v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED
	switch {
	case errors.Is(err, ErrUnknownCommand):
		code = v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND
	case errors.Is(err, ErrBadArguments):
		code = v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS
	}
	return &v1.RunCommandResponse{Error: code, Message: err.Error()}
}
