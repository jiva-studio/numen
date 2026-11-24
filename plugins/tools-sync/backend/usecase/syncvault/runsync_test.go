package syncvault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

const interval = 30 * time.Second

func startRun(t *testing.T, c *synctest.Computer, changed <-chan domain.VaultID) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- syncvault.NewRunSync(c.RunDeps(interval, changed)).Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("RunSync ended with %v, want the error of its context", err)
		}
	})
}

func hasHeadOf(t *testing.T, c *synctest.Computer, text string) func() bool {
	t.Helper()
	return func() bool {
		heads := c.ReadChanges(t, c.GetRemoteID(t, source))
		if len(heads) != 1 {
			return false
		}
		plain, isFound := c.FindBlob(t, c.GetVaultKey(t, source), heads[0].GetRevision())
		return isFound && string(plain) == text
	}
}

func TestRunSync_RunsAPassOverEveryConnectedVaultAtStart(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
}

func TestRunSync_RunsAPassAfterEachInterval(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	startRun(t, p.a, nil)
	synctest.WaitUntil(t, "the first pass", hasHeadOf(t, p.a, "milk"))
	if waited := p.a.Clock.AwaitWait(t); waited != interval {
		t.Fatalf("wait = %v, want the interval of %v", waited, interval)
	}
	p.a.WriteNote(source, "Shopping list.md", "bread")

	p.a.Clock.Advance(interval)

	synctest.WaitUntil(t, "the second pass", hasHeadOf(t, p.a, "bread"))
}

func TestRunSync_RunsAPassAtOnceWhenFilesOfAConnectedVaultChange(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	changed := make(chan domain.VaultID, 1)
	startRun(t, p.a, changed)
	p.a.Clock.AwaitWait(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")

	changed <- source

	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
}

func TestRunSync_WarnsWhileTheServerIsAwayAndWaitsTheSameIntervalEachTime(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	p.server.SetDown(true)
	startRun(t, p.a, nil)
	synctest.WaitUntil(t, "the warning", func() bool { return p.a.HasItem("Could not reach the sync server") })
	first := p.a.Clock.AwaitWait(t)
	p.a.Clock.Advance(interval)
	second := p.a.Clock.AwaitWait(t)

	p.server.SetDown(false)
	p.a.Clock.Advance(interval)

	if first != interval || second != interval {
		t.Errorf("waits = %v then %v, want %v each time with no backoff", first, second, interval)
	}
	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	synctest.WaitUntil(t, "the warning cleared", func() bool { return !p.a.HasItemKey("server-unreachable") })
}

func TestRunSync_PassesOverNoVaultThatIsNotConnected(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.AddVault(t, "Plain")
	p.a.WriteNote("Plain", "Private.md", "tea")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	startRun(t, p.a, nil)

	synctest.WaitUntil(t, "the note on the server", hasHeadOf(t, p.a, "milk"))
	if p.a.HasItemKey("server-unreachable") {
		t.Errorf("health items = %+v, want none for a vault that is not connected", p.a.Health.List())
	}
	if _, err := p.a.Host.Store("Plain").Read(t.Context(), synctest.LinkFile); err == nil {
		t.Error("link.json exists in a vault that was never connected")
	}
}

func TestRunSync_AChangedVaultDoesNotDelayThePassOverEveryVault(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	changed := make(chan domain.VaultID, 1)
	startRun(t, p.a, changed)
	p.a.Clock.AwaitWait(t)
	p.a.Clock.Advance(10 * time.Second)

	changed <- source

	if waited := p.a.Clock.AwaitWait(t); waited != interval-10*time.Second {
		t.Errorf("wait = %v, want the %v that are left of the interval", waited, interval-10*time.Second)
	}
}
