package host_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestInferenceRetainsItsTurnWhileCanceledNativeWorkUnwinds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backing := hosttest.NewQueue()
		var leaseContext context.Context
		queue := host.InferenceQueue(func(ctx context.Context, work host.Work) (host.Turn, error) {
			leaseContext = ctx
			return backing.EnqueueWork(ctx, work)
		})
		ctx, cancel := context.WithCancel(t.Context())
		turn, err := queue.Acquire(ctx, host.Work{IsUserStarted: true})
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		synctest.Wait()
		if leaseContext.Err() != nil {
			t.Fatal("canceling inference ended its turn before the native call returned")
		}
		next := make(chan error, 1)
		go func() {
			nextTurn, err := host.InferenceQueue(backing.EnqueueWork).Acquire(t.Context(), host.Work{})
			if err == nil {
				err = nextTurn.Release()
			}
			next <- err
		}()
		synctest.Wait()
		select {
		case <-next:
			t.Fatal("overlapping native inference")
		default:
		}
		select {
		case <-turn.WatchWaiting():
		default:
			t.Fatal("the running transcription did not see a waiting engine")
		}
		if err := turn.Release(); err != nil {
			t.Fatal(err)
		}
		if leaseContext.Err() == nil {
			t.Fatal("released turn retained its stream context")
		}
		if err := <-next; err != nil {
			t.Fatal(err)
		}
	})
}

func TestCanceledInferenceWaitingForAnotherEngineNeverRuns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue := host.InferenceQueue(hosttest.NewQueue().EnqueueWork)
		turn, err := queue.Acquire(t.Context(), host.Work{})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		waited := make(chan error, 1)
		go func() { _, err := queue.Acquire(ctx, host.Work{IsUserStarted: true}); waited <- err }()
		synctest.Wait()
		cancel()
		if err := <-waited; !errors.Is(err, context.Canceled) {
			t.Fatalf("wait returned %v", err)
		}
		if err := turn.Release(); err != nil {
			t.Fatal(err)
		}
	})
}
