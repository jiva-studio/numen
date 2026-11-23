package fakeserver_test

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type world struct {
	server  *fakeserver.Server
	url     string
	signAs  string
	record  identity.Record
	chain   identity.Chain
	service syncv1connect.SignInServiceClient
}

func (w world) deviceSignKey() []byte { return w.record.Device.Public().Sign }

func (w world) accountFingerprint() string { return w.chain.Root().Fingerprint().String() }

func newRecord(t *testing.T) (identity.Record, identity.Chain) {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
	chain, err := identity.NewChain(root, keys, now)
	if err != nil {
		t.Fatal(err)
	}
	chain, err = chain.AppendDevice(root, device.Public(), "Test device", now)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := keys.WrapForDevice(device.Public())
	if err != nil {
		t.Fatal(err)
	}
	return identity.Record{Device: device, Wrapped: wrapped}, chain
}

func newWorld(t *testing.T, signAs string, config fakeserver.Config) world {
	t.Helper()
	server := fakeserver.New(config)
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	server.SetAudience(served.URL)
	service := syncv1connect.NewSignInServiceClient(http.DefaultClient, served.URL, connect.WithInterceptors(syncprotocol.NewVersionInterceptor()))
	record, chain := newRecord(t)
	return world{server: server, url: served.URL, signAs: signAs, record: record, chain: chain, service: service}
}

func (w world) getChallenge(ctx context.Context, t *testing.T) *syncv1.GetChallengeResponse {
	t.Helper()
	response, err := w.service.GetChallenge(ctx, connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: w.deviceSignKey()}))
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg
}

func (w world) signIn(ctx context.Context, t *testing.T, challenge *syncv1.GetChallengeResponse) error {
	t.Helper()
	signature := w.record.Device.SignForPlugin(w.signAs, syncprotocol.GetSignInMessage(challenge, w.deviceSignKey(), syncprotocol.Version))
	_, err := w.service.SignIn(ctx, connect.NewRequest(&syncv1.SignInRequest{
		ChallengeId: challenge.GetChallengeId(), DeviceSignKey: w.deviceSignKey(), Signature: signature[:], Chain: w.chain.Bytes(),
	}))
	return err
}

func TestASignInOfTheServerPluginOpensASession(t *testing.T) {
	w := newWorld(t, syncprotocol.SignInPluginID, fakeserver.Config{})

	if err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t)); err != nil {
		t.Fatal(err)
	}

	sessions := w.server.ListSessions()
	if len(sessions) != 1 || sessions[0].AccountFingerprint != w.accountFingerprint() || string(sessions[0].DeviceSignKey) != string(w.deviceSignKey()) {
		t.Errorf("sessions = %+v, want one of this device and its account", sessions)
	}
}

func TestASignInOfAnotherPluginIsUnauthenticated(t *testing.T) {
	w := newWorld(t, "md.numen.tools.link", fakeserver.Config{})

	err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t))

	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(w.server.ListSessions()) != 0 {
		t.Errorf("error = %v with sessions %+v, want unauthenticated and none", err, w.server.ListSessions())
	}
}

func TestAChallengeIsUsedOnceWhateverTheResult(t *testing.T) {
	w := newWorld(t, syncprotocol.SignInPluginID, fakeserver.Config{})
	challenge := w.getChallenge(t.Context(), t)

	first := w.signIn(t.Context(), t, challenge)
	second := w.signIn(t.Context(), t, challenge)

	if first != nil || connect.CodeOf(second) != connect.CodeUnauthenticated {
		t.Errorf("first = %v, second = %v, want a session then unauthenticated", first, second)
	}
}

func TestAServerOfAnotherProtocolVersionAsksTheClientToUpdate(t *testing.T) {
	w := newWorld(t, syncprotocol.SignInPluginID, fakeserver.Config{Protocol: syncprotocol.Version + 1})

	_, err := w.service.GetChallenge(t.Context(), connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: w.deviceSignKey()}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want failed precondition", err)
	}
}

func TestEndingTheSessionsOfTheDeviceRemovesThem(t *testing.T) {
	w := newWorld(t, syncprotocol.SignInPluginID, fakeserver.Config{})
	if err := w.signIn(t.Context(), t, w.getChallenge(t.Context(), t)); err != nil {
		t.Fatal(err)
	}

	w.server.EndSessions(w.deviceSignKey())

	if sessions := w.server.ListSessions(); len(sessions) != 0 {
		t.Errorf("sessions = %+v, want none", sessions)
	}
}
