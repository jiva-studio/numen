package rotatevaultkeys_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
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
