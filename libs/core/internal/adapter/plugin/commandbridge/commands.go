// Package commandbridge runs the commands the manifests declare in the backends of their plugins,
// over PluginCommandService.
package commandbridge

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// ErrUnknownCommand is a call the plugin answered as for a command it does not run.
var ErrUnknownCommand = errors.New("the plugin runs no command of that id")

// ErrBadArguments is a call the plugin answered as not meeting the command.
var ErrBadArguments = errors.New("the plugin does not take those arguments")

// Host provides the plugins, their manifests and their activation.
type Host interface {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
	pluginport.PluginActivator
}

// Commands runs the commands of the plugins the host runs.
type Commands struct {
	host Host
}

// Open makes the bridge to the plugins the host runs. It starts none.
func Open(host Host) Commands {
	return Commands{host: host}
}

// RunCommand starts the plugin where it is not running and runs the command in its backend, holding
// its reference for the length of the call.
func (c Commands) RunCommand(
	ctx context.Context, call pluginport.PluginCommandCall,
) (result pluginport.PluginCommandResult, err error) {
	client, lease, err := bridge.Open(ctx, c.host, call.PluginID, numenv1connect.NewPluginCommandServiceClient)
	if err != nil {
		return pluginport.PluginCommandResult{}, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.RunCommand(ctx, connect.NewRequest(&v1.RunCommandRequest{
		Command: call.Command, Path: call.Path, Arguments: call.Arguments, ItemId: call.ItemID,
	}))
	if err != nil {
		return pluginport.PluginCommandResult{}, readCommandError(ctx, call.PluginID, err)
	}
	message := res.Msg.GetMessage()
	switch res.Msg.GetError() {
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED:
		return pluginport.PluginCommandResult{Message: message, OpenedPath: res.Msg.GetOpenedPath()}, nil
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND:
		return pluginport.PluginCommandResult{}, fmt.Errorf("%w: %s", ErrUnknownCommand, call.Command)
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS:
		return pluginport.PluginCommandResult{}, fmt.Errorf("%w: %s", ErrBadArguments, message)
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE:
		return pluginport.PluginCommandResult{}, &pluginport.UnavailableError{PluginID: call.PluginID, Reason: pluginport.PluginFailed, Err: errors.New(message)}
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED:
	}
	return pluginport.PluginCommandResult{}, &pluginport.CommandFailedError{Message: message}
}

func readCommandError(ctx context.Context, pluginID string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var failure *connect.Error
	if errors.As(err, &failure) && failure.Code() != connect.CodeUnavailable && failure.Code() != connect.CodeUnimplemented {
		return fmt.Errorf("command rpc: %w", err)
	}
	return &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginFailed, Err: err}
}

// NoCommands is an installation with no plugin to run a command.
type NoCommands struct{}

func (NoCommands) RunCommand(_ context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error) {
	return pluginport.PluginCommandResult{}, &pluginport.UnavailableError{PluginID: call.PluginID, Reason: pluginport.PluginNotInstalled}
}
