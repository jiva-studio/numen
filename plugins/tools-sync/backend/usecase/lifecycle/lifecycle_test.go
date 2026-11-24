package lifecycle_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
)

const interval = 30 * time.Second

func newLifecycle(w *synctest.PhoneWorld, s *synctest.Stream) *lifecycle.Lifecycle {
	return lifecycle.NewLifecycle(w.Phone.LifecycleDeps(interval, s, make(chan domain.VaultID)))
}

func TestResume_RunsAPassAtOnceAndOpensTheStream(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	w.Computer.WriteNote(synctest.ComputerVault, "Shopping list.md", "milk")
	w.Clock.Advance(time.Minute)
	w.ComputerSync(t)
	resume := newLifecycle(w, s)

	if err := resume.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := resume.Suspend(t.Context()); err != nil {
			t.Error(err)
		}
	})

	synctest.Eventually(t, "the note on the phone", func() bool {
		text, ok := w.Phone.ReadNote(synctest.PhoneVault, "Shopping list.md")
		return ok && text == "milk"
	})
	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })
}

func TestResume_IsIdempotent(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	for range 2 {
		if err := life.Resume(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if err := life.Suspend(t.Context()); err != nil {
			t.Error(err)
		}
	})

	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() >= 1 })
	if got := s.CountOpened(); got != 1 {
		t.Errorf("%d streams were opened by two calls, want 1", got)
	}
}

func TestSuspend_ClosesTheStreamAndAsksForNothingMore(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })

	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	if open := s.CountOpen(); open != 0 {
		t.Errorf("%d streams are open after Suspend, want 0", open)
	}
	asked := w.Server.CountRequests()
	w.Clock.Advance(2 * interval)
	if got := w.Server.CountRequests(); got != asked {
		t.Errorf("the server took %d requests after Suspend, want none in two intervals", got-asked)
	}
}

func TestSuspend_IsIdempotent(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	life := newLifecycle(w, &synctest.Stream{})
	for range 2 {
		if err := life.Suspend(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResume_OpensTheStreamAgainAfterSuspend(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := life.Suspend(t.Context()); err != nil {
			t.Error(err)
		}
	})

	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })
}

func TestSuspend_SendsTheNoteCreatedOnThePhoneBeforeItStops(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	life := newLifecycle(w, &synctest.Stream{})
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	w.Phone.WriteNote(synctest.PhoneVault, "Shopping list.md", "bread")
	w.Clock.Advance(time.Minute)

	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	w.ComputerSync(t)
	if text, ok := w.Computer.ReadNote(synctest.ComputerVault, "Shopping list.md"); !ok || text != "bread" {
		t.Errorf("note on the computer = %q (there: %v), want bread", text, ok)
	}
}

// slowPass is a pass that does not end until the context does: a connection that hangs.
type slowPass struct{ started chan struct{} }

func (p slowPass) Execute(ctx context.Context, _ domain.VaultID) error {
	select {
	case p.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestSuspend_StopsAfterTheGraceWhenThePassDoesNotEnd(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	deps := w.Phone.LifecycleDeps(interval, &synctest.Stream{}, make(chan domain.VaultID))
	pass := slowPass{started: make(chan struct{}, 1)}
	deps.Pass = pass
	life := lifecycle.NewLifecycle(deps)
	w.Phone.WriteNote(synctest.PhoneVault, "Shopping list.md", "bread")
	w.Clock.Advance(time.Minute)
	ended := make(chan error, 1)

	go func() { ended <- life.Suspend(t.Context()) }()
	<-pass.started
	if waited := w.Clock.AwaitWait(t); waited != lifecycle.Grace {
		t.Fatalf("Suspend waits %v for the pass, want %v", waited, lifecycle.Grace)
	}
	w.Clock.Advance(lifecycle.Grace)

	if err := <-ended; err != nil {
		t.Errorf("Suspend returned %v after the grace, want it to stop without an error", err)
	}
}

func TestResume_SendsTheNoteCreatedWhileTheApplicationWasClosed(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Phone.WriteNote(synctest.PhoneVault, "Shopping list.md", "eggs")
	w.Clock.Advance(time.Minute)
	life := newLifecycle(w, &synctest.Stream{})
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := life.Suspend(t.Context()); err != nil {
			t.Error(err)
		}
	})

	synctest.Eventually(t, "the note on the computer", func() bool {
		w.ComputerSync(t)
		text, ok := w.Computer.ReadNote(synctest.ComputerVault, "Shopping list.md")
		return ok && text == "eggs"
	})
}
