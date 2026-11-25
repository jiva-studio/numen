package session_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func newTokenKeep(t *testing.T, hasIdentity bool) (*session.Keep, *fakeserver.Server, *healthtest.Host) {
	t.Helper()
	clock := newManualClock(start())
	server := fakeserver.New(fakeserver.Config{Now: clock.Now})
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	server.SetAudience(served.URL)
	identity := accounttest.NewFake(t, syncprotocol.SignInPluginID)
	if !hasIdentity {
		identity.RemoveIdentity()
	}
	healthHost := healthtest.NewHost(t)
	identityEndpoint, err := hostcall.ReadVars(identity.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	healthEndpoint, err := hostcall.ReadVars(healthHost.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	keep := session.NewKeep(session.Deps{
		Identity: accountadapter.New(account.NewClient(identityEndpoint)),
		Server:   serveradapter.New(serveradapter.Config{Address: served.URL, HTTP: http.DefaultClient, Now: clock.Now}),
		Health:   health.NewClient(healthEndpoint),
		Clock:    clock,
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- keep.Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("Keep ended with %v, want the error of its context", err)
		}
	})
	return keep, server, healthHost
}

func TestKeep_GetsTheTokenOfTheSessionItOpened(t *testing.T) {
	keep, server, _ := newTokenKeep(t, true)

	eventually(t, "the token", func() bool { return keep.GetToken() != "" })

	if sessions := server.ListSessions(); len(sessions) != 1 || keep.GetToken() != sessions[0].Token {
		t.Error("GetToken is not the token of the session the server holds")
	}
}

func TestKeep_GetsNoTokenBeforeItSignedIn(t *testing.T) {
	keep, server, healthHost := newTokenKeep(t, false)

	eventually(t, "the warning of a missing identity", func() bool {
		for _, item := range healthHost.List() {
			if item.Key == "no-identity" {
				return true
			}
		}
		return false
	})

	if got := keep.GetToken(); got != "" {
		t.Errorf("GetToken = %q, want none without an identity", got)
	}
	if sessions := server.ListSessions(); len(sessions) != 0 {
		t.Errorf("sessions = %d, want none", len(sessions))
	}
}
