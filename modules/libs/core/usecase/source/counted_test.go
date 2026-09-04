package source

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// holding is a set of source queries whose first question never answers. It is
// a round that has begun and not ended.
type holding struct {
	port.SourceQueries
	asked chan struct{}
	let   chan struct{}
}

func (h *holding) Fingerprints(
	ctx context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	select {
	case h.asked <- struct{}{}:
	default:
	}
	select {
	case <-h.let:
	case <-ctx.Done():
	}
	// Nothing was read, so the round ends here rather than going on to ask what
	// this set of queries does not answer.
	return nil, errHeld
}

func (h *holding) Recognised(
	ctx context.Context, _ domain.VaultID, _ domain.SourceKind,
) ([]port.SourceText, error) {
	select {
	case h.asked <- struct{}{}:
	default:
	}
	select {
	case <-h.let:
	case <-ctx.Done():
	}
	return nil, errHeld
}

var errHeld = errors.New("the index was not asked")

// TestAQueueRunsBehindTheCallerAndIsCountedBeforeIt. The rounds a vault sets
// itself run behind whoever asked for them, and the count is taken in the call
// and not in the goroutine it counts: a wait beginning the instant Queue
// returns covers the rounds behind it, and a caller that asked for them goes on
// building the window.
func TestAQueueRunsBehindTheCallerAndIsCountedBeforeIt(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	held, v := listens(t, &deaf{})
	known := &holding{asked: make(chan struct{}, 1), let: make(chan struct{})}

	// A call that does not come back is a window that never opens.
	came := make(chan struct{})
	go func() {
		defer close(came)
		held.Queue(ctx, known, time.Hour, v)
	}()
	select {
	case <-came:
	case <-time.After(10 * time.Second):
		t.Fatal("the caller is still in Queue, and the window it was building is not open")
	}

	// The wait covers the round from the moment the call came back, whether or
	// not the goroutine behind it has reached the disk yet.
	waited := make(chan struct{})
	go func() {
		defer close(waited)
		held.Wait()
	}()
	select {
	case <-waited:
		t.Fatal("the wait returned with a round of the queue still to run")
	case <-time.After(200 * time.Millisecond):
	}

	select {
	case <-known.asked:
	case <-time.After(10 * time.Second):
		t.Fatal("no round of the queue ran behind the caller")
	}

	stop()
	close(known.let)
	select {
	case <-waited:
	case <-time.After(10 * time.Second):
		t.Fatal("the wait did not end with the round it was covering")
	}
}

// TestCollectingRunsBehindTheCallerAndIsCountedBeforeIt. The batches left with a
// proofreader are asked after on the same terms as the queue: behind whoever
// asked, and counted in the call rather than in the goroutine it counts.
func TestCollectingRunsBehindTheCallerAndIsCountedBeforeIt(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	w := recognising(t, nil)
	w.Recognising.with.Proofreading = Proofreading{
		Named: true, Batch: 1,
		Queue: func(string) (port.ProofreadQueue, error) { return leaves{}, nil },
	}
	v := domain.Vault{ID: "v", Path: t.TempDir()}
	known := &holding{asked: make(chan struct{}, 1), let: make(chan struct{})}

	came := make(chan struct{})
	go func() {
		defer close(came)
		w.Collecting(ctx, known, time.Hour, v)
	}()
	select {
	case <-came:
	case <-time.After(10 * time.Second):
		t.Fatal("the caller is still in Collecting, and the window it was building is not open")
	}

	waited := make(chan struct{})
	go func() {
		defer close(waited)
		w.Wait()
	}()
	select {
	case <-waited:
		t.Fatal("the wait returned with a round of the collecting still to run")
	case <-time.After(200 * time.Millisecond):
	}

	select {
	case <-known.asked:
	case <-time.After(10 * time.Second):
		t.Fatal("nothing was collected behind the caller")
	}

	stop()
	close(known.let)
	select {
	case <-waited:
	case <-time.After(10 * time.Second):
		t.Fatal("the wait did not end with the round it was covering")
	}
}
