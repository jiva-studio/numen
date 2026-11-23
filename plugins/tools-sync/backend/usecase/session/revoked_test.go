package session_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

// revokedPair is an account of two computers on a server, where the first has revoked the second.
type revokedPair struct {
	server *synctest.Server
	first  *synctest.Computer
	second *synctest.Computer
}

func newRevokedPair(t *testing.T) *revokedPair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	if _, err := chaincourier.NewAdopt(first.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := first.Fake.RevokeDevice(second.Fake.Public().DeviceSignKey); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(first.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the revoke published", published, err)
	}
	return &revokedPair{server: server, first: first, second: second}
}

func TestAServerEndsTheSessionOfARevokedDeviceAndRefusesItsSignIn(t *testing.T) {
	p := newRevokedPair(t)

	for _, one := range p.server.Fake.ListSessions() {
		if bytes.Equal(one.DeviceSignKey, p.second.Fake.Public().DeviceSignKey) {
			t.Error("the revoked device still has a session")
		}
	}
	err := p.second.SignIn(t.Context())

	if revoked, isRevoked := errors.AsType[domain.DeviceRevokedError](err); !isRevoked || len(revoked.Chain) == 0 {
		t.Errorf("sign-in error = %v, want the refusal that carries the chain", err)
	}
}

func TestKeep_RevokedDeviceRaisesItemAndPassesServerChainToCore(t *testing.T) {
	p := newRevokedPair(t)
	keep := session.NewKeep(session.Deps{
		Identity: p.second.Account, Server: p.second.Sessions, Health: p.second.NewHealth(), Clock: p.second.Clock, Chains: p.second.Client,
	})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() { done <- keep.Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("keep ended with %v, want the cancel of its context", err)
		}
	})

	eventually(t, "the item that says the device was revoked", func() bool { return p.second.HasItemKey("device-revoked") })
	eventually(t, "the core marking the device revoked", func() bool { return p.second.Fake.Public().IsRevoked })
	if !p.second.HasItem("This device was revoked") {
		t.Error("the item does not say This device was revoked")
	}
	if p.first.Fake.Public().IsRevoked {
		t.Error("the device that revoked is revoked too")
	}
}
