package watchchanges_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/watchchanges"
)

const (
	source      domain.VaultID = "Source"
	target      domain.VaultID = "Target"
	retryWait                  = 3 * time.Second
	heartbeat                  = 2 * time.Second
	silenceWait                = 3 * heartbeat
)

type world struct {
	server *synctest.Server
	a, b   *synctest.Computer
	wakes  *synctest.WakeLog
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{Heartbeat: heartbeat})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &world{server: server, a: a, b: b, wakes: &synctest.WakeLog{}}
}

type running struct {
	cancel context.CancelFunc
	done   chan error
}

func start(t *testing.T, c *synctest.Computer, hints ports.Hints, waker ports.Waker) *running {
	t.Helper()
	deps := c.WatchDeps(waker)
	deps.Hints = hints
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- watchchanges.NewWatchChanges(deps).Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("Execute ended with %v, want context.Canceled", err)
		}
	})
	return &running{cancel: cancel, done: done}
}

func (w *world) streams() int { return w.server.Fake.CountStreams() }

func (w *world) countOpenings() int {
	return w.server.Fake.CountCalls(syncv1connect.WatchServiceWatchChangesProcedure)
}

func (w *world) awaitStream(t *testing.T) {
	t.Helper()
	synctest.Eventually(t, "an open stream", func() bool { return w.streams() == 1 })
}

func (w *world) commitOnA(t *testing.T, text string) {
	t.Helper()
	w.a.WriteNote(source, "Shopping list.md", text)
	if err := w.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
}

func awaitWaitOf(t *testing.T, c *synctest.Clock, want time.Duration) {
	t.Helper()
	for {
		if got := c.AwaitWait(t); got == want {
			return
		}
	}
}

func TestWatchChanges_ACommitOfTheFirstComputerWakesTheVaultOfTheSecond(t *testing.T) {
	w := newWorld(t)
	start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)

	w.commitOnA(t, "milk")

	synctest.Eventually(t, "the wake of the vault", func() bool { return w.wakes.Count(target) >= 1 })
}

func TestWatchChanges_WakesOnlyTheVaultsLinkedToTheRemoteVaultOfTheHint(t *testing.T) {
	w := newWorld(t)
	w.b.AddVault(t, "Plain")
	w.a.AddVault(t, "Elsewhere")
	w.a.Connect(t, "Elsewhere")
	w.b.AddVault(t, "Mirror")
	w.b.ConnectExisting(t, "Mirror", w.a.Link(t, "Elsewhere").RemoteVaultID)
	start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)

	w.commitOnA(t, "milk")
	synctest.Eventually(t, "the first wake", func() bool { return w.wakes.Count(target) >= 1 })
	w.commitOnA(t, "bread")
	synctest.Eventually(t, "the second wake", func() bool { return w.wakes.Count(target) >= 2 })

	if got := w.wakes.Count("Plain") + w.wakes.Count("Mirror"); got != 0 {
		t.Errorf("wakes of a vault without a link or with another remote vault = %d, want none", got)
	}
}

func TestWatchChanges_DoesNotWakeAVaultThatIsAtTheSeqOfTheHint(t *testing.T) {
	w := newWorld(t)
	w.commitOnA(t, "milk")
	if err := w.b.Sync(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)

	w.commitOnA(t, "bread")

	synctest.Eventually(t, "the wake for the new seq", func() bool { return w.wakes.Count(target) >= 1 })
	if got := w.wakes.Count(target); got != 1 {
		t.Errorf("wakes = %d, want one for the commit after the pass, none for the seq already pulled", got)
	}
}

func TestWatchChanges_WakesAfterTheReopenForACommitMadeWhileTheStreamWasDown(t *testing.T) {
	w := newWorld(t)
	start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)
	w.server.Fake.EndStreams()
	awaitWaitOf(t, w.b.Clock, retryWait)
	if got := w.countOpenings(); got != 1 {
		t.Fatalf("opens before the retry wait ended = %d, want 1", got)
	}
	w.commitOnA(t, "milk")

	w.b.Clock.Advance(retryWait)

	synctest.Eventually(t, "the second open", func() bool { return w.countOpenings() == 2 })
	synctest.Eventually(t, "the wake of the catch-up", func() bool { return w.wakes.Count(target) >= 1 })
}

func TestWatchChanges_WaitsTheSameRetryWaitAfterEachFailureOfTheOpen(t *testing.T) {
	w := newWorld(t)
	w.server.SetDown(true)
	start(t, w.b, w.b.Cloud, w.wakes)

	for range 10 {
		if got := w.b.Clock.AwaitWait(t); got != retryWait {
			t.Fatalf("wait = %v, want %v each time with no growth", got, retryWait)
		}
		w.b.Clock.Advance(retryWait)
	}
}

func TestWatchChanges_RaisesNoHealthItemWhileTheStreamFails(t *testing.T) {
	w := newWorld(t)
	w.server.SetDown(true)
	start(t, w.b, w.b.Cloud, w.wakes)

	for range 3 {
		w.b.Clock.AwaitWait(t)
		w.b.Clock.Advance(retryWait)
	}

	if items := w.b.Health.List(); len(items) != 0 {
		t.Errorf("health items = %v, want none", items)
	}
}

func TestWatchChanges_ReopensTheStreamAfterAThreeHeartbeatsOfSilence(t *testing.T) {
	w := newWorld(t)
	relay := synctest.NewRelay(t, w.server.URL)
	start(t, w.b, w.b.NewCloud(relay.URL, w.b.SignIn), w.wakes)
	w.awaitStream(t)
	awaitWaitOf(t, w.b.Clock, silenceWait)
	relay.Pause()

	synctest.Eventually(t, "the second open", func() bool {
		w.b.Clock.Advance(silenceWait)
		return w.countOpenings() == 2
	})
}

func TestWatchChanges_ARealHeartbeatStartsTheSilenceWaitAgain(t *testing.T) {
	w := newWorld(t)
	start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)
	awaitWaitOf(t, w.b.Clock, silenceWait)
	w.b.Clock.Advance(4 * time.Second)
	awaitWaitOf(t, w.b.Clock, silenceWait)

	w.b.Clock.Advance(4 * time.Second)

	awaitWaitOf(t, w.b.Clock, silenceWait)
	if got := w.countOpenings(); got != 1 {
		t.Errorf("opens = %d, want the one stream kept after a heartbeat", got)
	}
}

func TestWatchChanges_SignsInOnceAndOpensAgainWhenTheOpenIsRefusedForWantOfASession(t *testing.T) {
	w := newWorld(t)
	var refreshes atomic.Int32
	hints := w.b.NewCloud(w.server.URL, func(ctx context.Context) error {
		refreshes.Add(1)
		return w.b.SignIn(ctx)
	})
	w.server.Fake.EndSessions(nil)

	start(t, w.b, hints, w.wakes)

	w.awaitStream(t)
	if got := refreshes.Load(); got != 1 {
		t.Errorf("sign-ins = %d, want 1", got)
	}
}

func TestWatchChanges_WaitsTheRetryWaitWhenTheSecondOpenIsRefusedToo(t *testing.T) {
	w := newWorld(t)
	var refreshes atomic.Int32
	hints := w.b.NewCloud(w.server.URL, func(context.Context) error {
		refreshes.Add(1)
		return nil
	})
	w.server.Fake.EndSessions(nil)

	start(t, w.b, hints, w.wakes)

	awaitWaitOf(t, w.b.Clock, retryWait)
	if got := refreshes.Load(); got != 1 {
		t.Errorf("sign-ins before the retry wait = %d, want 1", got)
	}
}

func TestWatchChanges_ReturnsAndClosesTheStreamWhenItsContextEnds(t *testing.T) {
	w := newWorld(t)
	run := start(t, w.b, w.b.Cloud, w.wakes)
	w.awaitStream(t)

	run.cancel()

	if err := <-run.done; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want none or the error of the context", err)
	}
	synctest.Eventually(t, "no open stream", func() bool { return w.streams() == 0 })
}
