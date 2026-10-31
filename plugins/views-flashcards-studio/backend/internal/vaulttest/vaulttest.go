// Package vaulttest gives the tests of the use cases a vault on disk that answers as the host does,
// and the ports over it.
package vaulttest

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
)

// Epoch is the moment that the clock of a test reports.
var Epoch = time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)

// NewClock creates a clock that reports Epoch until it is moved.
func NewClock() *clock.Fake {
	return clock.NewFake(Epoch)
}

// Queries answers queries about a vault's notes, links and parse problems from
// the index.
type Queries struct {
	hostports.Notes
	hostports.Links
	hostports.Problems
}

// Env is a vault written to a folder of the test's own, and the ports over it.
type Env struct {
	// Host is the vault as the host works it.
	Host *hosttest.Vault
	// Store is the plugin's folder of the vault's store.
	Store *hosttest.Store
	// Vault is the vault as the use cases name it.
	Vault domain.Vault
	// Ports answers every port the use cases take.
	Ports hostports.Ports
	// Queries answers the index's questions as one value.
	Queries Queries
}

type vaultCounter struct {
	made atomic.Int64
}

var vaults vaultCounter

// New is a vault of the notes given.
func New(tb testing.TB, notes map[string]string) Env {
	tb.Helper()
	host := hosttest.OnDisk(tb, fmt.Sprintf("vault-%d", vaults.made.Add(1)), notes)
	return Of(host)
}

// Of is the ports over a vault the test made.
func Of(host *hosttest.Vault) Env {
	ports := hostports.New(host, host.Store())
	return Env{
		Host:    host,
		Store:   host.Store(),
		Vault:   host.Vault(),
		Ports:   ports,
		Queries: Queries{Notes: hostports.Notes{Vault: host}, Links: hostports.Links{Vault: host}, Problems: hostports.Problems{Vault: host}},
	}
}

// Installation is a host with several vaults, as the application has, and the
// ports over it.
type Installation struct {
	vaults map[domain.VaultID]*hosttest.Vault
	// 	// Host is the host as the plugin addresses it. It has no vault of its own.
	Host *hosttest.Vault
	// Ports answers every port the use cases take.
	Ports hostports.Ports
	// Queries answers the index's questions as one value.
	Queries Queries
}

// NewInstallation returns a host with no vault yet.
func NewInstallation() Installation {
	host := hosttest.New("installation")
	host.Hide()
	ports := hostports.New(host, host.Store())
	return Installation{
		vaults:  map[domain.VaultID]*hosttest.Vault{},
		Host:    host,
		Ports:   ports,
		Queries: Queries{Notes: hostports.Notes{Vault: host}, Links: hostports.Links{Vault: host}, Problems: hostports.Problems{Vault: host}},
	}
}

// Add makes a vault of the notes given, known to the host.
func (i Installation) Add(tb testing.TB, notes map[string]string) domain.Vault {
	tb.Helper()
	added := hosttest.OnDisk(tb, fmt.Sprintf("vault-%d", vaults.made.Add(1)), notes)
	i.Host.Beside(added)
	i.vaults[added.Vault().ID] = added
	return added.Vault()
}

// Of is the vault of that identity as the host holds it, for a test that changes how it answers.
func (i Installation) Of(id domain.VaultID) *hosttest.Vault { return i.vaults[id] }

// Patience is how long a test gives something happening in the background before it calls the thing
// broken. It is a claim about the slowest machine the suite must pass on.
const Patience = 10 * time.Second

// WaitFor holds until something is so, and fails the test if it never is.
func WaitFor(tb testing.TB, condition func() bool) {
	tb.Helper()
	const step = 10 * time.Millisecond
	for range int(Patience / step) {
		if condition() {
			return
		}
		time.Sleep(step)
	}
	tb.Fatal("it never came to be so")
}
