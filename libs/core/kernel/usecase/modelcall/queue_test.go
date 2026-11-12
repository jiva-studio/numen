package modelcall_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

const (
	otherConsumerID = "md.example.second"
	addressLimited  = "https://limited.test/v1"
)

type queueSetup struct {
	world *modelcalltest.World
	calls *modelcall.Calls
	entry modelcalltest.Entry
}

func newQueueSetup(t *testing.T, parallelRequests int) *queueSetup {
	t.Helper()
	w := modelcalltest.New()
	entry := w.AddLimited(t, "Limited", addressLimited, parallelRequests)
	for _, plugin := range []string{consumerID, otherConsumerID} {
		w.Rows.Set(plugin, modelcalltest.CompleteOperation, modelcalltest.Row(entry, "m"))
		w.Rows.Set(plugin, modelcalltest.EmbedOperation, modelcalltest.Row(entry, "m"))
	}
	return &queueSetup{
		world: w, entry: entry,
		calls: modelcall.NewCalls(modelcall.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}

func (s *queueSetup) complete(ctx context.Context, plugin, text string, priority modelcalldomain.Priority) <-chan error {
	failed := make(chan error, 1)
	go func() {
		_, err := s.calls.Call(ctx, plugin, modelcalldomain.Request{
			Operation: modelcalltest.CompleteOperation, Payload: []byte(text), Priority: priority,
		}, func([]byte) error { return nil })
		failed <- err
	}()
	return failed
}

func (s *queueSetup) embed(ctx context.Context, plugin string, priority modelcalldomain.Priority) <-chan error {
	failed := make(chan error, 1)
	go func() {
		_, err := s.calls.Call(ctx, plugin, modelcalldomain.Request{
			Operation: modelcalltest.EmbedOperation, Payload: []byte("text"), Priority: priority,
		}, func([]byte) error { return nil })
		failed <- err
	}()
	return failed
}

func (s *queueSetup) awaitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-s.world.Provider.Entered():
	case <-time.After(10 * time.Second):
		t.Fatal("no call reached the provider")
	}
}

func (s *queueSetup) texts() []string {
	var texts []string
	for _, one := range s.world.Provider.Calls() {
		if one.Operation == modelcalltest.CompleteOperation {
			texts = append(texts, string(one.Payload))
		}
	}
	return texts
}

func awaitDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("the call did not return")
		return nil
	}
}

func TestASecondCallWaitsWhileTheConnectionRunsItsLimitOfOne(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	first := s.complete(t.Context(), consumerID, "first", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)

	second := s.complete(t.Context(), consumerID, "second", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)

	if got := s.texts(); len(got) != 1 {
		t.Fatalf("provider received %v while the first call ran", got)
	}
	s.world.Provider.Release(addressLimited)
	if err := awaitDone(t, first); err != nil {
		t.Fatal(err)
	}
	s.awaitEntered(t)
	s.world.Provider.Release(addressLimited)
	if err := awaitDone(t, second); err != nil {
		t.Fatal(err)
	}
	if got := s.texts(); len(got) != 2 || got[1] != "second" {
		t.Errorf("provider received %v", got)
	}
}

func TestTwoPluginsOfOneConnectionShareItsLimit(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	first := s.complete(t.Context(), consumerID, "from first plugin", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)

	second := s.complete(t.Context(), otherConsumerID, "from second plugin", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)

	if got := s.texts(); len(got) != 1 {
		t.Errorf("provider received %v while another plugin held the only slot", got)
	}
	s.world.Provider.Unblock(addressLimited)
	if err := errors.Join(awaitDone(t, first), awaitDone(t, second)); err != nil {
		t.Fatal(err)
	}
}

func TestTheStoredParallelRequestsSetTheLimit(t *testing.T) {
	s := newQueueSetup(t, 3)
	s.world.Provider.Block(addressLimited)
	var calls []<-chan error
	for _, text := range []string{"a", "b", "c"} {
		calls = append(calls, s.complete(t.Context(), consumerID, text, modelcalldomain.PriorityBackground))
		s.awaitEntered(t)
	}

	calls = append(calls, s.complete(t.Context(), consumerID, "d", modelcalldomain.PriorityBackground))
	s.world.Waits.AwaitCount(t, 1)

	if got := s.texts(); len(got) != 3 {
		t.Errorf("provider received %v, want three calls at once", got)
	}
	s.world.Provider.Unblock(addressLimited)
	for _, one := range calls {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheDefaultOfTheTypeSetsTheLimitWhereNothingIsStored(t *testing.T) {
	s := newQueueSetup(t, 0)
	s.world.Provider.Block(addressLimited)
	var calls []<-chan error
	for _, text := range []string{"a", "b"} {
		calls = append(calls, s.complete(t.Context(), consumerID, text, modelcalldomain.PriorityBackground))
		s.awaitEntered(t)
	}

	calls = append(calls, s.complete(t.Context(), consumerID, "c", modelcalldomain.PriorityBackground))
	s.world.Waits.AwaitCount(t, 1)

	if got := s.texts(); len(got) != 2 {
		t.Errorf("provider received %v, want the default of two calls at once", got)
	}
	s.world.Provider.Unblock(addressLimited)
	for _, one := range calls {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
}

func TestACallOfAConnectionWithNoLimitIsNeverQueued(t *testing.T) {
	w := modelcalltest.New()
	entry := w.AddServer(t, "First", addressOne, keyOne)
	w.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(entry, "m"))
	s := &queueSetup{world: w, calls: modelcall.NewCalls(modelcall.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside)}
	w.Provider.Block(addressOne)
	var calls []<-chan error
	for _, text := range []string{"a", "b", "c", "d", "e"} {
		calls = append(calls, s.complete(t.Context(), consumerID, text, modelcalldomain.PriorityBackground))
		s.awaitEntered(t)
	}

	if got := w.Waits.List(); len(got) != 0 {
		t.Errorf("waits = %v, want none", got)
	}
	w.Provider.Unblock(addressOne)
	for _, one := range calls {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnInteractiveCallGoesBeforeABackgroundCallThatWaitedLonger(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	running := s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)
	background := s.complete(t.Context(), consumerID, "background", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)
	interactive := s.complete(t.Context(), consumerID, "interactive", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 2)

	s.world.Provider.Release(addressLimited)
	s.awaitEntered(t)
	s.world.Provider.Release(addressLimited)
	s.awaitEntered(t)
	s.world.Provider.Unblock(addressLimited)

	for _, one := range []<-chan error{running, interactive, background} {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"running", "interactive", "background"}
	if got := s.texts(); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("provider received %v, want %v", got, want)
	}
}

func TestCallsOfOnePriorityGoInTheOrderTheyWaited(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	running := s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityInteractive)
	s.awaitEntered(t)
	second := s.complete(t.Context(), consumerID, "second", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 1)
	third := s.complete(t.Context(), consumerID, "third", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 2)

	s.world.Provider.Release(addressLimited)
	s.awaitEntered(t)
	s.world.Provider.Unblock(addressLimited)

	for _, one := range []<-chan error{running, second, third} {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.texts(); len(got) != 3 || got[1] != "second" || got[2] != "third" {
		t.Errorf("provider received %v", got)
	}
}

func TestAWaitingCallTimesOutAfterOneMinuteWhenInteractiveAndTenMinutesWhenBackground(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)

	s.complete(t.Context(), consumerID, "background", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)
	s.complete(t.Context(), consumerID, "interactive", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 2)

	got := s.world.Waits.List()
	if len(got) != 2 || got[0] != 10*time.Minute || got[1] != time.Minute {
		t.Errorf("waits = %v, want 10m0s for the background call and 1m0s for the interactive call", got)
	}
}

func TestAnEmbedCallWaitsAsLongAsItsPriorityAllows(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	s.embed(t.Context(), consumerID, modelcalldomain.PriorityBackground)
	s.awaitEntered(t)

	s.embed(t.Context(), consumerID, modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 1)
	s.embed(t.Context(), consumerID, modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 2)

	got := s.world.Waits.List()
	if len(got) != 2 || got[0] != time.Minute || got[1] != 10*time.Minute {
		t.Errorf("waits = %v, want 1m0s then 10m0s", got)
	}
}

func TestATimedOutWaitIsSkippedAsBusyAndTheNextRowServes(t *testing.T) {
	s := newQueueSetup(t, 1)
	second := s.world.AddServer(t, "Second", addressTwo, keyTwo)
	s.world.Rows.Set(otherConsumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.entry, "m"), modelcalltest.Row(second, "m"))
	s.world.Provider.Block(addressLimited)
	s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)
	queued := s.complete(t.Context(), otherConsumerID, "waiting", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 1)

	s.world.Waits.Elapse(time.Minute)

	if err := awaitDone(t, queued); err != nil {
		t.Fatal(err)
	}
	if got := len(s.world.Provider.CallsTo(addressTwo)); got != 1 {
		t.Errorf("the second row was called %d times, want once", got)
	}
	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v, want none for a wait that timed out", got)
	}
}

func TestATimedOutWaitSetsNoCooldown(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	running := s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)
	queued := s.complete(t.Context(), consumerID, "waiting", modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 1)
	s.world.Waits.Elapse(time.Minute)

	var all *connection.AllFailedError
	if err := awaitDone(t, queued); !errors.As(err, &all) || len(all.Failures) != 1 ||
		all.Failures[0].ConnectionName != "Limited" || all.Failures[0].Reason != "busy" {
		t.Fatalf("err = %v, want Limited: busy", err)
	}
	s.world.Provider.Unblock(addressLimited)
	if err := awaitDone(t, running); err != nil {
		t.Fatal(err)
	}
	if err := awaitDone(t, s.complete(t.Context(), consumerID, "again", modelcalldomain.PriorityInteractive)); err != nil {
		t.Errorf("the connection was set aside after a wait that timed out: %v", err)
	}
}

func TestACanceledQueuedCallIsRemovedFromTheQueueAndFreesNoSlot(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	running := s.complete(t.Context(), consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)
	ctx, cancel := context.WithCancel(t.Context())
	queued := s.complete(ctx, consumerID, "queued", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)

	cancel()

	if err := awaitDone(t, queued); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	s.world.Waits.AwaitCount(t, 0)
	next := s.complete(t.Context(), consumerID, "next", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)
	if got := s.texts(); len(got) != 1 {
		t.Errorf("provider received %v, the canceled call freed a slot it did not hold", got)
	}
	s.world.Provider.Unblock(addressLimited)
	for _, one := range []<-chan error{running, next} {
		if err := awaitDone(t, one); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.texts(); len(got) != 2 || got[1] != "next" {
		t.Errorf("provider received %v, want running then next", got)
	}
}

func TestACanceledRunningCallFreesItsSlotWhenTheProviderReturns(t *testing.T) {
	s := newQueueSetup(t, 1)
	s.world.Provider.Block(addressLimited)
	ctx, cancel := context.WithCancel(t.Context())
	running := s.complete(ctx, consumerID, "running", modelcalldomain.PriorityBackground)
	s.awaitEntered(t)
	next := s.complete(t.Context(), consumerID, "next", modelcalldomain.PriorityBackground)
	s.world.Waits.AwaitCount(t, 1)

	cancel()

	if err := awaitDone(t, running); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	s.awaitEntered(t)
	s.world.Provider.Unblock(addressLimited)
	if err := awaitDone(t, next); err != nil {
		t.Fatal(err)
	}
}
