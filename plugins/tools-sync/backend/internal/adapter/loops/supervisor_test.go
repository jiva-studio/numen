package loops_test

import (
	"context"
	"sync"
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
	supervisor.Configure("first")
	configured := make(chan struct{})
	go func() {
		supervisor.Configure("second")
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
		supervisor.Configure("first")
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
