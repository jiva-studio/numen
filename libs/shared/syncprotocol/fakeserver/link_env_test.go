package fakeserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

func serve(tb testing.TB, server *fakeserver.Server) string {
	tb.Helper()
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	return served.URL
}

func trySignIn(tb testing.TB, address string, device identity.DeviceKey, chain identity.Chain) (string, error) {
	tb.Helper()
	client := syncclient.NewSignInClient(address, http.DefaultClient, machineClock{}.Now)
	key := device.Public().Sign
	challenge, err := client.GetChallenge(tb.Context(), key)
	if err != nil {
		return "", err
	}
	signature := device.SignForPlugin(syncprotocol.SignInPluginID, challenge.Message)
	session, err := client.SignIn(tb.Context(), syncclient.Proof{ChallengeID: challenge.ID, DeviceSignKey: key, Signature: signature[:], Chain: chain.Bytes()})
	return session.Token, err
}
