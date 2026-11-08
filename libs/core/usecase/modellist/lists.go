package modellist

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
)

// ErrProviderUnavailable is a provider that cannot be started or that failed the call. The text after
// the sentinel is the reason, from the enum of reasons and never from a service.
var ErrProviderUnavailable = errors.New("the provider of the connection is unavailable")

// Lists asks the provider of a connection for what the editor offers: the models and the reasoning
// levels. The connection is the one named, and no cooldown applies.
type Lists struct {
	connections *connection.Connections
	source      connectionport.ProviderSource
}

// NewLists returns the lists over the connections and the providers the source opens.
func NewLists(connections *connection.Connections, source connectionport.ProviderSource) *Lists {
	return &Lists{connections: connections, source: source}
}

// ListModels returns the models of the connection that fit the operation. A model the service does not
// list is still accepted by the core. It fails with connection.ErrNotFound, connection.ErrTypeUnavailable,
// ErrProviderUnavailable or the error of the context.
func (l *Lists) ListModels(ctx context.Context, connectionID, operation string) ([]connectionport.Model, error) {
	return ask(ctx, l, connectionID, "", func(provider connectionport.ModelCatalog, call connectionport.ProviderCall) ([]connectionport.Model, error) {
		return provider.ListModels(ctx, call, operation)
	})
}

// ListReasoningLevels returns the reasoning levels the model of the connection takes. It fails like
// ListModels.
func (l *Lists) ListReasoningLevels(ctx context.Context, connectionID, model string) ([]connectionport.ReasoningLevel, error) {
	return ask(ctx, l, connectionID, model, func(provider connectionport.ModelCatalog, call connectionport.ProviderCall) ([]connectionport.ReasoningLevel, error) {
		return provider.ListReasoningLevels(ctx, call)
	})
}

func ask[T any](
	ctx context.Context, l *Lists, connectionID, model string,
	do func(connectionport.ModelCatalog, connectionport.ProviderCall) (T, error),
) (T, error) {
	var none T
	if err := ctx.Err(); err != nil {
		return none, err
	}
	resolved, err := l.connections.Resolve.Execute(connectionID)
	if err != nil {
		return none, err
	}
	provider, release, err := l.source.OpenProvider(ctx, resolved.PluginID)
	if err != nil {
		return none, failed(ctx, connectionport.FailureReasonUnreachable)
	}
	defer release()
	secrets, err := l.connections.ReadSecrets.Execute(ctx, resolved)
	switch {
	case errors.Is(err, connection.ErrSecretMissing):
		return none, fmt.Errorf("%w: key missing", ErrProviderUnavailable)
	case err != nil:
		return none, err
	}
	result, err := do(provider, resolved.NewCall(secrets, model, ""))
	if err != nil {
		return none, failed(ctx, readReason(err))
	}
	return result, nil
}

func failed(ctx context.Context, reason connectionport.FailureReason) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %s", ErrProviderUnavailable, reason)
}

func readReason(err error) connectionport.FailureReason {
	var tryNext *connectionport.TryNextError
	var stopError *connectionport.StopError
	switch {
	case errors.As(err, &tryNext):
		return tryNext.Reason
	case errors.As(err, &stopError):
		return stopError.Reason
	}
	return connectionport.FailureReasonUnreachable
}
