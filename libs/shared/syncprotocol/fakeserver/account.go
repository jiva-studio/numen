package fakeserver

import (
	"context"
	"crypto/ed25519"

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
