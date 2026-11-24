package synctest

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/device"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/shareinvite"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/invitewire"
)

const eventuallyLimit = 10 * time.Second

// InviteDeps are the dependencies of the use cases of sharing a folder by a link, over the relay of the server of this
// computer. They share the hub, so a link made through one set of use cases is found by the others.
func (c *Computer) InviteDeps(hub *shareinvite.Hub) shareinvite.Deps {
	return shareinvite.Deps{
		Relay: syncclient.NewLinkClient(c.Server.URL, http.DefaultClient), Health: c.NewHealth(), Clock: c.Clock, Identity: c.Account,
		Device: device.Host{}, Sharing: c.ShareDeps(), Receiving: c.ReceiveDeps(), Random: rand.Reader, Server: c.Server.URL, Hub: hub,
	}
}

// StartInviteCourier runs the courier of the links of this computer until the test ends, and returns the deps it works for.
func (c *Computer) StartInviteCourier(t *testing.T) shareinvite.Deps {
	t.Helper()
	hub := shareinvite.NewHub()
	deps := c.InviteDeps(hub)
	ctx, stop := context.WithCancel(t.Context())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		if err := shareinvite.NewCourier(hub).Execute(ctx); err != nil && ctx.Err() == nil {
			t.Error(err)
		}
	}()
	t.Cleanup(func() {
		stop()
		<-finished
	})
	return deps
}

// NewInvites returns the methods of the view about sharing by a link, with the courier running.
func (c *Computer) NewInvites(t *testing.T) *invitewire.Invites {
	t.Helper()
	return invitewire.New(c.StartInviteCourier(t))
}

// Eventually waits until the condition holds, as the couriers work in the background, and fails the test when it does not.
func Eventually(t *testing.T, what string, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), eventuallyLimit)
	defer cancel()
	for !condition() {
		select {
		case <-ctx.Done():
			t.Fatalf("waited for %s and it did not happen", what)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
