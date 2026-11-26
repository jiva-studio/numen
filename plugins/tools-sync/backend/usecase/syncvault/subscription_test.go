package syncvault_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const (
	keyNeeded     = "subscription-needed"
	graceTextHead = "Subscription ended: changes are sent until "
)

func countSubscriptionReads(p *pair) int {
	return p.server.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure)
}

func TestRunSync_WarnsThatChangesAreSentUntilTheEndOfTheGraceAndStillSendsTheNote(t *testing.T) {
	p := newGracePair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	writesEnd := p.a.Clock.Now().Add(6 * 24 * time.Hour).Format("2 Jan 2006")
	synctest.WaitUntil(t, "the grace warning", func() bool { return p.a.HasItem(graceTextHead + writesEnd) })
	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	if !p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want the warning under %s", p.a.Health.List(), keyNeeded)
	}
}

func TestRunSync_RaisesNoWarningForAnActiveSubscription(t *testing.T) {
	p := newPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	p.a.Clock.AwaitWait(t)
	if p.a.HasItemKey(keyNeeded) {
		t.Errorf("health items = %+v, want no subscription warning", p.a.Health.List())
	}
}

func TestRunSync_ReadsTheSubscriptionOncePerRoundWhateverTheNumberOfVaults(t *testing.T) {
	p := newPair(t)
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
	server := synctest.NewServer(t, clock)
	c := synctest.NewFirstComputer(t, server, clock, string(source), synctest.NewWords(t))
	before := server.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure)

	startRun(t, c, nil)
	c.Clock.AwaitWait(t)

	if got := server.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure) - before; got != 0 {
		t.Errorf("the server ran %d subscription reads, want none", got)
	}
}

func newGracePair(t *testing.T) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	a := synctest.NewUnsubscribedComputer(t, server, clock, string(source), words)
	a.SetPhase(t, "grace")
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &pair{server: server, a: a, b: b}
}
