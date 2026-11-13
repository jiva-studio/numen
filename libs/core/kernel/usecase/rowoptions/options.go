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

// GrantedReadTimeout is how long a read of the options may take from a provider that holds a grant: it
// reads the options of the connections the grant reaches, one after another.
const GrantedReadTimeout = 15 * time.Second

// RefreshLimit is how long a refresh in the background may take to open the provider and read the options.
const RefreshLimit = 30 * time.Second

// ConfigRevisions names the configuration a plugin has now; equal revisions are one configuration.
type ConfigRevisions interface {
	GetPluginConfigRevision(pluginID string) string
}

// GrantRevision names the connections a plugin reaches through its grants as they are now, and returns
// false where the plugin holds no grant.
type GrantRevision func(pluginID string) (string, bool, error)

// Deps is what Options lists from: the connections and their secrets, the providers Source opens, the
// providers Running opens while their plugins run, the Lists the answers are kept in, the Titles of the
// operations for the list of running plugins (nil names them by their ids), and the connections a provider
// reaches through its grants.
type Deps struct {
	Resolve     connectionport.ResolveConnection
	ReadSecrets connectionport.ReadConnectionSecrets
	Source      connectionport.ProviderSource
	Running     RunningProviders
	Lists       *Lists
	Titles      connectionport.OperationTitles
	Grants      GrantRevision
}

// Options lists the values a row field of a connection can take. A listing is kept in the lists and
// answered from them; the provider of the connection is asked when none is kept, when the settings ask
// for the rows of the connection, when the connection changes, and after a call through the connection
// while its plugin runs. No cooldown applies. A listing counts only for the configuration its provider had,
// and for a provider that holds a grant, only for the connections the grant reached then. An empty listing
// and one the provider marks partial are not kept.
type Options struct {
	deps Deps
}

// NewOptions returns the listing over deps.
func NewOptions(deps Deps) *Options {
	return &Options{deps: deps}
}

// request is a listing to ask for: what the answer is kept under, how long the read may take, and whether
// the provider reads past any list it keeps.
type request struct {
	key     connection.OptionList
	timeout time.Duration
	isFresh bool
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
	asked, err := o.newRequest(resolved, field, options, operations)
	if err != nil {
		return nil, err
	}
	if listed, isKept := o.deps.Lists.get(resolved.PluginID, asked.key); isKept {
		return listed, nil
	}
	return o.readOptions(ctx, resolved, declared, asked)
}

// RefreshOptions asks the provider, which may start its plugin, for the values the row field can take read
// anew, past any list the provider keeps, keeps them in place of the listing kept before and returns them.
// It fails as ListOptions does.
func (o *Options) RefreshOptions(
	ctx context.Context, connectionID, field string, options map[string]string, operations []string,
) ([]modelcall.Option, error) {
	resolved, declared, err := o.resolveField(ctx, connectionID, field)
	if err != nil {
		return nil, err
	}
	asked, err := o.newRequest(resolved, field, options, operations)
	if err != nil {
		return nil, err
	}
	asked.isFresh = true
	return o.readOptions(ctx, resolved, declared, asked)
}

// RefreshConnection asks the provider again, in the background and starting its plugin where needed, for
// every listing kept for the connection, with the connection as it is now and past any list the provider
// keeps. A failure keeps nothing.
func (o *Options) RefreshConnection(connectionID string) {
	go func() {
		ctx, cancel := context.WithTimeout(plugin.WithBackground(context.Background(), plugin.WhoConnections), RefreshLimit)
		defer cancel()
		o.refreshKept(ctx, connectionID, o.deps.Source.OpenProvider, true)
	}()
}

// DropConnection forgets the listings kept for a connection of the plugin.
func (o *Options) DropConnection(pluginID, connectionID string) {
	o.deps.Lists.drop(pluginID, connectionID)
}

// RefreshRunning asks the provider again, in the background, for every listing kept for the connection,
// only while the plugin of the connection runs: a plugin that does not run is not started. While such a
// refresh of the connection runs, another is not started.
func (o *Options) RefreshRunning(connectionID string) {
	if !o.deps.Lists.begin(connectionID) {
		return
	}
	go func() {
		defer o.deps.Lists.end(connectionID)
		ctx, cancel := context.WithTimeout(plugin.WithBackground(context.Background(), plugin.WhoConnections), RefreshLimit)
		defer cancel()
		o.refreshKept(ctx, connectionID, o.openRunningProvider, false)
	}()
}

func (o *Options) refreshKept(
	ctx context.Context, connectionID string,
	openProvider func(ctx context.Context, pluginID string) (connectionport.Provider, func(), error), isFresh bool,
) {
	resolved, err := o.deps.Resolve(connectionID)
	if err != nil {
		return
	}
	requests := o.deps.Lists.listRequests(resolved.PluginID, connectionID)
	if len(requests) == 0 {
		return
	}
	provider, release, err := openProvider(plugin.WithPurpose(ctx, "refresh the lists of "+resolved.Name), resolved.PluginID)
	if err != nil {
		return
	}
	defer release()
	secrets, err := o.deps.ReadSecrets(ctx, resolved)
	if err != nil {
		return
	}
	for _, kept := range requests {
		declared, isDeclared := resolved.Declaration.GetRowField(kept.Field)
		if !isDeclared || declared.OptionsFrom == "" {
			continue
		}
		asked, err := o.newRequest(resolved, kept.Field, kept.RowOptions, kept.Operations)
		if err != nil {
			return
		}
		asked.isFresh = isFresh
		if _, err := o.askProvider(ctx, provider, resolved, declared.OptionsFrom, secrets, asked); err != nil {
			return
		}
	}
}

func (o *Options) resolveField(ctx context.Context, connectionID, field string) (connection.Resolved, manifest.ConnectionField, error) {
	if err := ctx.Err(); err != nil {
		return connection.Resolved{}, manifest.ConnectionField{}, err
	}
	resolved, err := o.deps.Resolve(connectionID)
	if err != nil {
		return connection.Resolved{}, manifest.ConnectionField{}, err
	}
	declared, isDeclared := resolved.Declaration.GetRowField(field)
	if !isDeclared || declared.OptionsFrom == "" {
		return connection.Resolved{}, manifest.ConnectionField{}, fmt.Errorf("%w: %s", ErrNoOptions, field)
	}
	return resolved, declared, nil
}

func (o *Options) newRequest(
	resolved connection.Resolved, field string, options map[string]string, operations []string,
) (request, error) {
	grantRevision, isGranted, err := o.deps.Grants(resolved.PluginID)
	if err != nil {
		return request{}, err
	}
	key := newOptionList(resolved, field, options, operations)
	key.ConfigRevision = o.deps.Lists.getConfigRevision(resolved.PluginID)
	key.GrantRevision = grantRevision
	timeout := ReadTimeout
	if isGranted {
		timeout = GrantedReadTimeout
	}
	return request{key: key, timeout: timeout}, nil
}

func (o *Options) readOptions(
	ctx context.Context, resolved connection.Resolved, declared manifest.ConnectionField, asked request,
) ([]modelcall.Option, error) {
	provider, release, err := o.deps.Source.OpenProvider(plugin.WithPurpose(ctx, o.describeOperation(declared.OptionsFrom)+" of "+resolved.Name), resolved.PluginID)
	if err != nil {
		return nil, failed(ctx, connection.FailureReasonUnavailable)
	}
	defer release()
	secrets, err := o.deps.ReadSecrets(ctx, resolved)
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
	secrets map[string]string, asked request,
) ([]modelcall.Option, error) {
	readCtx, stop := context.WithTimeout(ctx, asked.timeout)
	defer stop()
	query := modelcall.OptionsQuery{Operations: asked.key.Operations, IsFresh: asked.isFresh}
	listed, err := provider.ListOptions(readCtx, operation, resolved.NewCall(secrets, asked.key.RowOptions), query)
	if err != nil {
		return nil, err
	}
	if len(listed.Options) > 0 && !listed.IsPartial {
		kept := asked.key
		kept.Options = slices.Clone(listed.Options)
		o.deps.Lists.put(resolved.PluginID, kept)
	}
	return listed.Options, nil
}

func (o *Options) openRunningProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	provider, release, isRunning := o.deps.Running.OpenRunningProvider(ctx, pluginID)
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
	if o.deps.Titles == nil {
		return operation
	}
	return connection.GetOperationTitle(o.deps.Titles.ListOperationTitles(), operation)
}
