package session_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

type setup struct {
	protocol    uint32
	signAs      string
	noIdentity  bool
	isOtherName bool
	isDown      bool
	serverSkew  time.Duration
	leftItem    string
	isIdle      bool
}

type kit struct {
	clock    *manualClock
	identity *accounttest.Fake
	server   *synctest.Server
	keep     *session.Keep
	health   *healthtest.Host
	done     chan error
	cancel   context.CancelFunc
}

func newKit(t *testing.T, s setup) *kit {
	t.Helper()
	k := &kit{clock: newManualClock(synctest.GetNow().Add(-s.serverSkew)), done: make(chan error, 1)}
	if s.signAs == "" {
		s.signAs = syncprotocol.SignInPluginID
	}
	k.server = synctest.NewServer(t, synctest.NewClock(synctest.GetNow()))
	if s.protocol != 0 {
		k.server.SetProtocol(s.protocol)
	}
	address := k.server.URL
	if s.isOtherName {
		address = strings.Replace(address, "127.0.0.1", "localhost", 1)
	}
	k.server.SetDown(s.isDown)

	k.identity = accounttest.NewFake(t, s.signAs)
	if s.noIdentity {
		k.identity.RemoveIdentity()
	}
	k.health = healthtest.NewHost(t)
	identityEndpoint, err := hostcall.ReadVars(k.identity.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	healthEndpoint, err := hostcall.ReadVars(k.health.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	if s.leftItem != "" {
		left := health.Item{Key: s.leftItem, Severity: health.Warning, Text: "Could not reach the sync server"}
		if err := health.NewClient(healthEndpoint).Report(t.Context(), left); err != nil {
			t.Fatal(err)
		}
	}
	keep := session.NewKeep(session.Deps{
		Identity: accountadapter.New(account.NewClient(identityEndpoint)),
		Server:   serveradapter.New(serveradapter.Config{Address: address, HTTP: k.server.NewHTTP(), Now: k.clock.Now}),
		Health:   health.NewClient(healthEndpoint),
		Clock:    k.clock,
		IsIdle:   func() bool { return s.isIdle },
	})
	k.keep = keep
	ctx, cancel := context.WithCancel(t.Context())
	k.cancel = cancel
	t.Cleanup(cancel)
	go func() { k.done <- keep.Execute(ctx) }()
	return k
}

func eventually(t *testing.T, what string, condition func() bool) {
	t.Helper()
	wall := clock.System{}
	deadline := wall.Now().Add(5 * time.Second)
	for !condition() {
		if wall.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (k *kit) item(key string) (health.Item, bool) {
	for _, item := range k.health.List() {
		if item.Key == key {
			return item, true
		}
	}
	return health.Item{}, false
}

func (k *kit) waitItem(t *testing.T, key string) health.Item {
	t.Helper()
	var found health.Item
	eventually(t, "the health item "+key, func() bool {
		item, ok := k.item(key)
		found = item
		return ok
	})
	return found
}

func TestKeep_SignsInWithTheIdentityOfTheDevice(t *testing.T) {
	k := newKit(t, setup{})

	eventually(t, "a session", func() bool { return k.keep.GetToken() != "" })

	fingerprint, isLive := k.server.ReadSession(t, k.keep.GetToken())
	if public := k.identity.Public(); !isLive || fingerprint != public.AccountFingerprint {
		t.Errorf("session of the account %q (live: %v), want a live session of the account of the identity", fingerprint, isLive)
	}
	if k.identity.SignCount() != 1 {
		t.Errorf("signatures made = %d, want 1", k.identity.SignCount())
	}
	if items := k.health.List(); len(items) != 0 {
		t.Errorf("health items = %+v, want none after a sign-in", items)
	}
}

func TestKeep_WithoutAnIdentityRaisesNoIdentityAndNeverCallsTheServer(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})

	if waited := k.clock.awaitWait(t); waited != 5*time.Second {
		t.Errorf("waited %v, want 5s", waited)
	}

	item := k.waitItem(t, "no-identity")
	if item.Text != "No identity on this device" {
		t.Errorf("text = %q", item.Text)
	}
	if k.server.CountRequests() != 0 || k.identity.SignCount() != 0 {
		t.Errorf("server requests = %d, signatures = %d, want none", k.server.CountRequests(), k.identity.SignCount())
	}
}

func TestKeep_AsksForTheIdentityAgainAfterFiveSecondsAndDoublesTheWaitAtEachFailure(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})
	first := k.clock.awaitWait(t)

	k.clock.Advance(first)
	second := k.clock.awaitWait(t)
	k.clock.Advance(second)
	third := k.clock.awaitWait(t)

	if first != 5*time.Second || second != 10*time.Second || third != 20*time.Second {
		t.Errorf("waits = %v, %v, %v, want 5s, 10s and 20s", first, second, third)
	}
}

func TestKeep_WaitsAtLeastTwoMinutesAfterAFailureWhileTheApplicationIsIdle(t *testing.T) {
	k := newKit(t, setup{noIdentity: true, isIdle: true})

	if waited := k.clock.awaitWait(t); waited != 2*time.Minute {
		t.Errorf("waited %v, want 2m", waited)
	}
}

func TestKeep_SignsInOnceTheIdentityIsCreatedAndClearsTheItem(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})
	k.clock.awaitWait(t)
	k.waitItem(t, "no-identity")

	k.identity.CreateIdentity()
	k.clock.Advance(5 * time.Second)

	eventually(t, "a session", func() bool { return k.keep.GetToken() != "" })
	eventually(t, "the item to clear", func() bool { _, ok := k.item("no-identity"); return !ok })
}

func TestKeep_AsksToUpdateWhenTheServerNeedsAnotherProtocolAndTriesAgainAfter30SecondsWithBackoff(t *testing.T) {
	k := newKit(t, setup{protocol: syncprotocol.Version + 1})

	first := k.clock.awaitWait(t)
	k.clock.Advance(30 * time.Second)
	second := k.clock.awaitWait(t)

	item := k.waitItem(t, "update-required")
	if item.Text != "Update Numen" {
		t.Errorf("text = %q", item.Text)
	}
	if first != 30*time.Second || second != time.Minute {
		t.Errorf("waits = %v, %v, want 30s and 1m", first, second)
	}
	if opened := k.server.CountCalls(syncv1connect.SignInServiceSignInProcedure); opened != 0 || k.identity.SignCount() != 0 {
		t.Errorf("sign-ins = %d, signatures = %d, want none: a mismatch is found before anything is signed", opened, k.identity.SignCount())
	}
}

func TestKeep_RaisesServerUnreachableWhenTheServerCannotBeReachedAndTriesAgainAfter30SecondsWithBackoff(t *testing.T) {
	k := newKit(t, setup{isDown: true})

	first := k.clock.awaitWait(t)
	k.clock.Advance(30 * time.Second)
	second := k.clock.awaitWait(t)

	item := k.waitItem(t, "server-unreachable")
	if item.Text != "Could not reach the sync server" {
		t.Errorf("text = %q", item.Text)
	}
	if first != 30*time.Second || second != time.Minute {
		t.Errorf("waits = %v, %v, want 30s and 1m", first, second)
	}
}

func TestKeep_RefusesToSignForAServerThatNamesAnotherAudience(t *testing.T) {
	k := newKit(t, setup{isOtherName: true})

	waited := k.clock.awaitWait(t)

	item := k.waitItem(t, "sign-in-failed")
	if item.Text != "Could not sign in" || waited != 30*time.Second {
		t.Errorf("text = %q, waited %v, want the item and 30s", item.Text, waited)
	}
	if k.identity.SignCount() != 0 {
		t.Errorf("signatures made = %d, want none for another audience", k.identity.SignCount())
	}
}

func TestKeep_RefusesToSignAChallengeThatExpiresMoreThan120SecondsFromItsClock(t *testing.T) {
	for name, skew := range map[string]time.Duration{"ahead": 4 * time.Minute, "behind": -4 * time.Minute} {
		k := newKit(t, setup{serverSkew: skew})

		waited := k.clock.awaitWait(t)

		item := k.waitItem(t, "sign-in-failed")
		if item.Text != "Could not sign in" || waited != 30*time.Second {
			t.Errorf("%s: text = %q, waited %v, want the item and 30s", name, item.Text, waited)
		}
		if k.identity.SignCount() != 0 {
			t.Errorf("%s: signatures made = %d, want none", name, k.identity.SignCount())
		}
	}
}

func TestKeep_RaisesSignInFailedWhenTheServerRefusesTheSignatureAndDoesNotRepeatAtOnce(t *testing.T) {
	k := newKit(t, setup{signAs: "md.numen.tools.link"})

	waited := k.clock.awaitWait(t)

	item := k.waitItem(t, "sign-in-failed")
	if item.Text != "Could not sign in" || waited != 30*time.Second {
		t.Errorf("text = %q, waited %v, want the item and 30s", item.Text, waited)
	}
	if got := k.server.CountRequests(); got != 2 {
		t.Errorf("server requests = %d, want a challenge and one sign-in, with no hot loop", got)
	}
}

func TestKeep_WaitsUntil60SecondsBeforeTheSessionExpiresToSignInAgain(t *testing.T) {
	k := newKit(t, setup{})
	eventually(t, "a session", func() bool { return k.keep.GetToken() != "" })

	waited := k.clock.awaitWait(t)

	if want := time.Hour - 60*time.Second; waited > want || waited < want-10*time.Second {
		t.Errorf("waited %v, want 59m less the seconds the sign-in took", waited)
	}
}

func TestKeep_SignsInAgainWhenTheServerEndedTheSession(t *testing.T) {
	k := newKit(t, setup{})
	eventually(t, "a session", func() bool { return k.keep.GetToken() != "" })
	ended := k.keep.GetToken()
	k.server.EndSessions()

	if err := k.keep.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}

	if _, isLive := k.server.ReadSession(t, ended); isLive {
		t.Error("the session the server ended is live")
	}
	if _, isLive := k.server.ReadSession(t, k.keep.GetToken()); !isLive || k.keep.GetToken() == ended {
		t.Error("the keep holds no new live session")
	}
}

func TestKeep_StopsWhenItsContextEnds(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})
	k.clock.awaitWait(t)

	k.cancel()

	select {
	case err := <-k.done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Execute did not return after its context ended")
	}
}

func TestKeep_DeletingAccountRaisesItemAndNeverRestoresByItself(t *testing.T) {
	p := newDeletingPair(t)
	keep := session.NewKeep(session.Deps{
		Identity: p.second.Account, Server: p.second.Sessions, Health: p.second.NewHealth(), Clock: p.second.Clock,
	})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() { done <- keep.Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Execute returned %v after its context ended, want context.Canceled", err)
		}
	})

	eventually(t, "the item that says the account is being deleted", func() bool { return p.second.HasItemKey("account-deleting") })
	if _, isDeleting := keep.GetDeletion(); !isDeleting {
		t.Error("the sign-in does not know the account is deleting")
	}
	if keep.GetToken() != "" {
		t.Error("the sign-in holds a token of a deleting account")
	}
	if err := p.second.SignIn(t.Context()); err == nil {
		t.Error("another sign-in without a restore opened a session of a deleting account")
	}
}

func TestKeep_RestoreCancelsTheDeletionAndClearsTheItem(t *testing.T) {
	p := newDeletingPair(t)
	keep := session.NewKeep(session.Deps{
		Identity: p.second.Account, Server: p.second.Sessions, Health: p.second.NewHealth(), Clock: p.second.Clock,
	})
	if err := keep.Refresh(t.Context()); err == nil {
		t.Fatal("sign-in to a deleting account opened a session")
	}

	if err := keep.Restore(t.Context()); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if _, isDeleting := keep.GetDeletion(); isDeleting || keep.GetToken() == "" {
		t.Errorf("deleting = %v, token empty = %v, want a signed-in account that is not deleting", isDeleting, keep.GetToken() == "")
	}
}

func TestKeep_ClearsAnItemAnEarlierRunLeftOnceTheSignInWorks(t *testing.T) {
	k := newKit(t, setup{leftItem: "server-unreachable"})

	eventually(t, "a session", func() bool { return k.keep.GetToken() != "" })
	eventually(t, "the left item to clear", func() bool { _, ok := k.item("server-unreachable"); return !ok })
}
