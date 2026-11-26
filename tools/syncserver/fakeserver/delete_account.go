package fakeserver

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// DeletionGrace is how long the server keeps an account that is deleting before it deletes it for good.
const DeletionGrace = 7 * 24 * time.Hour

func (s *Server) DeleteAccount(_ context.Context, req *connect.Request[syncv1.DeleteAccountRequest]) (*connect.Response[syncv1.DeleteAccountResponse], error) {
	session, err := s.authenticateSession(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isDeleteProven(session, req.Msg) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errNotSignedIn)
	}
	account := session.AccountFingerprint
	deleteAt, isDeleting := s.deleting[account]
	if !isDeleting {
		deleteAt = s.now().Add(DeletionGrace)
		s.deleting[account] = deleteAt
		delete(s.subscriptions, account)
		delete(s.planOf, account)
	}
	s.sessions = slices.DeleteFunc(s.sessions, func(one Session) bool { return one.AccountFingerprint == account })
	return connect.NewResponse(&syncv1.DeleteAccountResponse{DeleteAt: deleteAt.Unix()}), nil
}

func (s *Server) isDeleteProven(session Session, req *syncv1.DeleteAccountRequest) bool {
	pending, found := s.popChallenge(req.GetChallengeId())
	if !found || string(pending.deviceSignKey) != string(session.DeviceSignKey) || len(req.GetSignature()) != 64 {
		return false
	}
	message := syncprotocol.GetAccountDeleteMessage(s.getChallengeResponse(pending), session.DeviceSignKey, s.protocol)
	return identity.VerifyForPlugin(session.DeviceSignKey, syncprotocol.SignInPluginID, message, [64]byte(req.GetSignature()))
}

type exportedAccount struct {
	Account string          `json:"account"`
	Chain   []byte          `json:"chain"`
	Vaults  []exportedVault `json:"vaults"`
}

type exportedVault struct {
	ID    string `json:"id"`
	Files int    `json:"files"`
}

func (s *Server) ExportAccount(_ context.Context, req *connect.Request[syncv1.ExportAccountRequest], stream *connect.ServerStream[syncv1.ExportAccountChunk]) error {
	session, err := s.authenticateSession(req.Header())
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	account := session.AccountFingerprint
	exported := exportedAccount{Account: account, Chain: s.chains[account].Bytes(), Vaults: []exportedVault{}}
	for id, vault := range s.vaults {
		if vault.account == account {
			exported.Vaults = append(exported.Vaults, exportedVault{ID: hex.EncodeToString([]byte(id)), Files: len(vault.files)})
		}
	}
	slices.SortFunc(exported.Vaults, func(a, b exportedVault) int { return strings.Compare(a.ID, b.ID) })
	document, err := json.Marshal(exported)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	for chunk := range slices.Chunk(document, syncprotocol.MaxExportChunk) {
		if err := stream.Send(&syncv1.ExportAccountChunk{Data: chunk}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) applyDeletion(account string, restore bool) error {
	deleteAt, isDeleting := s.deleting[account]
	if !isDeleting {
		return nil
	}
	if !s.now().Before(deleteAt) {
		s.eraseAccount(account)
		return nil
	}
	if restore {
		delete(s.deleting, account)
		return nil
	}
	return newAccountDeleting(deleteAt)
}

func (s *Server) eraseAccount(account string) {
	delete(s.deleting, account)
	delete(s.chains, account)
	delete(s.catalogs, account)
	delete(s.used, account)
	for id, vault := range s.vaults {
		if vault.account == account {
			delete(s.vaults, id)
		}
	}
}

func newAccountDeleting(deleteAt time.Time) error {
	failure := newFailedSignIn()
	if detail, err := connect.NewErrorDetail(&syncv1.AccountDeleting{DeleteAt: deleteAt.Unix()}); err == nil {
		if refusal, ok := errors.AsType[*connect.Error](failure); ok {
			refusal.AddDetail(detail)
		}
	}
	return failure
}
