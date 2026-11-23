package hostendpoint

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// accountOperations answers the identity operations the endpoint asks for and records who asked.
type accountOperations struct {
	mu        sync.Mutex
	pluginIDs []string
	messages  [][]byte
	signature []byte
	public    identitydomain.Public
	err       error

	dataContexts [][]byte
	plains       [][]byte
	chains       [][]byte
	sealed       []byte
	opened       []byte
	pending      identitydomain.PendingRestore
	recovered    []byte
}

func (a *accountOperations) SignAsDevice(_ context.Context, pluginID string, message []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.messages = append(a.messages, bytes.Clone(message))
	return a.signature, a.err
}

func (a *accountOperations) GetPublicIdentity(context.Context) (identitydomain.Public, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.public, a.err
}

func (a *accountOperations) SealData(_ context.Context, pluginID string, dataContext, plain []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.dataContexts = append(a.dataContexts, bytes.Clone(dataContext))
	a.plains = append(a.plains, bytes.Clone(plain))
	return a.sealed, a.err
}

func (a *accountOperations) OpenData(_ context.Context, pluginID string, dataContext, sealed []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.dataContexts = append(a.dataContexts, bytes.Clone(dataContext))
	a.plains = append(a.plains, bytes.Clone(sealed))
	return a.opened, a.err
}

func (a *accountOperations) GetPendingRestore(context.Context) (identitydomain.PendingRestore, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pending, a.err
}

func (a *accountOperations) SignAsRecovery(_ context.Context, pluginID string, message []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.messages = append(a.messages, bytes.Clone(message))
	return a.recovered, a.err
}

func (a *accountOperations) CompleteRestore(_ context.Context, pluginID string, chain []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.chains = append(a.chains, bytes.Clone(chain))
	return a.err
}

func (a *accountOperations) callCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.pluginIDs)
}

type accountHost struct {
	operations *accountOperations
	callers    *callers
	client     func(token string) numenv1connect.HostAccountServiceClient
}

func newAccountHost(t *testing.T) accountHost {
	t.Helper()
	operations := &accountOperations{
		signature: bytes.Repeat([]byte{7}, 64),
		public: identitydomain.Public{
			AccountFingerprint: "3F2A 91C0 0000 0000 0000 0000 0000 0000",
			DeviceFingerprint:  "AAAA BBBB 0000 0000 0000 0000 0000 0000",
			DeviceSignKey:      bytes.Repeat([]byte{1}, 32),
			Chain:              []byte("chain"),
		},
	}
	fake := newCallers()
	fake.confined = true
	fake.granted = []string{manifest.PermissionIdentity}
	endpoint := &Endpoint{callers: fake.asCallers()}
	calls := AccountCalls{Account: operations, Restore: operations}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostAccountServiceHandler(&accountEndpoint{auth: endpoint, calls: calls}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return accountHost{operations: operations, callers: fake, client: func(token string) numenv1connect.HostAccountServiceClient {
		return numenv1connect.NewHostAccountServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.CreateAuthInterceptor(token)))
	}}
}

func TestAPluginWithTheIdentityPermissionSignsAsTheDevice(t *testing.T) {
	h := newAccountHost(t)

	response, err := h.client("token-a").SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: []byte("challenge")}))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(response.Msg.GetSignature(), h.operations.signature) {
		t.Errorf("signature = %x", response.Msg.GetSignature())
	}
	if len(h.operations.messages) != 1 || string(h.operations.messages[0]) != "challenge" {
		t.Errorf("messages signed = %q", h.operations.messages)
	}
}

func TestTheDeviceSignsForThePluginOfTheTokenAndNotOfTheRequest(t *testing.T) {
	h := newAccountHost(t)

	if _, err := h.client("token-c").SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: []byte("m")})); err != nil {
		t.Fatal(err)
	}

	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" {
		t.Errorf("plugin ids signed for = %v, want [md.numen.c]", h.operations.pluginIDs)
	}
}

func TestAPluginWithoutTheIdentityPermissionIsRefusedBothCalls(t *testing.T) {
	h := newAccountHost(t)
	h.callers.granted = []string{manifest.PermissionVaultRead}
	client := h.client("token-a")

	_, signErr := client.SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: []byte("m")}))
	_, readErr := client.GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))

	if connect.CodeOf(signErr) != connect.CodePermissionDenied || connect.CodeOf(readErr) != connect.CodePermissionDenied {
		t.Errorf("sign = %v, read = %v, want both permission denied", signErr, readErr)
	}
	if h.operations.callCount() != 0 {
		t.Error("the identity operations ran for a plugin without the permission")
	}
	if got := h.callers.checked; len(got) != 2 || got[0] != "md.numen.a identity" || got[1] != "md.numen.a identity" {
		t.Errorf("permissions checked = %v, want identity twice", got)
	}
}

func TestAnUnknownTokenIsUnauthenticatedForTheAccountCalls(t *testing.T) {
	h := newAccountHost(t)

	_, err := h.client("token-of-nobody").GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

func TestAMessageOfNoBytesOrOverTheLimitIsAnInvalidArgument(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-a")

	for name, message := range map[string][]byte{"empty": nil, "over the limit": bytes.Repeat([]byte{1}, 4097)} {
		_, err := client.SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: message}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want invalid argument", name, err)
		}
	}
	_, err := client.SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: bytes.Repeat([]byte{1}, 4096)}))
	if err != nil {
		t.Errorf("a message of 4096 bytes: %v", err)
	}
	if h.operations.callCount() != 1 {
		t.Errorf("operations ran %d times, want once for the message within the limit", h.operations.callCount())
	}
}

func TestADeviceWithoutAnIdentityAnswersAFailedPreconditionWithTheReasonNoIdentity(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrNoIdentity
	client := h.client("token-a")

	_, signErr := client.SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: []byte("m")}))
	_, readErr := client.GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))

	for name, err := range map[string]error{"sign": signErr, "read": readErr} {
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("%s: error = %v, want failed precondition", name, err)
			continue
		}
		if reason := readReason(err); reason != "NO_IDENTITY" {
			t.Errorf("%s: reason = %q, want NO_IDENTITY", name, reason)
		}
	}
}

func TestAKeyStoreFailureIsInternalAndNamesNoReason(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = errors.New("the keychain refuses")

	_, err := h.client("token-a").GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))

	if connect.CodeOf(err) != connect.CodeInternal || readReason(err) != "" {
		t.Errorf("error = %v, reason %q, want internal with no reason", err, readReason(err))
	}
}

func TestThePublicIdentityHasFingerprintsTheSignKeyAndTheChainAndNothingSecret(t *testing.T) {
	h := newAccountHost(t)

	response, err := h.client("token-a").GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := response.Msg
	want := h.operations.public
	if got.GetAccountFingerprint() != want.AccountFingerprint || got.GetDeviceFingerprint() != want.DeviceFingerprint ||
		!bytes.Equal(got.GetDeviceSignKey(), want.DeviceSignKey) || !bytes.Equal(got.GetChain(), want.Chain) {
		t.Errorf("response = %v, want %+v", got, want)
	}
	if fields := got.ProtoReflect().Descriptor().Fields().Len(); fields != 7 {
		t.Errorf("the response has %d fields, want the 7 public ones: 4 of the identity, the key epoch, the length of the chain and the revoked flag", fields)
	}
}

func readReason(err error) string {
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return ""
	}
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if info, isInfo := value.(*errdetails.ErrorInfo); isInfo {
			return info.GetReason()
		}
	}
	return ""
}
