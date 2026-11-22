package fakesync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/fakesync"
)

type world struct {
	server  *fakesync.Server
	url     string
	device  *account.Client
	public  account.PublicIdentity
	service syncv1connect.SignInServiceClient
}

func newWorld(t *testing.T, signAs string, config fakesync.Config) world {
	t.Helper()
	server := fakesync.New(config)
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	server.SetAudience(served.URL)
	fake := accounttest.NewFake(t, signAs)
	endpoint, err := hostcall.FromVars(fake.Vars("token"))
	if err != nil {
		t.Fatal(err)
	}
	client := account.NewClient(endpoint)
	public, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	service := syncv1connect.NewSignInServiceClient(http.DefaultClient, served.URL, connect.WithInterceptors(syncprotocol.NewVersionInterceptor()))
	return world{server: server, url: served.URL, device: client, public: public, service: service}
}

func (w world) getChallenge(ctx context.Context, t *testing.T) *syncv1.GetChallengeResponse {
	t.Helper()
	response, err := w.service.GetChallenge(ctx, connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: w.public.DeviceSignKey}))
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg
}

func (w world) signIn(ctx context.Context, t *testing.T, challenge *syncv1.GetChallengeResponse) error {
	t.Helper()
	signature, err := w.device.SignAsDevice(ctx, syncprotocol.GetSignInMessage(challenge, w.public.DeviceSignKey, syncprotocol.Version))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.service.SignIn(ctx, connect.NewRequest(&syncv1.SignInRequest{
		ChallengeId: challenge.ChallengeId, DeviceSignKey: w.public.DeviceSignKey, Signature: signature, Chain: w.public.Chain,
	}))
	return err
}

func TestASignInSignedUnderThePluginOfTheServerOpensASessionOfTheAccount(t *testing.T) {
	w := newWorld(t, fakesync.PluginID, fakesync.Config{})

	if err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t)); err != nil {
		t.Fatal(err)
	}

	sessions := w.server.ListSessions()
	if len(sessions) != 1 || sessions[0].AccountFingerprint != w.public.AccountFingerprint || string(sessions[0].DeviceSignKey) != string(w.public.DeviceSignKey) {
		t.Errorf("sessions = %+v, want one of this device and its account", sessions)
	}
}

func TestASignInSignedUnderAnotherPluginIsUnauthenticated(t *testing.T) {
	w := newWorld(t, "md.numen.tools.link", fakesync.Config{})

	err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t))

	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(w.server.ListSessions()) != 0 {
		t.Errorf("error = %v with sessions %+v, want unauthenticated and none", err, w.server.ListSessions())
	}
}

func TestAChallengeIsUsedOnceWhateverTheResult(t *testing.T) {
	w := newWorld(t, fakesync.PluginID, fakesync.Config{})
	challenge := w.getChallenge(t.Context(), t)

	first := w.signIn(t.Context(), t, challenge)
	second := w.signIn(t.Context(), t, challenge)

	if first != nil || connect.CodeOf(second) != connect.CodeUnauthenticated {
		t.Errorf("first = %v, second = %v, want a session then unauthenticated", first, second)
	}
}

func TestAServerOfAnotherProtocolVersionAsksTheClientToUpdate(t *testing.T) {
	w := newWorld(t, fakesync.PluginID, fakesync.Config{Protocol: syncprotocol.Version + 1})

	_, err := w.service.GetChallenge(t.Context(), connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: w.public.DeviceSignKey}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want failed precondition", err)
	}
}

func TestEndingTheSessionsOfTheDeviceRemovesThem(t *testing.T) {
	w := newWorld(t, fakesync.PluginID, fakesync.Config{})
	if err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t)); err != nil {
		t.Fatal(err)
	}

	w.server.EndSessions(w.public.DeviceSignKey)

	if sessions := w.server.ListSessions(); len(sessions) != 0 {
		t.Errorf("sessions = %+v, want none", sessions)
	}
}
