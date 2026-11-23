package session_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

func start() time.Time {
	return time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
}

type setup struct {
	protocol   uint32
	signAs     string
	noIdentity bool
	audience   string
	isDown     bool
	sessionTTL time.Duration
	serverSkew time.Duration
}

type kit struct {
	clock    *manualClock
	identity *accounttest.Fake
	server   *fakeserver.Server
	health   *healthtest.Host
	requests atomic.Int32
	done     chan error
	cancel   context.CancelFunc
}

func newKit(t *testing.T, s setup) *kit {
	t.Helper()
	k := &kit{clock: newManualClock(start()), done: make(chan error, 1)}
	if s.signAs == "" {
		s.signAs = syncprotocol.SignInPluginID
	}
	k.server = fakeserver.New(fakeserver.Config{Protocol: s.protocol, Now: func() time.Time { return k.clock.Now().Add(s.serverSkew) }, SessionTTL: s.sessionTTL})
	served := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k.requests.Add(1)
		k.server.ServeHTTP(w, r)
	}))
	t.Cleanup(served.Close)
	k.server.SetAudience(served.URL)
	if s.audience != "" {
		k.server.SetAudience(s.audience)
	}
	address := served.URL
	if s.isDown {
		served.Close()
	}

	k.identity = accounttest.NewFake(t, s.signAs)
	if s.noIdentity {
		k.identity.NoIdentity()
	}
	k.health = healthtest.NewHost(t)
	identityEndpoint, err := hostcall.FromVars(k.identity.Vars("token"))
	if err != nil {
		t.Fatal(err)
	}
	healthEndpoint, err := hostcall.FromVars(k.health.Vars("token"))
	if err != nil {
		t.Fatal(err)
	}
	keep := session.NewKeep(session.Deps{
		Identity: accountadapter.New(account.NewClient(identityEndpoint)),
		Server:   serveradapter.New(serveradapter.Config{Address: address, HTTP: http.DefaultClient, Now: k.clock.Now}),
		Health:   health.NewClient(healthEndpoint),
		Clock:    k.clock,
	})
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

	eventually(t, "a session", func() bool { return len(k.server.ListSessions()) == 1 })

	sessions := k.server.ListSessions()
	public := k.identity.Public()
	if string(sessions[0].DeviceSignKey) != string(public.DeviceSignKey) || sessions[0].AccountFingerprint != public.AccountFingerprint {
		t.Errorf("session = %+v, want the device and the account of the identity", sessions[0])
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
	if k.requests.Load() != 0 || k.identity.SignCount() != 0 || len(k.server.ListSessions()) != 0 {
		t.Errorf("server requests = %d, signatures = %d, sessions = %d, want none", k.requests.Load(), k.identity.SignCount(), len(k.server.ListSessions()))
	}
}

func TestKeep_AsksForTheIdentityAgainAfterFiveSecondsAtAFixedInterval(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})
	k.clock.awaitWait(t)

	k.clock.Advance(5 * time.Second)
	second := k.clock.awaitWait(t)
	k.clock.Advance(5 * time.Second)
	third := k.clock.awaitWait(t)

	if second != 5*time.Second || third != 5*time.Second {
		t.Errorf("waits = %v, %v, want 5s each: no backoff", second, third)
	}
}

func TestKeep_SignsInOnceTheIdentityIsCreatedAndClearsTheItem(t *testing.T) {
	k := newKit(t, setup{noIdentity: true})
	k.clock.awaitWait(t)
	k.waitItem(t, "no-identity")

	k.identity.CreateIdentity()
	k.clock.Advance(5 * time.Second)

	eventually(t, "a session", func() bool { return len(k.server.ListSessions()) == 1 })
	eventually(t, "the item to clear", func() bool { _, ok := k.item("no-identity"); return !ok })
}

func TestKeep_AsksToUpdateWhenTheServerNeedsAnotherProtocolAndTriesAgainEvery30Seconds(t *testing.T) {
	k := newKit(t, setup{protocol: syncprotocol.Version + 1})

	first := k.clock.awaitWait(t)
	k.clock.Advance(30 * time.Second)
	second := k.clock.awaitWait(t)

	item := k.waitItem(t, "update-required")
	if item.Text != "Update Numen" {
		t.Errorf("text = %q", item.Text)
	}
	if first != 30*time.Second || second != 30*time.Second {
		t.Errorf("waits = %v, %v, want 30s each", first, second)
	}
	if len(k.server.ListSessions()) != 0 || k.identity.SignCount() != 0 {
		t.Errorf("sessions = %d, signatures = %d, want none: a mismatch is found before anything is signed", len(k.server.ListSessions()), k.identity.SignCount())
	}
}

func TestKeep_RaisesServerUnreachableWhenTheServerCannotBeReachedAndTriesAgainEvery30Seconds(t *testing.T) {
	k := newKit(t, setup{isDown: true})

	first := k.clock.awaitWait(t)
	k.clock.Advance(30 * time.Second)
	second := k.clock.awaitWait(t)

	item := k.waitItem(t, "server-unreachable")
	if item.Text != "Could not reach the sync server" {
		t.Errorf("text = %q", item.Text)
	}
	if first != 30*time.Second || second != 30*time.Second {
		t.Errorf("waits = %v, %v, want 30s each: no backoff", first, second)
	}
}

func TestKeep_RefusesToSignForAServerThatNamesAnotherAudience(t *testing.T) {
	k := newKit(t, setup{audience: "https://evil.example"})

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
	if got := k.requests.Load(); got != 2 {
		t.Errorf("server requests = %d, want a challenge and one sign-in, with no hot loop", got)
	}
}

func TestKeep_SignsInAgain60SecondsBeforeTheSessionExpires(t *testing.T) {
	k := newKit(t, setup{sessionTTL: time.Hour})
	eventually(t, "a session", func() bool { return len(k.server.ListSessions()) == 1 })
	first := k.server.ListSessions()[0]

	waited := k.clock.awaitWait(t)
	k.clock.Advance(waited)

	if waited != time.Hour-60*time.Second {
		t.Errorf("waited %v, want 59m", waited)
	}
	eventually(t, "a second session", func() bool { return len(k.server.ListSessions()) == 2 })
	if k.server.ListSessions()[1].Token == first.Token {
		t.Error("the second sign-in kept the token of the first")
	}
}

func TestKeep_SignsInAgainAtTheRenewalWhenTheServerEndedTheSession(t *testing.T) {
	k := newKit(t, setup{sessionTTL: time.Hour})
	eventually(t, "a session", func() bool { return len(k.server.ListSessions()) == 1 })
	ended := k.server.ListSessions()[0]
	k.server.EndSessions(nil)

	k.clock.Advance(k.clock.awaitWait(t))

	eventually(t, "a new session", func() bool {
		sessions := k.server.ListSessions()
		return len(sessions) == 1 && sessions[0].Token != ended.Token
	})
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
