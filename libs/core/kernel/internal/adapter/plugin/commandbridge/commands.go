// Package commandbridge runs the commands the manifests declare in the backends of their plugins,
// over PluginCommandService.
package commandbridge

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// ErrUnknownCommand is a call the plugin answered as for a command it does not run.
var ErrUnknownCommand = errors.New("the plugin runs no command of that id")

// ErrBadArguments is a call the plugin answered as not meeting the command.
var ErrBadArguments = errors.New("the plugin does not take those arguments")

// Host activates the plugin a command runs in.
type Host interface {
	bridge.Activator
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
	ctx context.Context, call plugin.CommandCall,
) (result plugin.CommandResult, err error) {
	client, lease, err := bridge.Open(ctx, c.host, call.PluginID, numenv1connect.NewPluginCommandServiceClient)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.RunCommand(ctx, connect.NewRequest(&v1.RunCommandRequest{
		Command: call.Command, Path: call.Path, Arguments: call.Arguments, ItemId: call.ItemID,
	}))
	if err != nil {
		return plugin.CommandResult{}, readCommandError(ctx, call.PluginID, err)
	}
	message := res.Msg.GetMessage()
	switch res.Msg.GetError() {
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED:
		return plugin.CommandResult{Message: message, OpenedPath: res.Msg.GetOpenedPath()}, nil
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND:
		return plugin.CommandResult{}, fmt.Errorf("%w: %s", ErrUnknownCommand, call.Command)
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS:
		return plugin.CommandResult{}, fmt.Errorf("%w: %s", ErrBadArguments, message)
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE:
		return plugin.CommandResult{}, &plugin.UnavailableError{PluginID: call.PluginID, Reason: plugin.Failed, Err: errors.New(message)}
	case v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED:
	}
	return plugin.CommandResult{}, &plugin.CommandFailedError{Message: message}
}

func readCommandError(ctx context.Context, pluginID string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var failure *connect.Error
	if errors.As(err, &failure) && failure.Code() != connect.CodeUnavailable && failure.Code() != connect.CodeUnimplemented {
		return fmt.Errorf("command rpc: %w", err)
	}
	return &plugin.UnavailableError{PluginID: pluginID, Reason: plugin.Failed, Err: err}
}
