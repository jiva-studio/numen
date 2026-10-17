package recognition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// holding is a set of source queries whose first question never answers. It is
// a round that has begun and not ended.
type holding struct {
	port.SourceQueries
	asked chan struct{}
	let   chan struct{}
}

func (h *holding) GetFingerprints(
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
	// Nothing was read, so the round ends here.
	return nil, errHeld
}

func (h *holding) GetRecognisedSources(
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

// TestCollectingRunsBehindTheCallerAndIsCountedBeforeIt. The batches left with a
// proofreader are asked after on the same terms as the queue: behind whoever
// asked, and counted in the call that starts them.
func TestCollectingRunsBehindTheCallerAndIsCountedBeforeIt(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	w := newWatched(t, nil)
	w.Worker.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, Batch: 1,
		Queue: func() (port.ProofreadQueue, error) { return leaves{}, nil },
	}
	v := domain.Vault{ID: "v", Path: t.TempDir()}
	known := &holding{asked: make(chan struct{}, 1), let: make(chan struct{})}

	came := make(chan struct{})
	go func() {
		defer close(came)
		w.CollectBatches(ctx, known, time.Hour, v)
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

// waitForContext holds the caller until the context ends.
func waitForContext(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}
