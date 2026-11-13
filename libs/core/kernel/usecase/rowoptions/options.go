package rowoptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

var (
	// ErrProviderUnavailable is a provider that cannot be started or that failed the call. The text after
	// the sentinel is the reason, from the enum of reasons and never from a service.
	ErrProviderUnavailable = errors.New("the provider of the connection is unavailable")
	// ErrNoOptions is a row field the type does not declare, or one that names no optionsFrom.
	ErrNoOptions = errors.New("the type lists no options for the field")
)

// ReadTimeout is how long a read of the options may take. The read is not repeated: the editor shows what
// it got, or the reason it got nothing.
const ReadTimeout = 2 * time.Second

// Options asks the provider of a connection for the values a row field can take. The connection is the
// one named, and no cooldown applies.
type Options struct {
	resolve     connectionport.ResolveConnection
	readSecrets connectionport.ReadConnectionSecrets
	source      connectionport.ProviderSource
	cache       *Cache
}

// NewOptions returns the listing over the connections and the providers the source opens, which answers
// from cache what the providers listed before.
func NewOptions(
	resolve connectionport.ResolveConnection, readSecrets connectionport.ReadConnectionSecrets, source connectionport.ProviderSource,
	cache *Cache,
) *Options {
	return &Options{resolve: resolve, readSecrets: readSecrets, source: source, cache: cache}
}

// ListOptions calls the operation the row field names in optionsFrom with the options the row holds so far,
// and returns the values the field can take; a listing in the cache is returned without the provider. A
// value the provider does not list is still accepted by the core. operations are the operations the
// setting of rows is used for. It fails with connection.ErrNotFound, connection.ErrTypeUnavailable,
// ErrNoOptions, ErrProviderUnavailable or the error of the context.
func (o *Options) ListOptions(
	ctx context.Context, connectionID, field string, options map[string]string, operations []string,
) ([]modelcall.Option, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resolved, err := o.resolve(connectionID)
	if err != nil {
		return nil, err
	}
	declared, isDeclared := resolved.Declaration.GetRowField(field)
	if !isDeclared || declared.OptionsFrom == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoOptions, field)
	}
	key, state := newCacheKey(connectionID, field, options, operations), newConnectionState(resolved)
	if listed, isCached := o.cache.get(key, state); isCached {
		return listed, nil
	}
	provider, release, err := o.source.OpenProvider(ctx, resolved.PluginID)
	if err != nil {
		return nil, failed(ctx, connection.FailureReasonUnavailable)
	}
	defer release()
	secrets, err := o.readSecrets(ctx, resolved)
	switch {
	case errors.Is(err, connection.ErrSecretMissing):
		return nil, fmt.Errorf("%w: key missing", ErrProviderUnavailable)
	case err != nil:
		return nil, err
	}
	readCtx, stop := context.WithTimeout(ctx, ReadTimeout)
	defer stop()
	listed, err := provider.ListOptions(readCtx, declared.OptionsFrom, resolved.NewCall(secrets, options), operations)
	if err != nil {
		return nil, failed(ctx, readReason(err))
	}
	o.cache.put(key, state, listed)
	return listed, nil
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
	return connection.FailureReasonUnavailable
}
