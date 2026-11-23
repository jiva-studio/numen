package hostendpoint

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type memoryKeys struct {
	mu   sync.Mutex
	keys map[string]string
}

func (s *memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value, ok := s.keys[name]; ok {
		return value, nil
	}
	return "", plugindomain.ErrNoKey
}

func (s *memoryKeys) SetKey(_ context.Context, name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[name] = key
	return nil
}

func (s *memoryKeys) DeleteKey(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, name)
	return nil
}

type memoryChain struct {
	mu    sync.Mutex
	chain []byte
}

func (f *memoryChain) Read(context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), f.chain...), nil
}

func (f *memoryChain) Write(_ context.Context, chain []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chain = append([]byte(nil), chain...)
	return nil
}

func (f *memoryChain) Delete(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chain = nil
	return nil
}

type deviceNames struct{}

func (deviceNames) GetInstanceName() string { return "Laptop" }

// revokeSetup is a device with an account of two devices that is revoking the second one, served to a plugin by the host endpoint
// over the real use cases of the identity.
type revokeSetup struct {
	chains  *memoryChain
	client  numenv1connect.HostAccountServiceClient
	root    identitylib.RecoveryKey
	phone   identitylib.DeviceKey
	account identitylib.Chain
	clock   *testsupport.ManualClock
}

func newRevokeSetup(t *testing.T) *revokeSetup {
	t.Helper()
	clock := testsupport.NewClockAt(time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC))
	keys, chains, progress := &memoryKeys{keys: map[string]string{}}, &memoryChain{}, identity.NewProgress()
	deps := identity.NewDeps(keys, deviceNames{}, clock).WithChains(chains).WithProgress(progress)
	words, err := identity.NewGenerateRecoveryWords(deps).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identity.NewCreateIdentity(deps).Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}
	parsed, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		t.Fatal(err)
	}
	root, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	phone, err := identitylib.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := chains.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	own, err := identitylib.ParseChain(stored)
	if err != nil {
		t.Fatal(err)
	}
	withPhone, err := own.AppendDevice(root, phone.Public(), "Phone", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := chains.Write(t.Context(), withPhone.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := identity.NewRevokeDevice(deps).Execute(t.Context(), phone.Public().Fingerprint().String()); err != nil {
		t.Fatal(err)
	}
	account := identity.NewAccount(func() pluginport.KeyStore { return keys }, deviceNames{}, clock, progress, chains)
	fake := newCallers()
	fake.confined = true
	fake.granted = []string{manifest.PermissionIdentity, manifest.PermissionIdentityLink}
	endpoint := &Endpoint{callers: fake.asCallers()}
	calls := AccountCalls{Chain: identity.NewAccountLink(account)}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostAccountServiceHandler(&accountEndpoint{auth: endpoint, calls: calls}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostAccountServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.CreateAuthInterceptor("token-a")))
	return &revokeSetup{chains: chains, client: client, root: root, phone: phone, account: withPhone, clock: clock}
}

func (s *revokeSetup) getPendingChain(t *testing.T) *v1.GetChainPublishResponse {
	t.Helper()
	answer, err := s.client.GetChainPublish(t.Context(), connect.NewRequest(&v1.GetChainPublishRequest{}))
	if err != nil {
		t.Fatalf("GetChainPublish() = %v", err)
	}
	return answer.Msg
}

func (s *revokeSetup) readChain(t *testing.T, raw []byte) identitylib.Chain {
	t.Helper()
	chain, err := identitylib.ParseChain(raw)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func (s *revokeSetup) putPublished(t *testing.T, accepted bool, latest []byte) error {
	t.Helper()
	_, err := s.client.PutChainPublished(t.Context(), connect.NewRequest(&v1.PutChainPublishedRequest{Accepted: accepted, Latest: latest}))
	return err
}

func (s *revokeSetup) addDevice(t *testing.T, by identitylib.Signer) []byte {
	t.Helper()
	tablet, err := identitylib.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	head, err := s.account.AppendDevice(by, tablet.Public(), "Tablet", s.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	return head.Bytes()
}

func TestAPluginPublishesTheChainOfARevokeAndTheCoreKeepsItOnceTheServerAcceptedIt(t *testing.T) {
	s := newRevokeSetup(t)
	pending := s.getPendingChain(t)
	if !pending.GetIsPending() || s.readChain(t, pending.GetChain()).IsActiveDevice(s.phone.Public().Sign) {
		t.Fatalf("pending = %v, want a chain that revokes the phone", pending)
	}
	stored, err := s.chains.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !s.readChain(t, stored).IsActiveDevice(s.phone.Public().Sign) {
		t.Fatal("the core kept the revoke before the server accepted it")
	}

	if err := s.putPublished(t, true, nil); err != nil {
		t.Fatalf("PutChainPublished(accepted) = %v", err)
	}

	stored, err = s.chains.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if s.readChain(t, stored).IsActiveDevice(s.phone.Public().Sign) || s.getPendingChain(t).GetIsPending() {
		t.Error("after the answer the phone is still active or the chain is still pending")
	}
}

func TestAStaleRevokeIsBuiltAgainOnTheChainOfTheServerWhenAnotherDeviceAddedADevice(t *testing.T) {
	s := newRevokeSetup(t)
	head := s.addDevice(t, s.root)

	if err := s.putPublished(t, false, head); err != nil {
		t.Fatalf("PutChainPublished(stale) = %v", err)
	}

	pending := s.getPendingChain(t)
	rebuilt := s.readChain(t, pending.GetChain())
	if !pending.GetIsPending() || rebuilt.Len() <= s.readChain(t, head).Len() || rebuilt.IsActiveDevice(s.phone.Public().Sign) {
		t.Errorf("pending = %v, want the chain of the server with the revoke of the phone on it", pending)
	}
	if got := len(rebuilt.ListDevices()); got != 2 {
		t.Errorf("the rebuilt chain has %d active devices, want this device and the added one", got)
	}
}

func TestARevokeIsEndedWithoutAnErrorWhenTheRevokedDeviceAddedADeviceMeanwhile(t *testing.T) {
	s := newRevokeSetup(t)
	head := s.addDevice(t, s.phone)

	if err := s.putPublished(t, false, head); err != nil {
		t.Fatalf("PutChainPublished(raced) = %v, want an answer that is not a failure", err)
	}

	if s.getPendingChain(t).GetIsPending() {
		t.Error("the aborted revoke is still pending")
	}
	stored, err := s.chains.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	kept := s.readChain(t, stored)
	if !kept.IsActiveDevice(s.phone.Public().Sign) || kept.Len() != s.readChain(t, head).Len() {
		t.Error("the core must hold the chain of the server with the phone still active")
	}
}

func TestAnAnswerWithNoRevokeWaitingIsAFailedPrecondition(t *testing.T) {
	s := newRevokeSetup(t)
	if err := s.putPublished(t, true, nil); err != nil {
		t.Fatal(err)
	}

	err := s.putPublished(t, true, nil)

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want failed precondition", err)
	}
}
