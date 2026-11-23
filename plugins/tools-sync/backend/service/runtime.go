package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/conflict"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/digest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hostvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/linkcourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/watchchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

const (
	defaultInterval     = 30
	defaultRemovalDelay = 60
	wakeBuffer          = 16
)

var errNotConfigured = errors.New("no sync server is set")

// parts are the adapters the runtime builds a sync from.
type parts struct {
	identity *accountadapter.Account
	client   *account.Client
	health   *health.Client
	clock    clock.System
	vaults   *hostvault.Vaults
	state    *hoststore.State
}

// runtime keeps the sign-in, the restore of an identity and the sync loop running against the server of the
// setting server, and runs them again when that address changes.
type runtime struct {
	parts    parts
	wakes    *loops.Wakes
	interval *loops.Interval
	removal  *loops.Interval
	received *loops.Text
	session  *loops.Supervisor[*Backend]
}

func newRuntime(p parts) *runtime {
	r := &runtime{
		parts: p, wakes: loops.NewWakes(wakeBuffer), interval: loops.NewInterval(defaultInterval), removal: loops.NewInterval(defaultRemovalDelay),
		received: loops.NewText(), session: nil,
	}
	r.session = loops.NewSupervisor(r.start)
	return r
}

func (r *runtime) wake(vault domain.VaultID) { r.wakes.Wake(vault) }

func (r *runtime) stop() { r.session.Stop() }

func (r *runtime) getBackend() (*Backend, error) {
	backend, isStarted := r.session.Get()
	if !isStarted {
		return nil, errNotConfigured
	}
	return backend, nil
}

func (r *runtime) configure(values map[string]string) {
	r.interval.Set(values[settingInterval])
	r.removal.Set(values[settingRemovalDelay])
	r.received.Set(values[settingReceivedFolder])
	r.session.Configure(values[settingServer])
}

func (r *runtime) start(_ context.Context, address string) (*Backend, map[string]func(context.Context) error) {
	p := r.parts
	serverSide := serveradapter.New(serveradapter.Config{Address: address, HTTP: newHTTPClient(), Now: p.clock.Now})
	keep := session.NewKeep(session.Deps{Identity: p.identity, Server: serverSide, Health: p.health, Clock: p.clock, Chains: p.client})
	cloudSide := cloud.New(cloud.Config{
		Address: address, HTTP: newHTTPClient(), Token: keep.GetToken, Refresh: keep.Refresh, Stream: newStreamHTTPClient(),
	})
	deps := syncvault.Deps{
		Catalog: cloudSide, Chain: cloudSide, Revisions: cloudSide, Blobs: cloudSide, Crypto: p.identity, Identity: p.identity,
		Host: p.vaults, State: p.state, Progress: p.state, Health: p.health, Clock: p.clock, Random: loops.NewRandom(),
		Subscriptions: cloudSide, Shop: cloudSide, Pauses: p.state, Shares: cloudSide, ShareStates: p.state, Box: p.identity, Vaults: p.vaults,
		Folders: folders.NewReceived(r.received.Get),
		Restore: syncvault.RestoreDeps{Address: address, Epochs: cloudSide, Mirror: p.state},
	}
	wired := syncwire.Deps{
		Deps: deps, Uploads: cloud.NewUploader(cloudSide, p.clock.Now), UploadEntries: p.state, DownloadBlobs: cloud.NewBlobs(cloudSide),
		DownloadEntries: p.state, Pending: p.state, Approvals: p.state, Absences: p.state, RemovalDelay: r.removal.Get,
		Digests: digest.Sha256{}, Conflicts: conflict.New(conflict.Deps{Host: p.vaults, Log: nil}),
	}
	loop := syncvault.NewRunSync(syncvault.RunDeps{
		Pass: syncwire.NewSyncVault(wired), Host: p.vaults, State: p.state, Health: p.health, Clock: p.clock,
		Warn:     subscription.NewWarn(subscription.WarnDeps{Subscriptions: cloudSide, Health: p.health}),
		Interval: r.interval.Get, Changed: r.wakes.WatchVaults(),
	})
	push := watchchanges.NewWatchChanges(watchchanges.Deps{Hints: cloudSide, Host: p.vaults, Links: p.state, Waker: r.wakes, Clock: p.clock})
	watch := join.NewWatch(join.NewJoin(join.Deps{Identity: p.identity, Server: serverSide}), p.clock)
	return NewBackend(deps).WithDeletions(p.state, syncwire.NewResolveDeletions(wired)), map[string]func(context.Context) error{
		"sign-in": keep.Execute, "restore": watch.Execute, "sync": loop.Execute, "watch": push.Execute,
		"link":   linkcourier.NewCourier(linkcourier.Deps{Core: p.client, Relay: syncclient.NewLinkClient(address, newHTTPClient()), Health: p.health, Clock: p.clock}).Execute,
		"chain":  chaincourier.NewLoop(chaincourier.Deps{Identity: p.identity, Core: p.client, Chain: cloudSide}, p.clock).Execute,
		"rotate": rotatevaultkeys.NewLoop(rotateDeps(deps), p.clock).Execute,
	}
}

func rotateDeps(deps syncvault.Deps) rotatevaultkeys.Deps {
	return rotatevaultkeys.Deps{
		Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State, Random: deps.Random,
	}
}
