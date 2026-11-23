package plugincommand

import (
	"context"
	"errors"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// RunByName runs a command in the vault a person names.
type RunByName struct {
	lookup vaultport.VaultLookup
}

// NewRunByName creates the use case that resolves the vault with find.
func NewRunByName(lookup vaultport.VaultLookup) RunByName {
	return RunByName{lookup: lookup}
}

// ExecuteByName resolves the vault, opens the command runner over it, and runs the command.
func (u RunByName) ExecuteByName(
	ctx context.Context, open RunnerOpener, nameOrPath string, call plugin.CommandCall,
) (_ plugin.CommandResult, err error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
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
	return run.Run(ctx, v, call)
}

// ExecuteSharedByName resolves the vault, opens the command runner over it, picks the handler of the shared
// command, and runs it.
func (u RunByName) ExecuteSharedByName(
	ctx context.Context, open RunnerOpener, nameOrPath string, handlers []plugin.CommandEntry,
	pluginID string, call plugin.CommandCall,
) (_ plugin.CommandResult, err error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
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
	handler, err := FindHandler(ctx, run, v, handlers, pluginID, call.Path)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	call.PluginID = handler.PluginID
	call.Command = handler.Command.ID
	return run.Run(ctx, v, call)
}
