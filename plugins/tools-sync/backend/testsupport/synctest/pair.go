package synctest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

const (
	// FirstVault is the vault of the first computer of a pair, connected as a new synced vault.
	FirstVault domain.VaultID = "Test"
	// SecondVault is the vault of the second computer of a pair, connected to the synced vault of the first.
	SecondVault domain.VaultID = "Copy"
)

// Pair is two computers of one person on one server.
type Pair struct {
	Server *Server
	Clock  *Clock
	A      *Computer
	B      *Computer
}

// NewPair returns the first and the second computer, each with its vault, the second tied to the synced vault of the
// first.
func NewPair(t *testing.T, cfg fakeserver.Config) *Pair {
	t.Helper()
	clock := NewClock(GetStart())
	server := NewServer(t, clock, cfg)
	words := NewWords(t)
	a := NewFirstComputer(t, server, clock, string(FirstVault), words)
	a.Connect(t, FirstVault)
	b := NewSecondComputer(t, server, clock, string(SecondVault), words)
	b.ConnectExisting(t, SecondVault, a.Link(t, FirstVault).RemoteVaultID)
	return &Pair{Server: server, Clock: clock, A: a, B: b}
}

// GetVault returns the synced vault the two computers share, with its key.
func (p *Pair) GetVault(t *testing.T) transfer.Vault {
	t.Helper()
	return transfer.Vault{ID: p.A.GetRemoteID(t, FirstVault), Key: p.A.GetVaultKey(t, FirstVault), Later: nil}
}
