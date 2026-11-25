package syncvault_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func TestSyncVault_RestoreOnAnotherComputerKeepsTheFirstComputerWorking(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, string(source), words)
	first.Connect(t, source)
	first.WriteNote(source, "Eggs.md", "eggs")
	sync(t, first, source)

	restored := synctest.NewRestoredComputer(t, server, clock, string(target), words)
	if _, err := chaincourier.NewAdopt(first.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	own, err := first.Account.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.ListDevices()) != 2 || len(chain.ListRevokedDevices()) != 0 || chain.Epoch() != 1 {
		t.Errorf("devices = %d, revoked = %d, epoch = %d, want both computers active with the keys of the next epoch",
			len(chain.ListDevices()), len(chain.ListRevokedDevices()), chain.Epoch())
	}
	restored.ConnectExisting(t, target, first.Link(t, source).RemoteVaultID)
	sync(t, restored, target)
	wantNote(t, restored, target, "Eggs.md", "eggs")
	first.WriteNote(source, "Milk.md", "milk")
	sync(t, first, source)
	restored.WriteNote(target, "Bread.md", "bread")
	sync(t, restored, target)
	sync(t, first, source)
	sync(t, restored, target)
	wantNote(t, restored, target, "Milk.md", "milk")
	wantNote(t, first, source, "Bread.md", "bread")
}
