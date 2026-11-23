// Package accounttest is a host that serves the identity of a device, for the tests of a plugin that signs in
// with it. The identity is a real record of libs/shared/identity made in memory, so a signature it makes
// verifies with identity.VerifyForPlugin.
package accounttest

import (
	"context"
	"crypto/rand"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Fake serves the account service of a host on a loopback port for as long as the test runs. It signs
// under the plugin id it was made for, whatever the token of the call is.
type Fake struct {
	t        *testing.T
	port     string
	pluginID string

	mu      sync.Mutex
	record  *identity.Record
	failure error
	signed  int
}

// NewFake starts a host with an identity, serving a plugin of the id, and stops it when the test ends.
func NewFake(t *testing.T, pluginID string) *Fake {
	t.Helper()
	fake := &Fake{t: t, pluginID: pluginID}
	fake.CreateIdentity()
	_, handler := numenv1connect.NewHostAccountServiceHandler(&service{fake: fake})
	served := httptest.NewServer(handler)
	t.Cleanup(served.Close)
	address, err := url.Parse(served.URL)
	if err != nil {
		t.Fatal(err)
	}
	fake.port = address.Port()
	return fake
}

// BuildVars returns the variables a host gives a plugin it links in, naming this host and the token.
func (f *Fake) BuildVars(token string) map[string]string {
	return map[string]string{hostcall.HostPortEnv: f.port, hostcall.PluginTokenEnv: token}
}

// RemoveIdentity removes the identity: both calls answer account.ErrNoIdentity until CreateIdentity.
func (f *Fake) RemoveIdentity() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record = nil
}

// CreateIdentity gives the host a new identity, on a new account.
func (f *Fake) CreateIdentity() {
	f.t.Helper()
	record := newRecord(f.t)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record = &record
}

// Fail makes both calls fail with err, which is not account.ErrNoIdentity. A nil err ends the failure.
func (f *Fake) Fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failure = err
}

// SignCount is the number of signatures made.
func (f *Fake) SignCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.signed
}

// Public is what is public of the identity of the host now. It is the zero value where the host has none.
func (f *Fake) Public() account.PublicIdentity {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.record == nil {
		return account.PublicIdentity{}
	}
	return toPublicIdentity(*f.record)
}

// service serves the account service from the identity of the fake.
type service struct {
	numenv1connect.UnimplementedHostAccountServiceHandler

	fake *Fake
}

func (s *service) SignAsDevice(_ context.Context, req *connect.Request[v1.SignAsDeviceRequest]) (*connect.Response[v1.SignAsDeviceResponse], error) {
	f := s.fake
	record, err := f.read()
	if err != nil {
		return nil, err
	}
	signature := record.Device.SignForPlugin(f.pluginID, req.Msg.GetMessage())
	f.mu.Lock()
	f.signed++
	f.mu.Unlock()
	return connect.NewResponse(&v1.SignAsDeviceResponse{Signature: signature[:]}), nil
}

func (s *service) GetPublicIdentity(_ context.Context, _ *connect.Request[v1.GetPublicIdentityRequest]) (*connect.Response[v1.GetPublicIdentityResponse], error) {
	record, err := s.fake.read()
	if err != nil {
		return nil, err
	}
	public := toPublicIdentity(record)
	return connect.NewResponse(&v1.GetPublicIdentityResponse{
		AccountFingerprint: public.AccountFingerprint, DeviceFingerprint: public.DeviceFingerprint,
		DeviceSignKey: public.DeviceSignKey, Chain: public.Chain,
	}), nil
}

func (f *Fake) read() (identity.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failure != nil {
		return identity.Record{}, connect.NewError(connect.CodeInternal, f.failure)
	}
	if f.record == nil {
		failure := connect.NewError(connect.CodeFailedPrecondition, account.ErrNoIdentity)
		if detail, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "NO_IDENTITY"}); err == nil {
			failure.AddDetail(detail)
		}
		return identity.Record{}, failure
	}
	return *f.record, nil
}

func toPublicIdentity(record identity.Record) account.PublicIdentity {
	device := record.Device.Public()
	return account.PublicIdentity{
		AccountFingerprint: record.Chain.Root().Fingerprint().String(),
		DeviceFingerprint:  device.Fingerprint().String(),
		DeviceSignKey:      device.Sign,
		Chain:              record.Chain.Bytes(),
	}
}

func newRecord(t *testing.T) identity.Record {
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
	now := clock.System{}.Now()
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
	return identity.Record{Device: device, Wrapped: wrapped, Chain: chain}
}
