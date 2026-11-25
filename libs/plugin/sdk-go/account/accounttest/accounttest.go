// Package accounttest is a host that serves the identity of a device, for the tests of a plugin that signs in
// with it. The identity is a real record of libs/shared/identity made in memory, so a signature it makes
// verifies with identity.VerifyForPlugin.
package accounttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

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

	mu          sync.Mutex
	record      *fakeIdentity
	link        *fakeLink
	failure     error
	signed      int
	pending     *pendingRestore
	clock       interface{ Now() time.Time }
	linkStarted chan struct{}

	completeHook func()
}

// SetClock makes the host read the time of the clock when it signs a revoke, instead of the machine clock.
func (f *Fake) SetClock(source interface{ Now() time.Time }) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clock = source
}

func (f *Fake) now() time.Time {
	if f.clock != nil {
		return f.clock.Now()
	}
	return clock.System{}.Now()
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
	f.link = nil
}

// StartRestore makes the host keep a restore of the recovery words, as the window of a host does. The host has no
// identity until a call of CompleteRestore finishes the restore: an identity it had is removed. It fails the test for words that are not valid.
func (f *Fake) StartRestore(words []string) {
	f.t.Helper()
	restore := newPendingRestore(f.t, words)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record = nil
	f.pending = restore
}

// StartRestoreAsCore is StartRestore for a host that completes the restore as the core does: the recovery key starts a new epoch of keys
// for every device of the chain.
func (f *Fake) StartRestoreAsCore(words []string) {
	f.t.Helper()
	restore := newPendingRestore(f.t, words)
	restore.rotates = true
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record = nil
	f.pending = restore
}

// GetAccountKeys returns the account keys the host holds now, with the history of the earlier epochs. A test keeps them to check what a
// device that was revoked later could still open.
func (f *Fake) GetAccountKeys() (identity.AccountKeys, error) { return f.readAccountKeys() }

// Fail makes every call fail with err, which is not account.ErrNoIdentity. A nil err ends the failure.
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
		KeyEpoch: public.KeyEpoch, ChainLength: public.ChainLength, IsRevoked: public.IsRevoked,
	}), nil
}

func (f *Fake) read() (fakeIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failure != nil {
		return fakeIdentity{}, connect.NewError(connect.CodeInternal, f.failure)
	}
	if f.record == nil {
		failure := connect.NewError(connect.CodeFailedPrecondition, account.ErrNoIdentity)
		if detail, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "NO_IDENTITY"}); err == nil {
			failure.AddDetail(detail)
		}
		return fakeIdentity{}, failure
	}
	return *f.record, nil
}

func toPublicIdentity(record fakeIdentity) account.PublicIdentity {
	device := record.Device.Public()
	return account.PublicIdentity{
		AccountFingerprint: record.Chain.Root().Fingerprint().String(),
		DeviceFingerprint:  device.Fingerprint().String(),
		DeviceSignKey:      device.Sign,
		Chain:              record.Chain.Bytes(),
		KeyEpoch:           record.Chain.Epoch(),
		ChainLength:        uint32(record.Chain.Len()),
		IsRevoked:          isRevoked(record),
	}
}

func newRecord(t *testing.T) fakeIdentity {
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
	return joinChain(t, chain, root, keys, device, now)
}

func joinChain(t *testing.T, chain identity.Chain, root identity.RecoveryKey, keys identity.AccountKeys, device identity.DeviceKey, now time.Time) fakeIdentity {
	t.Helper()
	chain, err := chain.AppendDevice(root, device.Public(), "Test device", now)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := keys.WrapForDevice(device.Public())
	if err != nil {
		t.Fatal(err)
	}
	return fakeIdentity{Record: identity.Record{Device: device, Wrapped: wrapped}, Chain: chain, root: root, keys: keys}
}

// fakeIdentity is the identity of the fake: the record, the chain and the keys that make a revoke possible.
type fakeIdentity struct {
	identity.Record
	Chain identity.Chain
	root  identity.RecoveryKey
	keys  identity.AccountKeys
}

func isRevoked(record fakeIdentity) bool {
	return slices.ContainsFunc(record.Chain.ListRevokedDevices(), func(device identity.Device) bool {
		return bytes.Equal(device.Public.Sign, record.Device.SignerKey())
	})
}
