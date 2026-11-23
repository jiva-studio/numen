// Package vaulttest gives the tests of the use cases a vault on disk that answers as the host does,
// and the ports over it.
package vaulttest

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// GetEpoch returns the moment that the clock of a test reports.
func GetEpoch() time.Time {
	return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)
}

// NewClock creates a clock that reports GetEpoch until it is moved.
func NewClock() *clocktest.Fake {
	source := new(clocktest.Fake)
	source.Set(GetEpoch())
	return source
}

// Queries answers queries about a vault's notes, links and parse problems from
// the index.
type Queries struct {
	hostports.Notes
	hostports.Links
	hostports.Problems
}

// NewFolders gives the plugin folder in the store of the vault with an identity, seen from the vault.
func NewFolders(host *hosttest.Vault) func(domain.VaultID) ports.HostFolder {
	return func(id domain.VaultID) ports.HostFolder { return host.Store().With(id) }
}

// Env is a vault written to a folder of the test's own, and the ports over it.
type Env struct {
	Host    *hosttest.Vault
	Store   *hosttest.Store
	Vault   domain.Vault
	Ports   hostports.Ports
	Queries Queries
}

// New is a vault of the notes given.
func New(tb testing.TB, notes map[string]string) Env {
	tb.Helper()
	host := hosttest.OnDisk(tb, createVaultName(tb), notes)
	return NewEnv(host)
}

func createVaultName(tb testing.TB) string {
	tb.Helper()
	own := tb.TempDir()
	return "vault-" + filepath.Base(filepath.Dir(own)) + "-" + filepath.Base(own)
}

// NewEnv is the ports over a vault the test made.
func NewEnv(host *hosttest.Vault) Env {
	ports := hostports.New(NewVaults(host), NewFolders(host))
	return Env{
		Host:    host,
		Store:   host.Store(),
		Vault:   host.Vault(),
		Ports:   ports,
		Queries: Queries{Notes: hostports.Notes{GetVault: NewVaults(host)}, Links: hostports.Links{GetVault: NewVaults(host)}, Problems: hostports.Problems{GetVault: NewVaults(host)}},
	}
}

// Installation is a host with several vaults, as the application has, and the
// ports over it.
type Installation struct {
	vaults  map[domain.VaultID]*hosttest.Vault
	Host    *hosttest.Vault
	Ports   hostports.Ports
	Queries Queries
}

// NewInstallation returns a host with no vault yet.
func NewInstallation() Installation {
	host := hosttest.New("installation")
	host.Hide()
	ports := hostports.New(NewVaults(host), NewFolders(host))
	return Installation{
		vaults:  map[domain.VaultID]*hosttest.Vault{},
		Host:    host,
		Ports:   ports,
		Queries: Queries{Notes: hostports.Notes{GetVault: NewVaults(host)}, Links: hostports.Links{GetVault: NewVaults(host)}, Problems: hostports.Problems{GetVault: NewVaults(host)}},
	}
}

// Add makes a vault of the notes given, known to the host.
func (i Installation) Add(tb testing.TB, notes map[string]string) domain.Vault {
	tb.Helper()
	added := hosttest.OnDisk(tb, createVaultName(tb), notes)
	i.Host.Beside(added)
	i.vaults[added.Vault().ID] = added
	return added.Vault()
}

// GetVault is the vault of that identity as the host holds it, for a test that changes how it answers.
func (i Installation) GetVault(id domain.VaultID) *hosttest.Vault { return i.vaults[id] }

// Patience is how long a test gives something happening in the background before it calls the thing
// broken. It is a claim about the slowest machine the suite must pass on.
const Patience = 10 * time.Second

// NewVaults gives the ports' calls on the vault of v with an identity; the empty identity is v itself.
func NewVaults(v *hosttest.Vault) func(domain.VaultID) ports.HostVault {
	return func(id domain.VaultID) ports.HostVault {
		on := v.With(id)
		return ports.HostVault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
	}
}
