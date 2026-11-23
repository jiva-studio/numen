package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hostvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

const (
	defaultInterval = 30
	wakeBuffer      = 16
)

var errNotConfigured = errors.New("no sync server is set")

// parts are the adapters the runtime builds a sync from.
type parts struct {
	identity *accountadapter.Account
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
	received *loops.Text
	session  *loops.Supervisor[*Backend]
}

func newRuntime(p parts) *runtime {
	r := &runtime{
		parts: p, wakes: loops.NewWakes(wakeBuffer), interval: loops.NewInterval(defaultInterval), received: loops.NewText(), session: nil,
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
	r.received.Set(values[settingReceivedFolder])
	r.session.Configure(values[settingServer])
}

func (r *runtime) start(_ context.Context, address string) (*Backend, map[string]func(context.Context) error) {
	p := r.parts
	serverSide := serveradapter.New(serveradapter.Config{Address: address, HTTP: newHTTPClient(), Now: p.clock.Now})
	keep := session.NewKeep(session.Deps{Identity: p.identity, Server: serverSide, Health: p.health, Clock: p.clock})
	cloudSide := cloud.New(cloud.Config{Address: address, HTTP: newHTTPClient(), Token: keep.GetToken, Refresh: keep.Refresh})
	deps := syncvault.Deps{
		Catalog: cloudSide, Chain: cloudSide, Revisions: cloudSide, Blobs: cloudSide, Crypto: p.identity, Identity: p.identity,
		Host: p.vaults, State: p.state, Progress: p.state, Health: p.health, Clock: p.clock, Random: loops.NewRandom(),
		Subscriptions: cloudSide, Shares: cloudSide, ShareStates: p.state, Box: p.identity, Vaults: p.vaults,
		Folders: folders.NewReceived(r.received.Get),
	}
	loop := syncvault.NewRunSync(syncvault.RunDeps{
		Pass: syncwire.NewSyncVault(deps), Host: p.vaults, State: p.state, Health: p.health, Clock: p.clock,
		Warn:     subscription.NewWarn(subscription.WarnDeps{Subscriptions: cloudSide, Health: p.health}),
		Interval: r.interval.Get, Changed: r.wakes.WatchVaults(),
	})
	watch := join.NewWatch(join.NewJoin(join.Deps{Identity: p.identity, Server: serverSide}), p.clock)
	return NewBackend(deps), map[string]func(context.Context) error{"sign-in": keep.Execute, "restore": watch.Execute, "sync": loop.Execute}
}
