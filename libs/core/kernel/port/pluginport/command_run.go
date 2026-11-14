package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// CommandRun runs a command of a plugin checked against its declaration: its backend, its condition
// and its arguments.
type CommandRun interface {
	Execute(ctx context.Context, v domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error)
}
