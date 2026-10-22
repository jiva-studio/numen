package transcription

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

// holding is a set of source queries whose first question never answers. It is
// a round that has begun and not ended.
type holding struct {
	sourceport.SourceQueries
	requestChan chan struct{}
	let         chan struct{}
}

func (h *holding) GetFingerprints(
	ctx context.Context, _ domain.VaultID, _ domain.SourceKind,
) (map[string]domain.Fingerprint, error) {
	select {
	case h.requestChan <- struct{}{}:
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
) ([]sourceport.SourceText, error) {
	select {
	case h.requestChan <- struct{}{}:
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
func TestAQueueRunsPriorTheCallerAndIsCountedBeforeIt(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	target, v := newTranscriptionWorker(t, &deaf{})
	recognized := &holding{requestChan: make(chan struct{}, 1), let: make(chan struct{})}

	// A call that does not come back is a window that never opens.
	came := make(chan struct{})
	go func() {
		defer close(came)
		target.Queue(ctx, recognized, time.Hour, v)
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
		target.Wait()
	}()
	select {
	case <-waited:
		t.Fatal("the wait returned with a round of the queue still to run")
	case <-time.After(200 * time.Millisecond):
	}

	select {
	case <-recognized.requestChan:
	case <-time.After(10 * time.Second):
		t.Fatal("no round of the queue ran behind the caller")
	}

	stop()
	close(recognized.let)
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

// neverDelay accepts a delayed call and never makes it.
func neverDelay(time.Duration, func()) func() bool { return func() bool { return true } }
