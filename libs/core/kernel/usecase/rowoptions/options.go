package rowoptions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

var (
	// ErrProviderUnavailable is a provider that cannot be started or that failed the call. The text after
	// the sentinel is the reason, from the enum of reasons and never from a service.
	ErrProviderUnavailable = errors.New("the provider of the connection is unavailable")
	// ErrNoOptions is a row field the type does not declare, or one that names no optionsFrom.
	ErrNoOptions = errors.New("the type lists no options for the field")

	errNotRunning = errors.New("the plugin does not run")
)

// ReadTimeout is how long a read of the options may take. The read is not repeated: the editor shows what
// it got, or the reason it got nothing.
const ReadTimeout = 2 * time.Second

// RefreshLimit is how long a refresh in the background may take to open the provider and read the options.
const RefreshLimit = 30 * time.Second

// Options lists the values a row field of a connection can take. A listing is kept in the lists and
// answered from them; the provider of the connection is asked when none is kept, when the settings ask
// for the rows of the connection, when the connection changes, and after a call through the connection
// while its plugin runs. No cooldown applies.
type Options struct {
	resolve     connectionport.ResolveConnection
	readSecrets connectionport.ReadConnectionSecrets
	source      connectionport.ProviderSource
	running     RunningProviders
	lists       *Lists
	titles      connectionport.OperationTitles
}

// NewOptions returns the listing over the connections, the providers source opens and the providers
// running opens while their plugins run, which keeps what the providers listed in lists. titles names the
// operations in the list of running plugins; a nil titles names them by their ids.
func NewOptions(
	resolve connectionport.ResolveConnection, readSecrets connectionport.ReadConnectionSecrets, source connectionport.ProviderSource,
	running RunningProviders, lists *Lists, titles connectionport.OperationTitles,
) *Options {
	return &Options{resolve: resolve, readSecrets: readSecrets, source: source, running: running, lists: lists, titles: titles}
}

// ListOptions returns the values the row field can take from the listing kept for the connection as it is
// now, and asks the provider, which may start its plugin, only where none is kept. options are the values
// the row holds; only those of the row fields declared before the field are sent to the provider and
// kept with the listing. operations are the operations the setting of rows is used for. A value the
// provider does not list is still accepted by the core. It fails with connection.ErrNotFound,
// connection.ErrTypeUnavailable, ErrNoOptions, ErrProviderUnavailable or the error of the context.
func (o *Options) ListOptions(
	ctx context.Context, connectionID, field string, options map[string]string, operations []string,
) ([]modelcall.Option, error) {
	resolved, declared, err := o.resolveField(ctx, connectionID, field)
	if err != nil {
		return nil, err
	}
	asked := newOptionList(resolved, field, options, operations)
	if listed, isKept := o.lists.get(resolved.PluginID, asked); isKept {
		return listed, nil
	}
	return o.readOptions(ctx, resolved, declared, asked)
}

// RefreshOptions asks the provider, which may start its plugin, for the values the row field can take,
// keeps them in place of the listing kept before and returns them. It fails as ListOptions does.
func (o *Options) RefreshOptions(
	ctx context.Context, connectionID, field string, options map[string]string, operations []string,
) ([]modelcall.Option, error) {
	resolved, declared, err := o.resolveField(ctx, connectionID, field)
	if err != nil {
		return nil, err
	}
	return o.readOptions(ctx, resolved, declared, newOptionList(resolved, field, options, operations))
}

// RefreshConnection asks the provider again, in the background and starting its plugin where needed, for
// every listing kept for the connection, with the connection as it is now. A failure keeps nothing.
func (o *Options) RefreshConnection(connectionID string) {
	go func() {
		ctx, cancel := context.WithTimeout(plugin.WithBackground(context.Background(), plugin.WhoConnections), RefreshLimit)
		defer cancel()
		o.refreshKept(ctx, connectionID, o.source.OpenProvider)
	}()
}

// RefreshRunning asks the provider again, in the background, for every listing kept for the connection,
// only while the plugin of the connection runs: a plugin that does not run is not started. While such a
// refresh of the connection runs, another is not started.
func (o *Options) RefreshRunning(connectionID string) {
	if !o.lists.begin(connectionID) {
		return
	}
	go func() {
		defer o.lists.end(connectionID)
		ctx, cancel := context.WithTimeout(plugin.WithBackground(context.Background(), plugin.WhoConnections), RefreshLimit)
		defer cancel()
		o.refreshKept(ctx, connectionID, o.openRunningProvider)
	}()
}

func (o *Options) refreshKept(
	ctx context.Context, connectionID string,
	openProvider func(ctx context.Context, pluginID string) (connectionport.Provider, func(), error),
) {
	resolved, err := o.resolve(connectionID)
	if err != nil {
		return
	}
	requests := o.lists.listRequests(resolved.PluginID, connectionID)
	if len(requests) == 0 {
		return
	}
	provider, release, err := openProvider(plugin.WithPurpose(ctx, "refresh the lists of "+resolved.Name), resolved.PluginID)
	if err != nil {
		return
	}
	defer release()
	secrets, err := o.readSecrets(ctx, resolved)
	if err != nil {
		return
	}
	for _, request := range requests {
		declared, isDeclared := resolved.Declaration.GetRowField(request.Field)
		if !isDeclared || declared.OptionsFrom == "" {
			continue
		}
		asked := newOptionList(resolved, request.Field, request.RowOptions, request.Operations)
		if _, err := o.askProvider(ctx, provider, resolved, declared.OptionsFrom, secrets, asked); err != nil {
			return
		}
	}
}

func (o *Options) resolveField(ctx context.Context, connectionID, field string) (connection.Resolved, manifest.ConnectionField, error) {
	if err := ctx.Err(); err != nil {
		return connection.Resolved{}, manifest.ConnectionField{}, err
	}
	resolved, err := o.resolve(connectionID)
	if err != nil {
		return connection.Resolved{}, manifest.ConnectionField{}, err
	}
	declared, isDeclared := resolved.Declaration.GetRowField(field)
	if !isDeclared || declared.OptionsFrom == "" {
		return connection.Resolved{}, manifest.ConnectionField{}, fmt.Errorf("%w: %s", ErrNoOptions, field)
	}
	return resolved, declared, nil
}

func (o *Options) readOptions(
	ctx context.Context, resolved connection.Resolved, declared manifest.ConnectionField, asked connection.OptionList,
) ([]modelcall.Option, error) {
	provider, release, err := o.source.OpenProvider(plugin.WithPurpose(ctx, o.describeOperation(declared.OptionsFrom)+" of "+resolved.Name), resolved.PluginID)
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
	listed, err := o.askProvider(ctx, provider, resolved, declared.OptionsFrom, secrets, asked)
	if err != nil {
		return nil, failed(ctx, readReason(err))
	}
	return listed, nil
}

func (o *Options) askProvider(
	ctx context.Context, provider connectionport.Provider, resolved connection.Resolved, operation string,
	secrets map[string]string, asked connection.OptionList,
) ([]modelcall.Option, error) {
	readCtx, stop := context.WithTimeout(ctx, ReadTimeout)
	defer stop()
	listed, err := provider.ListOptions(readCtx, operation, resolved.NewCall(secrets, asked.RowOptions), asked.Operations)
	if err != nil {
		return nil, err
	}
	asked.Options = slices.Clone(listed)
	o.lists.put(resolved.PluginID, asked)
	return listed, nil
}

func (o *Options) openRunningProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	provider, release, isRunning := o.running.OpenRunningProvider(ctx, pluginID)
	if !isRunning {
		return nil, nil, errNotRunning
	}
	return provider, release, nil
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

func (o *Options) describeOperation(operation string) string {
	if o.titles == nil {
		return operation
	}
	return connection.GetOperationTitle(o.titles.ListOperationTitles(), operation)
}
