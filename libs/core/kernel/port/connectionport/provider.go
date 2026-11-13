package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// Provider is what a plugin that provides connection types carries out for the calls of other plugins.
// A call fails with *connection.TryNextError or *connection.StopError, with the error of the context when
// it was canceled, or with an error of its own when the provider could not be reached.
type Provider interface {
	// Call carries out the operation for the connection of call, sends its data frames in order and
	// returns the done payload. It returns the error of the first send that fails.
	Call(ctx context.Context, operation string, call modelcall.ProviderCall, payload []byte, send modelcall.Send) ([]byte, error)
	// ListOptions carries out an operation a row field names in optionsFrom and returns the values the
	// field can take.
	ListOptions(ctx context.Context, operation string, call modelcall.ProviderCall, query modelcall.OptionsQuery) (modelcall.ListedOptions, error)
	// ListDetected carries out connections.detected for the type of call, which names no connection, and returns
	// the connections the provider found ready on this machine.
	ListDetected(ctx context.Context, call modelcall.ProviderCall) ([]modelcall.DetectedConnection, error)
}

// ProviderSource opens the provider of a plugin.
type ProviderSource interface {
	// OpenProvider starts the plugin if it is not running and returns its provider with the function that
	// releases it. A plugin that cannot be started is an error.
	OpenProvider(ctx context.Context, pluginID string) (Provider, func(), error)
}
