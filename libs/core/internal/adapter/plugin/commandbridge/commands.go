// Package commandbridge connects the commands the manifests declare to the backends that run them,
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

// Commands is the commands of the plugins the host runs.
type Commands struct {
	host Host
}

// Open makes the bridge to the plugins the host runs. It starts none.
func Open(host Host) Commands {
	return Commands{host: host}
}

// ListCommands lists the commands the manifests of the enabled plugins declare. No plugin is started.
func (c Commands) ListCommands() []pluginport.PluginCommandEntry {
	var found []pluginport.PluginCommandEntry
	for _, info := range c.host.GetPluginInfos() {
		plugin, isEnabled := c.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, command := range plugin.Manifest.Contributes.Commands {
			found = append(found, pluginport.PluginCommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, Command: command,
			})
		}
	}
	return found
}

// ListSlots lists the slot entries the manifests of the enabled plugins declare. No plugin is started.
func (c Commands) ListSlots() []pluginport.PluginSlotEntry {
	var found []pluginport.PluginSlotEntry
	for _, info := range c.host.GetPluginInfos() {
		plugin, isEnabled := c.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, entry := range plugin.Manifest.Contributes.Slots {
			found = append(found, pluginport.PluginSlotEntry{PluginID: plugin.Manifest.ID, Entry: entry})
		}
	}
	return found
}

// ListHandlers lists the handlers the manifests of the enabled plugins declare, each as a command under
// the full id of its shared command. No plugin is started.
func (c Commands) ListHandlers() []pluginport.PluginCommandEntry {
	var found []pluginport.PluginCommandEntry
	for _, info := range c.host.GetPluginInfos() {
		plugin, isEnabled := c.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, handler := range plugin.Manifest.Contributes.Handlers {
			found = append(found, pluginport.PluginCommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name,
				Command: pluginport.PluginCommand{ID: handler.Command, Title: handler.Title, When: handler.When},
			})
		}
	}
	return found
}

// ListTriggers lists the triggers the manifests of the enabled plugins declare, leaving out those of a
// plugin the host holds suspended. No plugin is started.
func (c Commands) ListTriggers() []pluginport.PluginTriggerEntry {
	var found []pluginport.PluginTriggerEntry
	for _, info := range c.host.GetPluginInfos() {
		plugin, isEnabled := c.host.GetPlugin(info.ID)
		if !isEnabled || info.Failure != nil {
			continue
		}
		for _, trigger := range plugin.Manifest.Contributes.Triggers {
			found = append(found, pluginport.PluginTriggerEntry{PluginID: plugin.Manifest.ID, Trigger: trigger})
		}
	}
	return found
}

// GetParts says what the enabled plugin of the id has, and whether the host holds it suspended.
func (c Commands) GetParts(pluginID string) pluginport.PluginParts {
	plugin, isEnabled := c.host.GetPlugin(pluginID)
	if !isEnabled {
		return pluginport.PluginParts{}
	}
	parts := pluginport.PluginParts{HasBackend: plugin.Manifest.HasBackend(), HasFrontend: plugin.Manifest.HasFrontend()}
	for _, info := range c.host.GetPluginInfos() {
		if info.ID == pluginID {
			parts.IsSuspended = info.Failure != nil
		}
	}
	return parts
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
		Command: call.Command, Path: call.Path, Arguments: call.Arguments,
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

// readCommandError sorts the failure of a call: the end of the caller's context is the caller's own,
// a plugin that serves no commands or cannot be reached is unavailable, and anything else is returned
// as it is.
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

// NoCommands is an installation with no plugin that declares a command.
type NoCommands struct{}

func (NoCommands) ListCommands() []pluginport.PluginCommandEntry { return nil }

func (NoCommands) ListSlots() []pluginport.PluginSlotEntry { return nil }

func (NoCommands) ListHandlers() []pluginport.PluginCommandEntry { return nil }

func (NoCommands) GetParts(string) pluginport.PluginParts { return pluginport.PluginParts{} }

func (NoCommands) RunCommand(_ context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error) {
	return pluginport.PluginCommandResult{}, &pluginport.UnavailableError{PluginID: call.PluginID, Reason: pluginport.PluginNotInstalled}
}
