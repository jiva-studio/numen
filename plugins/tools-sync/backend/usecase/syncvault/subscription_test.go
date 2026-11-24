package syncvault_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const (
	textNeeded    = "Sync needs a subscription"
	textReadOnly  = "Subscription ended: changes stay on this device"
	keyNeeded     = "subscription-needed"
	graceTextHead = "Subscription ended: changes are sent until "
)

func setPhase(t *testing.T, server *synctest.Server, phase string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/admin/subscription?phase="+phase, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.StatusCode >= 300 {
		t.Fatalf("the admin path answered %d", response.StatusCode)
	}
}

func newSubscribedPair(t *testing.T) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{NeedsSubscription: true})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	setPhase(t, server, "active")
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &pair{server: server, a: a, b: b}
}

func countSubscriptionReads(p *pair) int {
	return p.server.Fake.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure)
}

func TestSyncVault_WarnsThatSyncNeedsASubscriptionWhenTheAccountHasNone(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "none")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)

	if !p.a.HasItem(textNeeded) || !p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want %q under %s", p.a.Health.List(), textNeeded, keyNeeded)
	}
}

func TestSyncVault_WarnsThatChangesStayOnThisDeviceInReadOnlyAndSendsNothing(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "read_only")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)

	if !p.a.HasItem(textReadOnly) || !p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want %q under %s", p.a.Health.List(), textReadOnly, keyNeeded)
	}
	if got := len(p.a.ReadChanges(t, p.a.GetRemoteID(t, source))); got != 0 {
		t.Errorf("server holds %d revisions, want none", got)
	}
}

func TestSyncVault_WarnsThatChangesStayOnThisDeviceAfterTheCloudDeletedTheVaults(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "deleted")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)

	if !p.a.HasItem(textReadOnly) {
		t.Errorf("health items = %+v, want %q", p.a.Health.List(), textReadOnly)
	}
}

func TestSyncVault_KeepsTheTextThatSyncNeedsASubscriptionWhenTheSubscriptionCannotBeRead(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "read_only")
	p.a.WriteNote(source, "Shopping list.md", "milk")
	p.server.LoseNextResponse("/GetSubscription")

	sync(t, p.a, source)

	if !p.a.HasItem(textNeeded) || p.a.HasItem(textReadOnly) {
		t.Errorf("health items = %+v, want only %q", p.a.Health.List(), textNeeded)
	}
}

func TestSyncVault_ClearsTheSubscriptionWarningAndSendsTheNoteAfterTheSubscriptionIsBoughtAgain(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "read_only")
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)

	setPhase(t, p.server, "active")
	sync(t, p.a, source)
	sync(t, p.b, target)

	if p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want the warning cleared", p.a.Health.List())
	}
	wantNote(t, p.b, target, "Shopping list.md", "milk")
}

func TestRunSync_WarnsThatChangesAreSentUntilTheEndOfTheGraceAndStillSendsTheNote(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "grace")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	writesEnd := synctest.GetStart().Add(6 * 24 * time.Hour).Format("2 Jan 2006")
	synctest.WaitUntil(t, "the grace warning", func() bool { return p.a.HasItem(graceTextHead + writesEnd) })
	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	if !p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want the warning under %s", p.a.Health.List(), keyNeeded)
	}
}

func TestRunSync_ClearsTheGraceWarningWhenTheSubscriptionIsActiveAgain(t *testing.T) {
	p := newSubscribedPair(t)
	setPhase(t, p.server, "grace")
	startRun(t, p.a, nil)
	synctest.WaitUntil(t, "the grace warning", func() bool { return p.a.HasItemKey(keyNeeded) })
	p.a.Clock.AwaitWait(t)

	setPhase(t, p.server, "active")
	p.a.Clock.Advance(interval)

	synctest.WaitUntil(t, "the warning cleared", func() bool { return !p.a.HasItemKey(keyNeeded) })
}

func TestRunSync_RaisesNoWarningForAnActiveSubscription(t *testing.T) {
	p := newSubscribedPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	p.a.Clock.AwaitWait(t)
	if p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want no subscription warning", p.a.Health.List())
	}
}

func TestRunSync_ReadsTheSubscriptionOncePerRoundWhateverTheNumberOfVaults(t *testing.T) {
	p := newSubscribedPair(t)
	p.a.AddVault(t, "Second")
	p.a.Connect(t, "Second")
	before := countSubscriptionReads(p)

	startRun(t, p.a, nil)
	p.a.Clock.AwaitWait(t)

	if got := countSubscriptionReads(p) - before; got != 1 {
		t.Errorf("the server ran %d subscription reads in the first round, want 1", got)
	}
}

func TestRunSync_ReadsNoSubscriptionWhenNoVaultIsConnected(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{NeedsSubscription: true})
	c := synctest.NewFirstComputer(t, server, clock, string(source), synctest.NewWords(t))

	startRun(t, c, nil)
	c.Clock.AwaitWait(t)

	if got := server.Fake.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure); got != 0 {
		t.Errorf("the server ran %d subscription reads, want none", got)
	}
}
