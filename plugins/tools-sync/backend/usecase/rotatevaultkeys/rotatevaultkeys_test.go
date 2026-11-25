package rotatevaultkeys_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	"github.com/jiva-studio/numen/tools/syncserver/wiretest"
)

const vault = "Test"

func newComputer(t *testing.T) *synctest.Computer {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	c := synctest.NewFirstComputer(t, server, clock, vault, synctest.NewWords(t))
	c.Connect(t, vault)
	return c
}

func TestExecute_DoesNothingWhileTheEpochOfTheVaultIsTheEpochOfTheAccount(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	before := len(c.ReadCatalog(t))

	rotated, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context())

	if err != nil || rotated != 0 || len(c.ReadCatalog(t)) != before {
		t.Errorf("rotated = %d, error = %v, entries %d -> %d, want nothing changed", rotated, err, before, len(c.ReadCatalog(t)))
	}
}

func TestExecute_AppendsAnEntryWithTheNewKeyAndKeepsTheOldOne(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	oldKey := c.GetVaultKey(t, vault)
	before := len(c.ReadCatalog(t))
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}

	rotated, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context())

	if err != nil || rotated != 1 {
		t.Fatalf("rotated = %d, error = %v, want 1", rotated, err)
	}
	entries := c.ReadCatalog(t)
	if len(entries) != before+1 || entries[len(entries)-1].GetCreatesVault() {
		t.Fatalf("catalog has %d entries, the last creates a vault = %v, want one entry more that creates none", len(entries), entries[len(entries)-1].GetCreatesVault())
	}
	body := c.OpenCatalogBody(t, entries[len(entries)-1])
	if len(body.Keys) != 2 || body.Keys[0].Epoch != 0 || body.Keys[1].Epoch != 1 {
		t.Fatalf("keys of the body = %+v, want epochs 0 and 1", body.Keys)
	}
	if !bytes.Equal(body.Keys[0].Key, oldKey[:]) || bytes.Equal(body.Keys[1].Key, oldKey[:]) || len(body.Keys[1].Key) != 32 {
		t.Error("the old key changed, or the new key is the old one")
	}
}

func TestExecute_RotatesOncePerEpoch(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	if _, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	entries := len(c.ReadCatalog(t))

	rotated, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context())

	if err != nil || rotated != 0 || len(c.ReadCatalog(t)) != entries {
		t.Errorf("second run: rotated = %d, error = %v, entries %d -> %d, want nothing", rotated, err, entries, len(c.ReadCatalog(t)))
	}
}

func TestExecute_SecondDeviceRotatesOnceAndFirstAdoptsItsKey(t *testing.T) {
	t.Parallel()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, vault, words)
	first.Connect(t, vault)
	if err := first.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(first.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v", published, err)
	}
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	second.ConnectExisting(t, "Copy", first.Link(t, vault).RemoteVaultID)

	rotatedBySecond, err := rotatevaultkeys.NewRotateVaultKeys(second.RotateDeps()).Execute(t.Context())
	if err != nil || rotatedBySecond != 1 {
		t.Fatalf("second device rotated = %d, error = %v, want 1", rotatedBySecond, err)
	}
	entries := len(first.ReadCatalog(t))
	rotatedByFirst, err := rotatevaultkeys.NewRotateVaultKeys(first.RotateDeps()).Execute(t.Context())

	if err != nil || rotatedByFirst != 0 || len(first.ReadCatalog(t)) != entries {
		t.Fatalf("first device rotated = %d, error = %v, entries %d -> %d, want the entry of the second device kept", rotatedByFirst, err, entries, len(first.ReadCatalog(t)))
	}
	keysOfFirst, err := vaultlink.OpenKeys(t.Context(), first.Account, first.Link(t, vault))
	if err != nil {
		t.Fatal(err)
	}
	keysOfSecond, err := vaultlink.OpenKeys(t.Context(), second.Account, second.Link(t, "Copy"))
	if err != nil {
		t.Fatal(err)
	}
	if len(keysOfFirst) != 2 || len(keysOfSecond) != 2 || !bytes.Equal(keysOfFirst[1].Key, keysOfSecond[1].Key) || keysOfFirst[1].Epoch != 1 {
		t.Errorf("keys of the first = %d, of the second = %d, want the same key of epoch 1 on both", len(keysOfFirst), len(keysOfSecond))
	}
}

func TestExecute_AVaultAlreadyRotatedCostsNoCallToTheServer(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	if _, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	reads := c.Server.Fake.CountCalls(syncv1connect.CatalogServiceGetCatalogProcedure)

	if _, err := rotatevaultkeys.NewRotateVaultKeys(c.RotateDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := c.Server.Fake.CountCalls(syncv1connect.CatalogServiceGetCatalogProcedure); got != reads {
		t.Errorf("catalog reads %d -> %d, want none for a vault whose key is of the epoch of the account", reads, got)
	}
}

type forgedCatalog struct {
	*cloud.Cloud

	forge func(entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry
}

func (f *forgedCatalog) GetCatalog(ctx context.Context, since uint64) ([]*syncv1.CatalogEntry, error) {
	entries, err := f.Cloud.GetCatalog(ctx, since)
	if err != nil {
		return nil, err
	}
	cloned := make([]*syncv1.CatalogEntry, len(entries))
	for i, entry := range entries {
		cloned[i] = wiretest.CloneCatalogEntry(entry)
	}
	return f.forge(cloned), nil
}

func TestExecute_StoresNoKeyOfAnEntryASignerOutsideTheAccountForged(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	outsider := synctest.NewFirstComputer(t, c.Server, synctest.NewClock(synctest.GetStart()), "Other", synctest.NewWords(t))
	before := c.Link(t, vault)
	deps := c.RotateDeps()
	deps.Catalog = &forgedCatalog{Cloud: c.Cloud, forge: func(entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry {
		last := entries[len(entries)-1]
		plain, err := vaultlink.CatalogBody{LocalID: "x", Name: "x", Keys: []vaultlink.VaultKey{{Epoch: 1, Key: bytes.Repeat([]byte{7}, 32)}}}.Encode()
		if err != nil {
			t.Fatal(err)
		}
		forged := &syncv1.CatalogEntry{Version: last.GetVersion() + 1, VaultId: last.GetVaultId()}
		prev := syncprotocol.GetCatalogEntryID(last)
		forged.PrevHash = prev[:]
		if forged.EncBody, err = c.Account.SealData(t.Context(), synctest.GetCatalogContext(forged), plain); err != nil {
			t.Fatal(err)
		}
		outsider.SignCatalogEntry(t, forged)
		return append(entries, forged)
	}}

	rotated, err := rotatevaultkeys.NewRotateVaultKeys(deps).Execute(t.Context())

	if !errors.Is(err, rotatevaultkeys.ErrInvalidCatalog) || rotated != 0 {
		t.Errorf("rotated = %d, error = %v, want ErrInvalidCatalog", rotated, err)
	}
	if after := c.Link(t, vault); !bytes.Equal(after.SealedKey, before.SealedKey) || len(after.LaterKeys) != len(before.LaterKeys) {
		t.Error("the link stores a key of the forged entry")
	}
}

type brokenChain struct{ ports.Chain }

func (brokenChain) GetAccountChain(context.Context) ([]byte, error) {
	return nil, errors.New("the chain is not served")
}

func TestLoop_ShowsAFailedRoundAndDoublesTheWaitUntilARoundSucceeds(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	if err := c.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	deps := c.RotateDeps()
	deps.Chain = brokenChain{Chain: deps.Chain}
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		loop := rotatevaultkeys.NewLoop(rotatevaultkeys.LoopDeps{Deps: deps, Health: c.NewHealth(), IsIdle: nil}, c.Clock)
		if err := loop.Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()

	first := c.Clock.AwaitWait(t)
	c.Clock.Advance(first)
	second := c.Clock.AwaitWait(t)

	if first != 10*time.Second || second != 20*time.Second {
		t.Errorf("waits = %v, %v, want 10s and 20s", first, second)
	}
	if !c.HasItemKey("rotate-vault-keys-failed") {
		t.Error("the failed round is not shown")
	}
}

func TestLoop_WaitsAMinuteBetweenRoundsWhileTheApplicationIsIdle(t *testing.T) {
	t.Parallel()
	c := newComputer(t)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		loop := rotatevaultkeys.NewLoop(rotatevaultkeys.LoopDeps{Deps: c.RotateDeps(), Health: c.NewHealth(), IsIdle: func() bool { return true }}, c.Clock)
		if err := loop.Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()

	if waited := c.Clock.AwaitWait(t); waited != time.Minute {
		t.Errorf("waited %v, want 1m", waited)
	}
}
