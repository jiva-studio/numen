package chaincourier_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
)

func newComputer(t *testing.T) (*synctest.Computer, []string) {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	return synctest.NewFirstComputer(t, server, clock, "Test", words), words
}

func parseWords(t *testing.T, words []string) identity.RecoveryKey {
	t.Helper()
	parsed, err := identity.ParseRecoveryWords(words)
	if err != nil {
		t.Fatal(err)
	}
	root, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPublish_DoesNothingWhenCoreHasNoPendingChain(t *testing.T) {
	t.Parallel()
	c, _ := newComputer(t)

	published, err := chaincourier.NewPublish(c.ChainDeps()).Execute(t.Context())

	if err != nil || published {
		t.Errorf("published = %v, error = %v, want nothing published", published, err)
	}
}

func TestAdopt_PassesLongerServerChainToCore(t *testing.T) {
	t.Parallel()
	c, words := newComputer(t)
	own, err := identity.ParseChain(c.Fake.Public().Chain)
	if err != nil {
		t.Fatal(err)
	}
	other, err := identity.NewDeviceKey(bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	longer, err := own.AppendDevice(parseWords(t, words), other.Public(), "Phone", synctest.GetStart())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Cloud.AppendChain(t.Context(), longer.Bytes()); err != nil {
		t.Fatalf("AppendChain() = %v", err)
	}

	adopted, err := chaincourier.NewAdopt(c.ChainDeps()).Execute(t.Context())

	if err != nil || !adopted {
		t.Fatalf("adopted = %v, error = %v, want the longer chain adopted", adopted, err)
	}
	if got := c.Fake.Public().Chain; !bytes.Equal(got, longer.Bytes()) {
		t.Errorf("the core holds %d bytes of chain, want the chain of the server (%d)", len(got), len(longer.Bytes()))
	}
}

func TestAdopt_DoesNothingWhenTheChainsAreTheSameLength(t *testing.T) {
	t.Parallel()
	c, _ := newComputer(t)

	adopted, err := chaincourier.NewAdopt(c.ChainDeps()).Execute(t.Context())

	if err != nil || adopted {
		t.Errorf("adopted = %v, error = %v, want nothing adopted", adopted, err)
	}
}

func newRevokedByFirst(t *testing.T) (observer *synctest.Computer, clock *synctest.Clock) {
	t.Helper()
	clock = synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	third := synctest.NewSecondComputer(t, server, clock, "Tablet", words)
	if _, err := chaincourier.NewAdopt(first.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := first.Fake.RevokeDeviceAt(second.Fake.Public().DeviceSignKey, clock.Now()); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(first.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the revoke published", published, err)
	}
	return third, clock
}

func TestAdopt_RaisesANoticeWhenAnotherDeviceRemovedADevice(t *testing.T) {
	t.Parallel()
	observer, _ := newRevokedByFirst(t)

	if _, err := chaincourier.NewAdopt(observer.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	items := observer.Health.List()
	if len(items) != 1 || items[0].Key != "device-revoked-others" {
		t.Errorf("health items = %+v, want one notice that a device removed another device", items)
	}
}

func TestAdopt_ClearsTheNoticeWhenTheCoolingOffEnds(t *testing.T) {
	t.Parallel()
	observer, clock := newRevokedByFirst(t)
	adopt := chaincourier.NewAdopt(observer.ChainDeps())
	if _, err := adopt.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	clock.Advance(identity.CoolingOff + time.Hour)
	if _, err := adopt.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	if items := observer.Health.List(); len(items) != 0 {
		t.Errorf("health items = %+v, want none after the cooling-off", items)
	}
}
