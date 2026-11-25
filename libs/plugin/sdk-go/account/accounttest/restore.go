package accounttest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const maxSealPlain = 32 << 10

var errDataSize = errors.New("the data to seal has at most 32 KiB")

type pendingRestore struct {
	recovery identity.RecoveryKey
	device   identity.DeviceKey
}

func newPendingRestore(t *testing.T, words []string) *pendingRestore {
	t.Helper()
	parsed, err := identity.ParseRecoveryWords(words)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &pendingRestore{recovery: recovery, device: device}
}

func refuse(code connect.Code, reason string) error {
	failure := connect.NewError(code, errors.New(reason))
	if detail, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: reason}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func (f *Fake) readPendingRestore() (*pendingRestore, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failure != nil {
		return nil, connect.NewError(connect.CodeInternal, f.failure)
	}
	if f.pending == nil {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_PENDING_RESTORE")
	}
	return f.pending, nil
}

func (f *Fake) readAccountKeys() (identity.AccountKeys, error) {
	record, err := f.read()
	if err != nil {
		return identity.AccountKeys{}, err
	}
	keys, err := record.Device.UnwrapAccountKeys(record.Wrapped)
	if err != nil {
		return identity.AccountKeys{}, err
	}
	return record.Chain.OpenKeyHistory(keys)
}

func (s *service) SealData(_ context.Context, req *connect.Request[v1.SealDataRequest]) (*connect.Response[v1.SealDataResponse], error) {
	if len(req.Msg.GetPlain()) > maxSealPlain {
		return nil, connect.NewError(connect.CodeInvalidArgument, errDataSize)
	}
	keys, err := s.fake.readAccountKeys()
	if err != nil {
		return nil, err
	}
	sealed, err := keys.SealForPlugin(rand.Reader, s.fake.pluginID, req.Msg.GetContext(), req.Msg.GetPlain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SealDataResponse{Sealed: sealed}), nil
}

func (s *service) OpenData(_ context.Context, req *connect.Request[v1.OpenDataRequest]) (*connect.Response[v1.OpenDataResponse], error) {
	keys, err := s.fake.readAccountKeys()
	if err != nil {
		return nil, err
	}
	plain, err := keys.OpenForPlugin(s.fake.pluginID, req.Msg.GetContext(), req.Msg.GetSealed())
	if err != nil {
		return nil, refuse(connect.CodeFailedPrecondition, "CANNOT_OPEN")
	}
	return connect.NewResponse(&v1.OpenDataResponse{Plain: plain}), nil
}

func (s *service) GetPendingRestore(context.Context, *connect.Request[v1.GetPendingRestoreRequest]) (*connect.Response[v1.GetPendingRestoreResponse], error) {
	restore, err := s.fake.readPendingRestore()
	if err != nil {
		if connect.CodeOf(err) == connect.CodeInternal {
			return nil, err
		}
		return connect.NewResponse(&v1.GetPendingRestoreResponse{}), nil
	}
	return connect.NewResponse(&v1.GetPendingRestoreResponse{
		IsPending:     true,
		RootSignKey:   restore.recovery.Public().Sign,
		DeviceSignKey: restore.device.Public().Sign,
	}), nil
}

func (s *service) SignAsRecovery(_ context.Context, req *connect.Request[v1.SignAsRecoveryRequest]) (*connect.Response[v1.SignAsRecoveryResponse], error) {
	restore, err := s.fake.readPendingRestore()
	if err != nil {
		return nil, err
	}
	message := req.Msg.GetMessage()
	if len(message) == 0 || len(message) > 4096 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the message to sign has 1 to 4096 bytes"))
	}
	signature := restore.recovery.SignForRecovery(s.fake.pluginID, message)
	return connect.NewResponse(&v1.SignAsRecoveryResponse{Signature: signature[:]}), nil
}

func (s *service) CompleteRestore(_ context.Context, req *connect.Request[v1.CompleteRestoreRequest]) (*connect.Response[v1.CompleteRestoreResponse], error) {
	restore, err := s.fake.readPendingRestore()
	if err != nil {
		return nil, err
	}
	if s.fake.completeHook != nil {
		s.fake.completeHook()
	}
	state, err := joinRestore(restore, req.Msg.GetChain(), s.fake.now())
	if err != nil {
		return nil, err
	}
	s.fake.mu.Lock()
	defer s.fake.mu.Unlock()
	if s.fake.pending != restore {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_PENDING_RESTORE")
	}
	s.fake.record = &state
	s.fake.pending = nil
	return connect.NewResponse(&v1.CompleteRestoreResponse{}), nil
}

func joinRestore(restore *pendingRestore, chainBytes []byte, now time.Time) (fakeIdentity, error) {
	if len(chainBytes) == 0 {
		keys, err := identity.NewAccountKeys(rand.Reader)
		if err != nil {
			return fakeIdentity{}, connect.NewError(connect.CodeInternal, err)
		}
		chain, err := identity.NewChain(restore.recovery, keys, now)
		if err != nil {
			return fakeIdentity{}, connect.NewError(connect.CodeInternal, err)
		}
		return appendDevice(restore, chain, keys, now)
	}
	chain, err := identity.ParseChain(chainBytes)
	if err != nil || !bytes.Equal(chain.Root().Sign, restore.recovery.Public().Sign) {
		return fakeIdentity{}, refuse(connect.CodeInvalidArgument, "INVALID_CHAIN")
	}
	keys, err := restore.recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil {
		return fakeIdentity{}, refuse(connect.CodeInvalidArgument, "INVALID_CHAIN")
	}
	return appendDevice(restore, chain, keys, now)
}

func appendDevice(restore *pendingRestore, chain identity.Chain, keys identity.AccountKeys, now time.Time) (fakeIdentity, error) {
	chain, err := chain.AppendDevice(restore.recovery, restore.device.Public(), "Test device", now)
	if err != nil {
		return fakeIdentity{}, connect.NewError(connect.CodeInternal, err)
	}
	wrapped, err := keys.WrapForDevice(restore.device.Public())
	if err != nil {
		return fakeIdentity{}, connect.NewError(connect.CodeInternal, err)
	}
	return fakeIdentity{Record: identity.Record{Device: restore.device, Wrapped: wrapped}, Chain: chain, root: restore.recovery, keys: keys}, nil
}

func (s *service) OpenSealedBox(_ context.Context, req *connect.Request[v1.OpenSealedBoxRequest]) (*connect.Response[v1.OpenSealedBoxResponse], error) {
	keys, err := s.fake.readAccountKeys()
	if err != nil {
		return nil, err
	}
	plain, err := keys.OpenSealedBox(s.fake.pluginID, req.Msg.GetContext(), req.Msg.GetSealed())
	if err != nil {
		return nil, refuse(connect.CodeFailedPrecondition, "CANNOT_OPEN")
	}
	return connect.NewResponse(&v1.OpenSealedBoxResponse{Plain: plain}), nil
}
