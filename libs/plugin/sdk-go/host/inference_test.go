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
		release, err := queue.Acquire(ctx, true)
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
			releaseNext, err := host.InferenceQueue(backing.EnqueueWork).Acquire(t.Context(), false)
			if err == nil {
				err = releaseNext()
			}
			next <- err
		}()
		synctest.Wait()
		select {
		case <-next:
			t.Fatal("overlapping native inference")
		default:
		}
		if err := release(); err != nil {
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
		release, err := queue.Acquire(t.Context(), false)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		waited := make(chan error, 1)
		go func() { _, err := queue.Acquire(ctx, true); waited <- err }()
		synctest.Wait()
		cancel()
		if err := <-waited; !errors.Is(err, context.Canceled) {
			t.Fatalf("wait returned %v", err)
		}
		if err := release(); err != nil {
			t.Fatal(err)
		}
	})
}
