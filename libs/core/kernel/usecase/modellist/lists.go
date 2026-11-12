package modellist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// ErrProviderUnavailable is a provider that cannot be started or that failed the call. The text after
// the sentinel is the reason, from the enum of reasons and never from a service.
var ErrProviderUnavailable = errors.New("the provider of the connection is unavailable")

// ReadTimeout is how long a read of the models or the reasoning levels may take. The read is not repeated:
// the editor shows what it got, or the reason it got nothing.
const ReadTimeout = 2 * time.Second

// Lists asks the provider of a connection for what the editor offers: the models and the reasoning
// levels. The connection is the one named, and no cooldown applies.
type Lists struct {
	resolve     connectionport.ResolveConnection
	readSecrets connectionport.ReadConnectionSecrets
	source      connectionport.CatalogSource
}

// NewLists returns the lists over the resolver of connections and the catalogs the source opens.
func NewLists(
	resolve connectionport.ResolveConnection, readSecrets connectionport.ReadConnectionSecrets, source connectionport.CatalogSource,
) *Lists {
	return &Lists{resolve: resolve, readSecrets: readSecrets, source: source}
}

// ListModels returns the models of the connection that fit the operation. A model the service does not
// list is still accepted by the core. It fails with connection.ErrNotFound, connection.ErrTypeUnavailable,
// ErrProviderUnavailable or the error of the context.
func (l *Lists) ListModels(ctx context.Context, connectionID, operation string) ([]modelcall.Model, error) {
	return ask(ctx, l, connectionID, "", func(ctx context.Context, provider connectionport.ModelCatalog, call modelcall.ProviderCall) ([]modelcall.Model, error) {
		return provider.ListModels(ctx, call, operation)
	})
}

// ListReasoningLevels returns the reasoning levels the model of the connection takes. It fails like
// ListModels.
func (l *Lists) ListReasoningLevels(ctx context.Context, connectionID, model string) ([]modelcall.ReasoningLevel, error) {
	return ask(ctx, l, connectionID, model, func(ctx context.Context, provider connectionport.ModelCatalog, call modelcall.ProviderCall) ([]modelcall.ReasoningLevel, error) {
		return provider.ListReasoningLevels(ctx, call)
	})
}

func ask[T any](
	ctx context.Context, l *Lists, connectionID, model string,
	do func(context.Context, connectionport.ModelCatalog, modelcall.ProviderCall) (T, error),
) (T, error) {
	var none T
	if err := ctx.Err(); err != nil {
		return none, err
	}
	resolved, err := l.resolve(connectionID)
	if err != nil {
		return none, err
	}
	provider, release, err := l.source.OpenCatalog(ctx, resolved.PluginID)
	if err != nil {
		return none, failed(ctx, connection.FailureReasonUnreachable)
	}
	defer release()
	secrets, err := l.readSecrets(ctx, resolved)
	switch {
	case errors.Is(err, connection.ErrSecretMissing):
		return none, fmt.Errorf("%w: key missing", ErrProviderUnavailable)
	case err != nil:
		return none, err
	}
	readCtx, stop := context.WithTimeout(ctx, ReadTimeout)
	defer stop()
	result, err := do(readCtx, provider, resolved.NewCall(secrets, model, ""))
	if err != nil {
		if readCtx.Err() != nil && ctx.Err() == nil {
			return none, failed(ctx, connection.FailureReasonTimeout)
		}
		return none, failed(ctx, readReason(err))
	}
	return result, nil
}

func failed(ctx context.Context, reason connection.FailureReason) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %s", ErrProviderUnavailable, reason)
}

func readReason(err error) connection.FailureReason {
	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	switch {
	case errors.As(err, &tryNext):
		return tryNext.Reason
	case errors.As(err, &stopError):
		return stopError.Reason
	}
	return connection.FailureReasonUnreachable
}
