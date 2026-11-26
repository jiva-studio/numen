package synctest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	domainsession "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/conflict"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/digest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/heads"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/vaultnames"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/endorse"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/linkcourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/preview"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/upload"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/errorwire"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

// Host is the host of a computer for the plugin: the vaults of the installation and the files and the store
// folder of each, served by the vaults of the SDK test host.
type Host struct {
	Root  *hosttest.Vault
	Names *vaultnames.Edge
}

func (h Host) GetVaults(ctx context.Context) ([]domain.Vault, error) { return h.Root.GetVaults(ctx) }

func (h Host) GetFiles(id domain.VaultID) ports.Files {
	if h.Names == nil {
		return h.Root.With(id)
	}
	return h.Names.GetFiles(id, h.Root.With(id))
}

func (h Host) GetNotes(id domain.VaultID) ports.Notes {
	if h.Names == nil {
		return h.Root.With(id)
	}
	return h.Names.GetNotes(id, h.Root.With(id), h.Root.With(id))
}

func (h Host) GetAbsent(id domain.VaultID) ports.Absent {
	if h.Names == nil {
		return h.Root.With(id)
	}
	return h.Names.GetAbsent(id, h.Root.With(id), h.Root.With(id))
}

func (h Host) Store(id domain.VaultID) ports.Store { return h.Root.Store().With(id) }

func (h Host) GetTransfers(id domain.VaultID) ports.Transfers {
	if h.Names == nil {
		return h.Root.With(id)
	}
	return h.Names.GetTransfers(id, h.Root.With(id), h.Root.With(id))
}

// Computer is one computer of the person: an identity held by a host, a health host, vaults, a clock and the
// adapters of the plugin over them, signed in to the server.
type Computer struct {
	Clock     *Clock
	Server    *Server
	Fake      *accounttest.Fake
	Client    *account.Client
	Health    *healthtest.Host
	Vaults    *hosttest.Vault
	Host      ports.Host
	Notes     ports.NoteHost
	Cloud     *cloud.Cloud
	Roots     []ed25519.PublicKey
	Account   *accountadapter.Account
	Sessions  *serveradapter.Server
	State     *hoststore.State
	Folders   *folders.ReceivedFolders
	LazyHeavy bool

	subscriptions  ports.Subscriptions
	network        *link
	healthEndpoint hostcall.Endpoint
	token          atomic.Value
	downloadsOnce  sync.Once
	downloads      *download.DownloadFile
	retries        *pull.Retries
}

func NewWords(t *testing.T) []string {
	t.Helper()
	return NewWordsFor(t, NewLabel(t))
}

// DefaultPlan is the plan a first computer buys, the cheapest of the payment stand-in.
const DefaultPlan = "prod_basic"

// NewFirstComputer makes a computer whose identity is the one the words recover, with a vault of the name, signs it in and
// buys the default plan: the account does not exist on the server before this call.
func NewFirstComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	return NewFirstComputerOnPlan(t, server, clock, vault, words, DefaultPlan)
}

func NewFirstComputerOnPlan(t *testing.T, server *Server, clock *Clock, vault string, words []string, plan string) *Computer {
	t.Helper()
	c := NewUnsubscribedComputer(t, server, clock, vault, words)
	c.Subscribe(t, server.Stack, plan)
	return c
}

func NewUnsubscribedComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	c := newComputer(t, server, clock, vault)
	c.Fake.StartRestore(words)
	if err := c.Client.CompleteRestore(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SignIn(t.Context()); err != nil {
		t.Fatal(err)
	}
	return c
}

// NewSecondComputer makes a computer that restores the identity of the words, joining the account the first
// computer made on the server, with a vault of the name, and signs it in.
func NewSecondComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	c := newComputer(t, server, clock, vault)
	c.Fake.StartRestore(words)
	if _, err := join.NewJoin(c.JoinDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := c.SignIn(t.Context()); err != nil {
		t.Fatal(err)
	}
	return c
}

// NewRestoredComputer makes a computer that restores the identity of the words as the core does: the other devices of the account stay and
// the keys move to a new epoch. It joins the account on the server, has a vault of the name, and signs in.
func NewRestoredComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	c := newComputer(t, server, clock, vault)
	c.Fake.StartRestoreAsCore(words)
	if _, err := join.NewJoin(c.JoinDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := c.SignIn(t.Context()); err != nil {
		t.Fatal(err)
	}
	return c
}

func newComputer(t *testing.T, server *Server, clock *Clock, vault string) *Computer {
	t.Helper()
	fake := accounttest.NewFake(t, syncprotocol.SignInPluginID)
	fake.SetClock(clock)
	healthHost := healthtest.NewHost(t)
	accountEndpoint, err := hostcall.ReadVars(fake.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	healthEndpoint, err := hostcall.ReadVars(healthHost.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	client := account.NewClient(accountEndpoint)
	root := hosttest.New(vault)
	host := Host{Root: root, Names: vaultnames.New()}
	network := &link{isOffline: atomic.Bool{}, origin: server.origin, dial: server.dial, server: server}
	web := &http.Client{Transport: network}
	receivedRoot := t.TempDir()
	c := &Computer{
		Clock: clock, Server: server, Fake: fake, Client: client, Health: healthHost, Vaults: root, Host: host, Notes: host,
		Cloud: nil, Roots: server.getRoots(), State: nil,
		LazyHeavy: false, Folders: folders.NewReceivedFolders(func() string { return receivedRoot }), token: atomic.Value{},
		network: network, downloadsOnce: sync.Once{}, downloads: nil, retries: pull.NewRetries(), subscriptions: nil,
		Account:        accountadapter.New(client),
		Sessions:       serveradapter.New(serveradapter.Config{Address: server.URL, HTTP: web, Now: clock.Now}),
		healthEndpoint: healthEndpoint,
	}
	c.State = hoststore.New(c.Host)
	c.token.Store("")
	c.Cloud = c.NewCloud(server.URL, c.SignIn)
	return c
}

func (c *Computer) getToken() string {
	token, ok := c.token.Load().(string)
	if !ok {
		return ""
	}
	return token
}

func (c *Computer) SignIn(ctx context.Context) error {
	public, err := c.Account.GetPublicIdentity(ctx)
	if err != nil {
		return err
	}
	challenge, err := c.Sessions.GetChallenge(ctx, public.DeviceSignKey)
	if err != nil {
		return err
	}
	signature, err := c.Account.SignAsDevice(ctx, challenge.Message)
	if err != nil {
		return err
	}
	opened, err := c.Sessions.SignIn(ctx, domainsession.Proof{
		ChallengeID: challenge.ID, DeviceSignKey: public.DeviceSignKey, Signature: signature, Chain: public.Chain, RestoreAccount: false,
	})
	if err != nil {
		return err
	}
	c.token.Store(opened.Token)
	return nil
}

func (c *Computer) AddVault(t *testing.T, name string) {
	t.Helper()
	if _, err := c.Vaults.AddVault(t.Context(), "/vaults/"+name, name); err != nil {
		t.Fatal(err)
	}
}

func (c *Computer) NewHTTP() *http.Client { return &http.Client{Transport: c.network} }

func (c *Computer) NewHealth() *health.Client { return health.NewClient(c.healthEndpoint) }

func (c *Computer) CatalogDeps() catalog.Deps {
	return catalog.Deps{
		Catalog: c.Cloud, Chain: c.Cloud, Crypto: c.Account, Identity: c.Account, Host: c.Host, State: c.State, Journals: c.State,
		Cloud: c.Cloud, Challenges: c.Sessions, Forget: c.State, Health: c.NewHealth(), Random: rand.Reader, Locker: c.State,
	}
}

func (c *Computer) LocalChangesDeps() localchanges.Deps {
	return localchanges.Deps{Host: c.Host, State: c.State, Clock: c.Clock}
}

func (c *Computer) WireDeps() syncwire.Deps {
	uploader := cloud.NewUploader(c.Cloud, c.Clock.Now)
	wired := syncwire.Deps{
		Deps: c.SyncDeps(), Uploads: uploader, UploadEntries: c.State, DownloadBlobs: cloud.NewBlobs(c.Cloud), DownloadEntries: c.State,
		Pending: c.State, Approvals: c.State, Absences: c.State, RemovalDelay: func() time.Duration { return RemovalDelay }, Digests: digest.Sha256{},
		Conflicts: conflict.New(conflict.Deps{Host: c.Host, Log: nil}), LazyHeavy: c.LazyHeavy,
		Downloads: nil, Retries: c.retries,
	}
	c.downloadsOnce.Do(func() { c.downloads = syncwire.NewDownloads(wired) })
	wired.Downloads = c.downloads
	return wired
}

func (c *Computer) CommitDeps() commit.Deps {
	wired := c.WireDeps()
	uploads := upload.NewUploadFile(upload.Deps{
		Uploads: wired.Uploads, Entries: c.State, Crypto: c.Account, Host: c.Host, Random: rand.Reader, Digests: digest.Sha256{},
	})
	return commit.Deps{Revisions: c.Cloud, Blobs: c.Cloud, Identity: c.Account, Crypto: c.Account, Uploads: uploads, Random: rand.Reader}
}

func (c *Computer) PullDeps() pull.Deps {
	wired := c.WireDeps()
	return pull.Deps{
		Revisions: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Host: c.Host, State: c.State, Downloads: wired.Downloads,
		Guard: syncwire.NewGuard(wired), Health: c.NewHealth(),
		Conflicts: conflict.New(conflict.Deps{Host: c.Host, Log: nil}), Clock: c.Clock, LazyHeavy: c.LazyHeavy, Retries: c.retries,
	}
}

func (c *Computer) JoinDeps() join.Deps {
	return join.Deps{Identity: c.Account, Server: c.Sessions}
}

func (c *Computer) SyncDeps() syncvault.Deps {
	return syncvault.Deps{
		Catalog: c.Cloud, Chain: c.Cloud, Revisions: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Identity: c.Account,
		Host: c.Host, Notes: c.Notes, State: c.State, Progress: c.State, Health: c.NewHealth(), Clock: c.Clock, Random: rand.Reader, Subscriptions: c.getSubscriptions(), Shop: c.Cloud,
		Pauses: c.State, Restore: syncvault.RestoreDeps{Address: c.Server.URL, Epochs: c.Cloud, Mirror: c.State},
		Shares: c.Cloud, ShareStates: c.State, Box: c.Account, Vaults: c.Vaults, Folders: c.Folders, Forget: c.State, Cloud: c.Cloud, Challenges: c.Sessions, Locker: c.State,
	}
}

func (c *Computer) NewSyncVault() *syncvault.SyncVault {
	return syncwire.NewSyncVault(c.WireDeps())
}

func (c *Computer) NewSyncVaultReading(shares syncvault.SharesReader) *syncvault.SyncVault {
	return syncwire.NewSyncVaultReading(c.WireDeps(), shares)
}

func (c *Computer) Sync(ctx context.Context, id domain.VaultID) error {
	return c.NewSyncVault().Execute(ctx, id)
}

// NewPendingComputer makes a computer whose host keeps a restore of the words, with no identity and not signed
// in, as a second computer is before the plugin joins the account.
func NewPendingComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	c := newComputer(t, server, clock, vault)
	c.Fake.StartRestore(words)
	return c
}

func (c *Computer) RunDeps(interval time.Duration, changed <-chan domain.VaultID) syncvault.RunDeps {
	return syncvault.RunDeps{
		Pass: c.NewSyncVault(), Host: c.Host, State: c.State, Health: c.NewHealth(), Clock: c.Clock,
		Warn:     subscription.NewWarn(subscription.WarnDeps{Subscriptions: c.getSubscriptions(), Health: c.NewHealth()}),
		Interval: func() time.Duration { return interval }, Changed: changed,
		Describe: errorwire.GetText, Report: func(context.Context, domain.VaultID, error) {},
	}
}

// RemovalDelay is how long a removed file is watched before its removal is sent, in the passes of the tests.
const RemovalDelay = time.Second

func (c *Computer) Spy() *SpyHost {
	if spy, isSpy := c.Host.(*SpyHost); isSpy {
		return spy
	}
	spy := NewSpyHost(c.Host, c.Notes)
	c.Host = spy
	return spy
}

func (c *Computer) DetectDeps() localchanges.DetectDeps {
	return localchanges.DetectDeps{Host: c.Host, State: c.State, Clock: c.Clock, Delay: func() time.Duration { return RemovalDelay }}
}

func (c *Computer) DeleteDeps() deletions.Deps {
	return deletions.Deps{Pending: c.State, Approvals: c.State, Health: c.NewHealth(), Clock: c.Clock}
}

func (c *Computer) PreviewDeps() preview.Deps {
	return preview.Deps{
		Heads: heads.New(heads.Deps{Catalog: c.Cloud, Chain: c.Cloud, Revisions: c.Cloud, Crypto: c.Account, Identity: c.Account}),
		Host:  c.Host,
	}
}

func (c *Computer) RestoreDeps() restore.Deps {
	return restore.Deps{
		Address: c.Server.URL, Epochs: c.Cloud, Mirror: c.State, Catalog: c.Cloud, Chain: c.Cloud, Identity: c.Account, Revisions: c.Cloud,
		Blobs: c.Cloud, Crypto: c.Account, Host: c.Host, State: c.State, Commit: commit.NewCommitRevisions(c.CommitDeps()),
		Remove: commit.NewCommitRemovals(c.CommitDeps()), Clock: c.Clock,
	}
}

// NewCloud returns the cloud adapter of this computer over the address, with the stream client that has no
// timeout, both over the network link of this computer, and refresh as the way to sign in again.
func (c *Computer) NewCloud(address string, refresh func(context.Context) error) *cloud.Cloud {
	return cloud.New(cloud.Config{
		Address: address, HTTP: &http.Client{Transport: c.network}, Stream: &http.Client{Transport: c.network},
		Token: c.getToken, Refresh: refresh, Roots: c.Roots, Now: c.Clock.Now, Storage: nil, Serials: nil,
	})
}

func (c *Computer) RotateDeps() rotatevaultkeys.Deps {
	return rotatevaultkeys.Deps{
		Catalog: c.Cloud, Chain: c.Cloud, Crypto: c.Account, Identity: c.Account, Host: c.Host, State: c.State, Random: rand.Reader,
	}
}

func (c *Computer) ChainDeps() chaincourier.Deps {
	return chaincourier.Deps{
		Identity: c.Account, Core: c.Client, Chain: c.Cloud, Health: c.NewHealth(), Clock: c.Clock, Sightings: c.State, IsIdle: nil, Endorse: endorse.NewEndorse(c.EndorseDeps()),
	}
}

func (c *Computer) EndorseDeps() endorse.Deps {
	return endorse.Deps{Identity: c.Account, Host: c.Host, State: c.State, Revisions: c.Cloud}
}

func (c *Computer) LinkDeps() linkcourier.Deps {
	return linkcourier.Deps{
		Core: c.Client, Relay: syncclient.NewLinkClient(c.Server.URL, c.Server.NewHTTP()), Health: c.NewHealth(), Clock: c.Clock, Server: c.Server.URL,
	}
}

func (c *Computer) ShareDeps() share.Deps {
	return share.Deps{
		Shares: c.Cloud, Revisions: c.Cloud, Chain: c.Cloud, Crypto: c.Account, Identity: c.Account, Links: c.State, States: c.State,
		Clock: c.Clock, Random: rand.Reader,
	}
}

func (c *Computer) ReceiveDeps() receive.Deps {
	return receive.Deps{
		Shares: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Box: c.Account, Identity: c.Account, Host: c.Host, Notes: c.Notes, Links: c.State,
		Records: c.State, Statuses: c.State, Vaults: c.Vaults, Folders: c.Folders, Clock: c.Clock, Locker: c.State,
	}
}

func (c *Computer) ClearRoots() {
	c.Roots = nil
	c.Cloud = c.NewCloud(c.Server.URL, c.SignIn)
}
