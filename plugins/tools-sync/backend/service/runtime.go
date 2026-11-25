package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/besthealth"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/conflict"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/device"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/digest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hostvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/jitter"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/logging"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/endorse"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/linkcourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/shareinvite"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/watchchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/downloadwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/invitewire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/settingswire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/sharewire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

const (
	defaultInterval     = 30
	defaultRemovalDelay = 60
	wakeBuffer          = 16
)

var errNotConfigured = sessiondomain.ErrNoServer

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
	parts        parts
	wakes        *loops.Wakes
	interval     *loops.Interval
	removal      *loops.Interval
	received     *loops.Text
	session      *loops.Supervisor[*Backend]
	isForeground loops.Flag
	isOnDemand   loops.Flag
	settings     *settingswire.Settings
	epochRoot    *loops.Text
	failures     *errorwire.Failures
}

func newRuntime(p parts) *runtime {
	r := &runtime{
		parts: p, wakes: loops.NewWakes(wakeBuffer), interval: loops.NewInterval(defaultInterval), removal: loops.NewInterval(defaultRemovalDelay),
		received: loops.NewText(), settings: settingswire.New(p.health, logging.Warn),
		epochRoot: loops.NewText(), failures: errorwire.NewFailures(), isForeground: loops.Flag{}, isOnDemand: loops.Flag{}, session: nil,
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
	r.isForeground.Set(values[settingForeground] == "true")
	r.interval.Set(values[settingInterval])
	r.removal.Set(values[settingRemovalDelay])
	r.received.Set(values[settingReceivedFolder])
	r.epochRoot.Set(values[settingswire.SettingEpochRoot])
	r.isOnDemand.Set(isOnDemandOn(values))
	r.session.Configure(r.settings.ReadServerAddress(values[settingswire.SettingServer]), getSessionOptions(values))
}

func isOnDemandOn(values map[string]string) bool { return values[settingOnDemand] != "false" }

func getSessionOptions(values map[string]string) string {
	return values[settingForeground] + "\x00" + values[settingswire.SettingEpochRoot] + "\x00" + strconv.FormatBool(isOnDemandOn(values))
}

func (r *runtime) start(ctx context.Context, address string) (*Backend, map[string]func(context.Context) error) {
	p := r.parts
	var life *lifecycle.Lifecycle
	isIdle := func() bool { return r.isForeground.IsOn() && (life == nil || !life.IsResumed()) }
	serverSide := serveradapter.New(serveradapter.Config{Address: address, HTTP: newHTTPClient(), Now: p.clock.Now})
	keep := session.NewKeep(session.Deps{Identity: p.identity, Server: serverSide, Health: p.getHealth(), Clock: p.clock, Chains: p.client, IsIdle: isIdle})
	roots, rootsErr := cloud.GetRoots(address, r.epochRoot.Get())
	r.settings.ReportEpochRoot(ctx, rootsErr)
	cloudSide := cloud.New(cloud.Config{
		Address: address, HTTP: newHTTPClient(), Token: keep.GetToken, Refresh: keep.Refresh, Stream: newStreamHTTPClient(),
		Storage: syncclient.NewBlobHTTPClient(address, requestTimeout),
		Roots:   roots, Now: p.clock.Now, Serials: cloud.NewSerials(p.state, address),
	})
	deps := syncvault.Deps{
		Catalog: cloudSide, Chain: cloudSide, Revisions: cloudSide, Blobs: cloudSide, Crypto: p.identity, Identity: p.identity,
		Host: p.vaults, Notes: p.vaults, State: p.state, Progress: p.state, Health: p.getHealth(), Clock: p.clock, Random: loops.NewRandom(),
		Subscriptions: cloudSide, Shop: cloudSide, Pauses: p.state, Shares: cloudSide, ShareStates: p.state, Box: p.identity, Vaults: p.vaults,
		Folders: folders.NewReceivedFolders(r.received.Get), Forget: p.state, Cloud: cloudSide, Locker: p.state,
		Restore: syncvault.RestoreDeps{Address: address, Epochs: cloudSide, Mirror: p.state},
	}
	wired := syncwire.Deps{
		Deps: deps, Uploads: cloud.NewUploader(cloudSide, p.clock.Now), UploadEntries: p.state, DownloadBlobs: cloud.NewBlobs(cloudSide),
		DownloadEntries: p.state, Pending: p.state, Approvals: p.state, Absences: p.state, RemovalDelay: r.removal.Get,
		Digests: digest.Sha256{}, Conflicts: conflict.New(conflict.Deps{Host: p.vaults, Log: nil}), LazyHeavy: r.isOnDemand.IsOn(),
		Downloads: nil, Retries: nil,
	}
	wired.Downloads = syncwire.NewDownloads(wired)
	pass := syncwire.NewSyncVault(wired)
	loop := syncvault.NewRunSync(syncvault.RunDeps{
		Pass: pass, Host: p.vaults, State: p.state, Health: p.getHealth(), Clock: p.clock,
		Warn:     subscription.NewWarn(subscription.WarnDeps{Subscriptions: cloudSide, Health: p.getHealth()}),
		Interval: r.interval.Get, Changed: r.wakes.WatchVaults(), Describe: errorwire.GetText, Report: logging.ReportPassFailure,
	})
	push := watchchanges.NewWatchChanges(watchchanges.Deps{Hints: cloudSide, Host: p.vaults, Links: p.state, Waker: r.wakes, Clock: p.clock, Jitter: jitter.NewSystem()})
	watch := join.NewWatch(join.NewJoin(join.Deps{Identity: p.identity, Server: serverSide}), p.clock)
	fetching := downloadwire.Deps{
		Fetch: fetchfile.Deps{
			Stage: wired.Downloads, Revisions: cloudSide, Blobs: wired.DownloadBlobs, Crypto: p.identity, Host: p.vaults, State: p.state,
			Shares: share.NewReadShares(sharewire.NewShareDeps(deps)), Identity: p.identity, Chain: cloudSide,
		},
		Entries: p.state,
	}
	downloads := downloadwire.New(fetching)
	hub := shareinvite.NewHub()
	invites := shareinvite.Deps{
		Relay: syncclient.NewLinkClient(address, newHTTPClient()), Health: p.getHealth(), Clock: p.clock, Identity: p.identity, Device: device.Host{},
		Sharing:   sharewire.NewFolderSharer(sharewire.NewShareDeps(deps)),
		Receiving: sharewire.NewInvitationReceiver(sharewire.NewReceiveDeps(deps)),
		Random:    deps.Random, Server: address, Hub: hub,
	}
	backend := NewBackend(deps).
		WithInvites(invitewire.New(invites)).
		WithDeletions(p.state, syncwire.NewResolveDeletions(wired)).
		WithTransfers(p.state, p.state).
		WithDownloads(downloads).
		WithAccountData(accountdata.Deps{Identity: p.identity, Challenges: serverSide, Account: cloudSide, Deletion: keep})
	life = lifecycle.NewLifecycle(lifecycle.Deps{
		Pass: pass, Loop: loop, Stream: idleStream{}, Host: p.vaults, State: p.state, Clock: p.clock,
	})
	backend.SetLifecycle(life)
	loopsOf := map[string]func(context.Context) error{
		"sign-in": keep.Execute, "restore": watch.Execute,
		"link": linkcourier.NewCourier(linkcourier.Deps{Core: p.client, Relay: syncclient.NewLinkClient(address, newHTTPClient()), Health: p.getHealth(), Clock: p.clock, Server: address}).Execute,
		"chain": chaincourier.NewLoop(chaincourier.Deps{
			Identity: p.identity, Core: p.client, Chain: cloudSide, Health: p.getHealth(), Clock: p.clock, Sightings: p.state,
			IsIdle:  isIdle,
			Endorse: endorse.NewEndorse(endorse.Deps{Identity: p.identity, Host: p.vaults, State: p.state, Revisions: cloudSide}),
		}, p.clock).Execute,
		"share-invite": shareinvite.NewCourier(hub).Execute,
		"downloads":    func(ctx context.Context) error { <-ctx.Done(); downloads.Close(); return ctx.Err() },
		"lifecycle":    func(ctx context.Context) error { <-ctx.Done(); return errors.Join(ctx.Err(), life.Close()) },
		"rotate":       rotatevaultkeys.NewLoop(rotatevaultkeys.LoopDeps{Deps: rotateDeps(deps), Health: p.getHealth(), IsIdle: isIdle}, p.clock).Execute,
	}
	if !r.isForeground.IsOn() {
		loopsOf["sync"] = loop.Execute
		loopsOf["watch"] = push.Execute
	}
	return backend, loopsOf
}

func rotateDeps(deps syncvault.Deps) rotatevaultkeys.Deps {
	return rotatevaultkeys.Deps{
		Catalog: deps.Catalog, Chain: deps.Chain, Crypto: deps.Crypto, Identity: deps.Identity, Host: deps.Host, State: deps.State, Random: deps.Random,
	}
}

func (p parts) getHealth() ports.Health { return besthealth.New(p.health, logging.Warn) }
