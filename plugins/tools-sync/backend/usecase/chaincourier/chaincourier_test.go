package chaincourier_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
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

func newThreeRestores(t *testing.T) (earlier, current *synctest.Computer) {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	synctest.NewFirstComputer(t, server, clock, "Test", words)
	earlier = synctest.NewSecondComputer(t, server, clock, "Test", words)
	current = synctest.NewSecondComputer(t, server, clock, "Test", words)
	return earlier, current
}

func TestPublish_AcceptsRevokeAfterEarlierRestoreOfSameName(t *testing.T) {
	t.Parallel()
	earlier, current := newThreeRestores(t)
	target := earlier.GetDeviceSignKey(t)

	if err := current.Fake.RevokeDevice(target); err != nil {
		t.Fatalf("RevokeDevice() = %v", err)
	}
	published, err := chaincourier.NewPublish(current.ChainDeps()).Execute(t.Context())

	if err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the revoke published", published, err)
	}
	stored, err := current.Cloud.GetAccountChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(stored)
	if err != nil {
		t.Fatal(err)
	}
	if chain.IsActiveDevice(target) || len(chain.ListDevices()) != 2 {
		t.Errorf("the server holds %d active devices and the earlier one is active = %v, want 2 and false", len(chain.ListDevices()), chain.IsActiveDevice(target))
	}
}

func TestLoop_ShowsAFailedPublishAndClearsItWhenTheServerIsBack(t *testing.T) {
	t.Parallel()
	earlier, current := newThreeRestores(t)
	if err := current.Fake.RevokeDevice(earlier.GetDeviceSignKey(t)); err != nil {
		t.Fatal(err)
	}
	current.GoOffline()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		if err := chaincourier.NewLoop(current.ChainDeps(), current.Clock).Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()

	current.Clock.AwaitWait(t)
	items := current.Health.List()
	if len(items) != 1 || items[0].Key != "chain-publish-failed" || items[0].Text != "Could not save the change of the devices on the server" || items[0].Detail == "" {
		t.Fatalf("health items = %+v, want the failed publish", items)
	}

	current.GoOnline()
	current.Clock.Advance(2 * time.Second)
	synctest.WaitUntil(t, "the failure is cleared", func() bool { return len(current.Health.List()) == 0 })
}

func TestLoop_ClearsTheFailureAnEarlierRunLeftWhenNothingIsPending(t *testing.T) {
	t.Parallel()
	_, current := newThreeRestores(t)
	stale := health.Item{Key: "chain-publish-failed", Severity: health.Warning, Text: "Could not save the change of the devices on the server"}
	if err := current.ChainDeps().Health.Report(t.Context(), stale); err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		if err := chaincourier.NewLoop(current.ChainDeps(), current.Clock).Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()

	synctest.WaitUntil(t, "the stale failure is cleared", func() bool { return len(current.Health.List()) == 0 })
}

func TestAdopt_RaisesANoticeWhenAnotherDeviceAddedADevice(t *testing.T) {
	t.Parallel()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	observer := synctest.NewSecondComputer(t, server, clock, "Tablet", words)
	if _, err := chaincourier.NewAdopt(first.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Fake.AddDeviceAt(stranger.Public(), "Stranger", clock.Now()); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(first.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the addition published", published, err)
	}

	if _, err := chaincourier.NewAdopt(observer.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	items := observer.Health.List()
	if len(items) != 1 || items[0].Key != "device-revoked-others" || !strings.Contains(items[0].Text, "added another device") {
		t.Errorf("health items = %+v, want one notice that a device added another device", items)
	}
}
