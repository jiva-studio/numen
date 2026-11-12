package plugincommand

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// RunByName runs a command in the vault a person names.
type RunByName struct {
	find VaultFinding
}

// NewRunByName creates the use case that resolves the vault with find.
func NewRunByName(find VaultFinding) RunByName {
	return RunByName{find: find}
}

// ExecuteByName resolves the vault, opens the command runner over it, and runs the command.
func (u RunByName) ExecuteByName(
	ctx context.Context, open RunnerOpener, nameOrPath string, call plugin.CommandCall,
) (_ plugin.CommandResult, err error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	run, closeRun, err := open(ctx, v)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	defer func() {
		if closeRun != nil {
			err = errors.Join(err, closeRun())
		}
	}()
	return run.Execute(ctx, v, call)
}

// ExecuteSharedByName resolves the vault, opens the command runner over it, picks the handler of the shared
// command, and runs it.
func (u RunByName) ExecuteSharedByName(
	ctx context.Context, open RunnerOpener, nameOrPath string, handlers []plugin.CommandEntry,
	pluginID string, call plugin.CommandCall,
) (_ plugin.CommandResult, err error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	run, closeRun, err := open(ctx, v)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	defer func() {
		if closeRun != nil {
			err = errors.Join(err, closeRun())
		}
	}()
	handler, err := run.FindHandler(ctx, v, handlers, pluginID, call.Path)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	call.PluginID = handler.PluginID
	call.Command = handler.Command.ID
	return run.Execute(ctx, v, call)
}
