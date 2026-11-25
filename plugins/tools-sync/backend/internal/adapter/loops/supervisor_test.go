package loops_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
)

func TestSupervisor_GetAnswersWhileConfigureWaitsForTheOldLoopsToStop(t *testing.T) {
	stopping := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	supervisor := loops.NewSupervisor(func(_ context.Context, address string) (string, map[string]func(context.Context) error) {
		return address, map[string]func(context.Context) error{"slow": func(ctx context.Context) error {
			<-ctx.Done()
			once.Do(func() { close(stopping) })
			<-release
			return nil
		}}
	})
	supervisor.Configure("first", "")
	configured := make(chan struct{})
	go func() {
		supervisor.Configure("second", "")
		close(configured)
	}()
	<-stopping

	got := make(chan string, 1)
	go func() {
		value, _ := supervisor.Get()
		got <- value
	}()

	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Error("Get was blocked by a Configure that waits for loops to stop")
	}
	close(release)
	<-configured
	supervisor.Stop()
}

func TestSupervisor_GetAnswersWhileTheSessionIsBeingBuilt(t *testing.T) {
	building := make(chan struct{})
	release := make(chan struct{})
	supervisor := loops.NewSupervisor(func(_ context.Context, address string) (string, map[string]func(context.Context) error) {
		close(building)
		<-release
		return address, nil
	})
	configured := make(chan struct{})
	go func() {
		supervisor.Configure("first", "")
		close(configured)
	}()
	<-building

	got := make(chan bool, 1)
	go func() {
		_, isStarted := supervisor.Get()
		got <- isStarted
	}()

	select {
	case isStarted := <-got:
		if isStarted {
			t.Error("a session that is still built was reported as started")
		}
	case <-time.After(2 * time.Second):
		t.Error("Get was blocked by a Configure that builds the session")
	}
	close(release)
	<-configured
	supervisor.Stop()
}

func TestSupervisor_GetReportsNoSessionWhileTheLoopsStop(t *testing.T) {
	stopping := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	supervisor := loops.NewSupervisor(func(_ context.Context, address string) (string, map[string]func(context.Context) error) {
		return address, map[string]func(context.Context) error{"slow": func(ctx context.Context) error {
			<-ctx.Done()
			once.Do(func() { close(stopping) })
			<-release
			return nil
		}}
	})
	supervisor.Configure("first", "")
	stopped := make(chan struct{})
	go func() {
		supervisor.Stop()
		close(stopped)
	}()
	<-stopping

	_, isStarted := supervisor.Get()

	if isStarted {
		t.Error("Get returned the session whose loops are being stopped")
	}
	close(release)
	<-stopped
}

func TestSupervisor_ConfigureStartsTheSessionAgainWhenOnlyTheOptionsChange(t *testing.T) {
	var starts atomic.Int64
	supervisor := loops.NewSupervisor(func(_ context.Context, address string) (string, map[string]func(context.Context) error) {
		starts.Add(1)
		return address, nil
	})
	t.Cleanup(supervisor.Stop)

	supervisor.Configure("first", "a")
	supervisor.Configure("first", "a")
	supervisor.Configure("first", "b")

	if got := starts.Load(); got != 2 {
		t.Errorf("the session was built %d times, want 2: once for each options value", got)
	}
}
