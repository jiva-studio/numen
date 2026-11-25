package session_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

type signedIn struct{ expiresAt time.Time }

func (signedIn) GetPublicIdentity(context.Context) (domain.Identity, error) {
	return domain.Identity{DeviceSignKey: []byte{1}}, nil
}

func (signedIn) SignAsDevice(context.Context, []byte) ([]byte, error) { return []byte{2}, nil }

func (signedIn) GetChallenge(context.Context, []byte) (domain.Challenge, error) {
	return domain.Challenge{Message: []byte{3}}, nil
}

func (s signedIn) SignIn(context.Context, domain.Proof) (domain.Session, error) {
	return domain.Session{Token: "t", ExpiresAt: s.expiresAt}, nil
}

type gatedHealth struct {
	gate     chan struct{}
	entered  chan struct{}
	inFlight atomic.Int32
	mostSeen atomic.Int32
	once     sync.Once
}

func (h *gatedHealth) Report(context.Context, health.Item) error { return nil }

func (h *gatedHealth) Clear(context.Context, string) error {
	now := h.inFlight.Add(1)
	defer h.inFlight.Add(-1)
	for {
		seen := h.mostSeen.Load()
		if now <= seen || h.mostSeen.CompareAndSwap(seen, now) {
			break
		}
	}
	h.once.Do(func() { close(h.entered) })
	<-h.gate
	return nil
}

func TestKeep_RestoreWaitsForTheSignInThatClearsTheItems(t *testing.T) {
	clock := newManualClock(start())
	fake := signedIn{expiresAt: start().Add(time.Hour)}
	spy := &gatedHealth{gate: make(chan struct{}), entered: make(chan struct{})}
	keep := session.NewKeep(session.Deps{Identity: fake, Server: fake, Health: spy, Clock: clock, Chains: nil})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		if err := keep.Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()
	<-spy.entered

	restored := make(chan error, 1)
	go func() { restored <- keep.Restore(ctx) }()
	time.Sleep(50 * time.Millisecond)
	close(spy.gate)
	if err := <-restored; err != nil {
		t.Fatal(err)
	}

	if most := spy.mostSeen.Load(); most != 1 {
		t.Fatalf("%d clears ran at once", most)
	}
}
