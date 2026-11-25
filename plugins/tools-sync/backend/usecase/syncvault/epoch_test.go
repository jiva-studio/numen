package syncvault_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func advanceEpoch(t *testing.T, c *synctest.Computer) {
	t.Helper()
	if _, err := chaincourier.NewAdopt(c.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(c.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the chain of the epoch published", published, err)
	}
	if rotated, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context()); err != nil || rotated != 1 {
		t.Fatalf("rotated = %d, error = %v, want the vault rotated", rotated, err)
	}
}

func readKeys(t *testing.T, c *synctest.Computer, link vaultlink.Link) []vaultlink.VaultKey {
	t.Helper()
	keys, err := vaultlink.OpenKeys(t.Context(), c.Account, link)
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

func TestSyncVault_NoteAddedAfterNewEpochIsCommittedWithKeyOfEpoch(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Eggs.md", "eggs")
	sync(t, p.a, source)
	advanceEpoch(t, p.a)
	p.a.WriteNote(source, "Milk.md", "milk")

	sync(t, p.a, source)

	keys := readKeys(t, p.a, p.a.Link(t, source))
	if len(keys) != 2 || keys[1].Epoch != 1 {
		t.Fatalf("keys of the link = %+v, want epochs 0 and 1", keys)
	}
	var milk, eggs bool
	for _, stored := range p.a.ReadChanges(t, p.a.GetRemoteID(t, source)) {
		revision := stored.GetRevision()
		switch string(p.a.OpenBlob(t, keyOf(t, keys, revision.GetEpoch()), revision)) {
		case "milk":
			milk = revision.GetEpoch() == 1
		case "eggs":
			eggs = revision.GetEpoch() == 0
		}
	}
	if !milk || !eggs {
		t.Errorf("milk under epoch 1 = %v, eggs under epoch 0 = %v, want both", milk, eggs)
	}
}

func keyOf(t *testing.T, keys []vaultlink.VaultKey, epoch uint32) [32]byte {
	t.Helper()
	for _, key := range keys {
		if key.Epoch == epoch {
			return [32]byte(key.Key)
		}
	}
	t.Fatalf("no key of epoch %d", epoch)
	return [32]byte{}
}

func TestSyncVault_ADeviceLinkedAfterTheNewEpochReadsTheNotesOfBothPeriods(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, string(source), words)
	first.Connect(t, source)
	first.WriteNote(source, "Eggs.md", "eggs")
	sync(t, first, source)
	advanceEpoch(t, first)
	first.WriteNote(source, "Milk.md", "milk")
	sync(t, first, source)

	late := synctest.NewSecondComputer(t, server, clock, string(target), words)
	late.ConnectExisting(t, target, first.Link(t, source).RemoteVaultID)
	sync(t, late, target)

	wantNote(t, late, target, "Eggs.md", "eggs")
	wantNote(t, late, target, "Milk.md", "milk")
	lateKeys, firstKeys := readKeys(t, late, late.Link(t, target)), readKeys(t, first, first.Link(t, source))
	if len(lateKeys) != 2 || !bytes.Equal(lateKeys[1].Key, firstKeys[1].Key) {
		t.Errorf("the late device holds %d keys, want the two keys of the first computer", len(lateKeys))
	}
}
