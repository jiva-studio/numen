package service

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hostvault"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
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
	parts parts
	woken chan domain.VaultID
	secs  atomic.Int64

	mu      sync.Mutex
	address string
	cancel  context.CancelFunc
	done    chan struct{}
	current *Backend
}

func newRuntime(p parts) *runtime {
	r := &runtime{
		parts: p, woken: make(chan domain.VaultID, wakeBuffer), secs: atomic.Int64{},
		mu: sync.Mutex{}, address: "", cancel: nil, done: nil, current: nil,
	}
	r.secs.Store(defaultInterval)
	return r
}

func (r *runtime) wake(vault domain.VaultID) {
	select {
	case r.woken <- vault:
	default:
	}
}

func (r *runtime) getBackend() (*Backend, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return nil, errNotConfigured
	}
	return r.current, nil
}

func (r *runtime) configure(values map[string]string) {
	if secs, err := strconv.ParseInt(values[settingInterval], 10, 64); err == nil && secs >= 1 {
		r.secs.Store(secs)
	} else {
		r.secs.Store(defaultInterval)
	}
	address := values[settingServer]
	r.mu.Lock()
	defer r.mu.Unlock()
	if address == r.address && r.cancel != nil {
		return
	}
	r.stopWithLock()
	if address == "" {
		return
	}
	r.startWithLock(address)
}

func (r *runtime) startWithLock(address string) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	r.address, r.cancel, r.done = address, cancel, done
	p := r.parts
	serverSide := serveradapter.New(serveradapter.Config{Address: address, HTTP: newHTTPClient(), Now: p.clock.Now})
	keep := session.NewKeep(session.Deps{Identity: p.identity, Server: serverSide, Health: p.health, Clock: p.clock})
	cloudSide := cloud.New(cloud.Config{Address: address, HTTP: newHTTPClient(), Token: keep.GetToken, Refresh: keep.Refresh})
	deps := syncvault.Deps{
		Catalog: cloudSide, Chain: cloudSide, Revisions: cloudSide, Blobs: cloudSide, Crypto: p.identity, Identity: p.identity,
		Host: p.vaults, State: p.state, Health: p.health, Clock: p.clock, Random: rand.Reader, Subscriptions: cloudSide,
	}
	r.current = NewBackend(deps)
	loop := syncvault.NewRunSync(syncvault.RunDeps{
		Pass: syncvault.NewSyncVault(deps), Host: p.vaults, State: p.state, Health: p.health, Clock: p.clock,
		Subscriptions: cloudSide,
		Interval:      func() time.Duration { return time.Duration(r.secs.Load()) * time.Second }, Changed: r.woken,
	})
	watch := join.NewWatch(join.NewJoin(join.Deps{Identity: p.identity, Server: serverSide}), p.clock)
	var group sync.WaitGroup
	for name, run := range map[string]func(context.Context) error{"sign-in": keep.Execute, "restore": watch.Execute, "sync": loop.Execute} {
		group.Go(func() { report(ctx, name, run(ctx)) })
	}
	go func() {
		group.Wait()
		close(done)
	}()
}

func report(ctx context.Context, name string, err error) {
	if ctx.Err() == nil {
		slog.Warn("a loop of Numen Sync ended", "loop", name, "error", err)
	}
}

func (r *runtime) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopWithLock()
}

func (r *runtime) stopWithLock() {
	if r.cancel == nil {
		return
	}
	r.cancel()
	<-r.done
	r.address, r.cancel, r.done, r.current = "", nil, nil, nil
}
