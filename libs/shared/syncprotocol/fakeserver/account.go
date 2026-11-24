package fakeserver

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"slices"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *Server) GetAccountChain(_ context.Context, req *connect.Request[syncv1.GetAccountChainRequest]) (*connect.Response[syncv1.GetAccountChainResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return connect.NewResponse(&syncv1.GetAccountChainResponse{Chain: s.chains[account].Bytes()}), nil
}

func (s *Server) GetAccountChainByRoot(_ context.Context, req *connect.Request[syncv1.GetAccountChainByRootRequest]) (*connect.Response[syncv1.GetAccountChainByRootResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isChainReadProven(req.Msg) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errNotSignedIn)
	}
	account := identity.RecoveryPublicKey{Sign: req.Msg.GetRootSignKey()}.Fingerprint().String()
	chain, found := s.chains[account]
	if !found {
		return connect.NewResponse(&syncv1.GetAccountChainByRootResponse{}), nil
	}
	return connect.NewResponse(&syncv1.GetAccountChainByRootResponse{Found: true, Chain: chain.Bytes()}), nil
}

func (s *Server) isChainReadProven(req *syncv1.GetAccountChainByRootRequest) bool {
	pending, found := s.popChallenge(req.GetChallengeId())
	root := req.GetRootSignKey()
	if !found || len(root) != ed25519.PublicKeySize || string(pending.deviceSignKey) != string(root) || len(req.GetSignature()) != 64 {
		return false
	}
	message := syncprotocol.GetChainReadMessage(s.getChallengeResponse(pending), root, s.protocol)
	return identity.VerifyForRecovery(root, syncprotocol.SignInPluginID, message, [64]byte(req.GetSignature()))
}

func isRevoked(chain identity.Chain, deviceSignKey []byte) bool {
	return chain.IsKnownDevice(deviceSignKey) && !chain.IsActiveDevice(deviceSignKey)
}

func (s *Server) AppendChain(_ context.Context, req *connect.Request[syncv1.AppendChainRequest]) (*connect.Response[syncv1.AppendChainResponse], error) {
	session, err := s.authenticateSession(req.Header())
	if err != nil {
		return nil, err
	}
	presented, err := identity.ParseChain(req.Msg.GetChain())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.chains[session.AccountFingerprint]
	if err := checkAppend(stored, presented, session); err != nil {
		return nil, err
	}
	s.applyChain(session.AccountFingerprint, presented)
	return connect.NewResponse(&syncv1.AppendChainResponse{Accepted: true, Chain: s.chains[session.AccountFingerprint].Bytes()}), nil
}

func checkAppend(stored, presented identity.Chain, session Session) error {
	if presented.Root().Fingerprint().String() != session.AccountFingerprint {
		return connect.NewError(connect.CodeInvalidArgument, errInvalidRequest)
	}
	if !presented.IsActiveDevice(session.DeviceSignKey) {
		return connect.NewError(connect.CodePermissionDenied, errRevokedAuthor)
	}
	if !presented.IsExtensionOf(stored) {
		return newChainStale(stored)
	}
	for _, signer := range presented.GetSignersSince(stored) {
		if !bytes.Equal(signer, session.DeviceSignKey) && !bytes.Equal(signer, presented.Root().Sign) {
			return connect.NewError(connect.CodePermissionDenied, errInvalidRequest)
		}
	}
	return nil
}

func newChainStale(head identity.Chain) error {
	failure := connect.NewError(connect.CodeFailedPrecondition, errChainStale)
	if detail, err := connect.NewErrorDetail(&syncv1.ChainStale{Chain: head.Bytes()}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func (s *Server) applyChain(account string, chain identity.Chain) {
	stored := s.chains[account]
	s.chains[account] = chain
	for _, revoked := range chain.GetRevokedSince(stored) {
		s.sessions = slices.DeleteFunc(s.sessions, func(one Session) bool {
			return one.AccountFingerprint == account && bytes.Equal(one.DeviceSignKey, revoked)
		})
	}
}
