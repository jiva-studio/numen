package synctest

import (
	"context"
	"crypto/rand"
	"net/http"
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
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainsession "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

// Host is the host of a computer for the plugin: the vaults of the installation and the files and the store
// folder of each, served by the vaults of the SDK test host.
type Host struct {
	Root *hosttest.Vault
}

func (h Host) GetVaults(ctx context.Context) ([]domain.Vault, error) { return h.Root.GetVaults(ctx) }

func (h Host) GetFiles(id domain.VaultID) ports.Files { return h.Root.With(id) }

func (h Host) GetIndexer(id domain.VaultID) ports.Indexer { return h.Root.With(id) }

// Store returns the store folder of the plugin in the vault.
func (h Host) Store(id domain.VaultID) ports.Store { return h.Root.Store().With(id) }

// Remover returns the removal of notes of the vault.
func (h Host) Remover(id domain.VaultID) ports.Remover { return h.Root.With(id) }

// Computer is one computer of the person: an identity held by a host, a health host, vaults, a clock and the
// adapters of the plugin over them, signed in to the server.
type Computer struct {
	Clock    *Clock
	Server   *Server
	Fake     *accounttest.Fake
	Client   *account.Client
	Health   *healthtest.Host
	Vaults   *hosttest.Vault
	Host     Host
	Cloud    *cloud.Cloud
	Account  *accountadapter.Account
	Sessions *serveradapter.Server
	State    *hoststore.State
	Folders  *folders.Received

	healthEndpoint hostcall.Endpoint
	token          atomic.Value
}

// NewWords makes recovery words that identify one account.
func NewWords(t *testing.T) []string {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return words.List()
}

// NewFirstComputer makes a computer whose identity is the one the words recover, with a vault of the name, and
// signs it in: the account does not exist on the server before this call.
func NewFirstComputer(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
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

func newComputer(t *testing.T, server *Server, clock *Clock, vault string) *Computer {
	t.Helper()
	fake := accounttest.NewFake(t, syncprotocol.SignInPluginID)
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
	receivedRoot := t.TempDir()
	c := &Computer{
		Clock: clock, Server: server, Fake: fake, Client: client, Health: healthHost, Vaults: root, Host: Host{Root: root},
		Cloud: nil, State: nil, Folders: folders.NewReceived(func() string { return receivedRoot }), token: atomic.Value{},
		Account:        accountadapter.New(client),
		Sessions:       serveradapter.New(serveradapter.Config{Address: server.URL, HTTP: http.DefaultClient, Now: clock.Now}),
		healthEndpoint: healthEndpoint,
	}
	c.State = hoststore.New(c.Host)
	c.token.Store("")
	c.Cloud = cloud.New(cloud.Config{Address: server.URL, HTTP: http.DefaultClient, Token: c.getToken, Refresh: c.SignIn})
	return c
}

func (c *Computer) getToken() string {
	token, ok := c.token.Load().(string)
	if !ok {
		return ""
	}
	return token
}

// SignIn signs this computer in to the server with its identity and keeps the session token for the calls.
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
		ChallengeID: challenge.ID, DeviceSignKey: public.DeviceSignKey, Signature: signature, Chain: public.Chain,
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

// NewHealth returns a client of the health host of this computer.
func (c *Computer) NewHealth() *health.Client { return health.NewClient(c.healthEndpoint) }

// CatalogDeps are the dependencies of the catalog use cases.
func (c *Computer) CatalogDeps() catalog.Deps {
	return catalog.Deps{
		Catalog: c.Cloud, Chain: c.Cloud, Crypto: c.Account, Identity: c.Account, Host: c.Host, State: c.State, Random: rand.Reader,
	}
}

// LocalChangesDeps are the dependencies of the scan of local changes.
func (c *Computer) LocalChangesDeps() localchanges.Deps {
	return localchanges.Deps{Host: c.Host, State: c.State, Clock: c.Clock}
}

// CommitDeps are the dependencies of the commit of revisions.
func (c *Computer) CommitDeps() commit.Deps {
	return commit.Deps{Revisions: c.Cloud, Blobs: c.Cloud, Identity: c.Account, Random: rand.Reader}
}

// PullDeps are the dependencies of the pull of changes.
func (c *Computer) PullDeps() pull.Deps {
	return pull.Deps{Revisions: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Host: c.Host, State: c.State}
}

// JoinDeps are the dependencies of the join of an existing account.
func (c *Computer) JoinDeps() join.Deps {
	return join.Deps{Identity: c.Account, Server: c.Sessions}
}

// SyncDeps are the dependencies of the pass over one vault and of the loop over the vaults.
func (c *Computer) SyncDeps() syncvault.Deps {
	return syncvault.Deps{
		Catalog: c.Cloud, Chain: c.Cloud, Revisions: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Identity: c.Account,
		Host: c.Host, State: c.State, Progress: c.State, Health: c.NewHealth(), Clock: c.Clock, Random: rand.Reader, Subscriptions: c.Cloud,
		Shares: c.Cloud, ShareStates: c.State, Box: c.Account, Vaults: c.Vaults, Folders: c.Folders,
	}
}

// NewSyncVault returns the pass over one vault with the use cases of its steps.
func (c *Computer) NewSyncVault() *syncvault.SyncVault {
	return c.NewSyncVaultReading(share.NewReadShares(c.ShareDeps()))
}

// NewSyncVaultReading is NewSyncVault with another way to read the shares of a vault at the start of a pass.
func (c *Computer) NewSyncVaultReading(shares syncvault.SharesReader) *syncvault.SyncVault {
	return syncvault.NewSyncVault(c.SyncDeps(), syncvault.Steps{
		Rename:       catalog.NewRenameRemoteVault(c.CatalogDeps()),
		Pull:         pull.NewPullChanges(c.PullDeps()),
		Scan:         localchanges.NewScanLocalChanges(c.LocalChangesDeps()),
		Commit:       commit.NewCommitRevisions(c.CommitDeps()),
		Subscription: subscription.NewReadSubscription(subscription.Deps{Subscriptions: c.Cloud}),
		Shares:       shares,
		Rewrap:       share.NewRewrapFolder(c.ShareDeps()),
		Receive:      receive.NewPullShare(c.ReceiveDeps()),
	})
}

// Sync runs one pass over the vault.
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

// RunDeps are the dependencies of the loop over the vaults, with a pass over one vault as SyncDeps makes it.
func (c *Computer) RunDeps(interval time.Duration, changed <-chan domain.VaultID) syncvault.RunDeps {
	return syncvault.RunDeps{
		Pass: c.NewSyncVault(), Host: c.Host, State: c.State, Health: c.NewHealth(), Clock: c.Clock,
		Warn:     subscription.NewWarn(subscription.WarnDeps{Subscriptions: c.Cloud, Health: c.NewHealth()}),
		Interval: func() time.Duration { return interval }, Changed: changed,
	}
}

// ShareDeps are the dependencies of the use cases of sharing a folder.
func (c *Computer) ShareDeps() share.Deps {
	return share.Deps{
		Shares: c.Cloud, Revisions: c.Cloud, Chain: c.Cloud, Crypto: c.Account, Identity: c.Account, Links: c.State, States: c.State,
		Clock: c.Clock, Random: rand.Reader,
	}
}

// ReceiveDeps are the dependencies of the use cases of receiving a folder.
func (c *Computer) ReceiveDeps() receive.Deps {
	return receive.Deps{
		Shares: c.Cloud, Blobs: c.Cloud, Crypto: c.Account, Box: c.Account, Identity: c.Account, Host: c.Host, Links: c.State,
		Records: c.State, Statuses: c.State, Vaults: c.Vaults, Folders: c.Folders, Clock: c.Clock,
	}
}
