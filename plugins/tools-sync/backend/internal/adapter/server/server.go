// Package server is the sync server, reached through the protocol library.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
)

// Config is where the server is (an address such as https://sync.numen.md), what the adapter reaches it
// with and the clock a challenge is checked against.
type Config struct {
	Address string
	HTTP    *http.Client
	Now     func() time.Time
}

// Server is the sync server at an address.
type Server struct {
	client  *syncclient.SignInClient
	account *syncclient.AccountClient
}

// New returns the server at the address of the config.
func New(cfg Config) *Server {
	return &Server{
		client:  syncclient.NewSignInClient(cfg.Address, cfg.HTTP, cfg.Now),
		account: syncclient.NewAccountClient(cfg.Address, cfg.HTTP, func() string { return "" }),
	}
}

// GetChallenge asks for a challenge. It returns one only if it names the address dialed, expires near
// the clock of this device and has the sizes of the protocol; any other is an error.
func (s *Server) GetChallenge(ctx context.Context, deviceSignKey []byte) (session.Challenge, error) {
	challenge, err := s.client.GetChallenge(ctx, deviceSignKey)
	if err != nil {
		return session.Challenge{}, readError(err)
	}
	return session.Challenge{ID: challenge.ID, Message: challenge.Message}, nil
}

// SignIn sends the proof and returns the session the server gives.
func (s *Server) SignIn(ctx context.Context, proof session.Proof) (session.Session, error) {
	opened, err := s.client.SignIn(ctx, syncclient.Proof{
		ChallengeID: proof.ChallengeID, DeviceSignKey: proof.DeviceSignKey, Signature: proof.Signature, Chain: proof.Chain, RestoreAccount: proof.RestoreAccount,
	})
	if err != nil {
		return session.Session{}, readError(err)
	}
	return session.Session{Token: opened.Token, ExpiresAt: opened.ExpiresAt, AccountFingerprint: opened.AccountFingerprint}, nil
}

func readError(err error) error {
	if deleting, ok := errors.AsType[syncprotocol.AccountDeletingError](err); ok {
		return session.AccountDeletingError{DeleteAt: deleting.DeleteAt}
	}
	if revoked, ok := errors.AsType[syncprotocol.DeviceRevokedError](err); ok {
		return session.DeviceRevokedError{Chain: revoked.Chain}
	}
	switch {
	case errors.Is(err, syncprotocol.ErrUpdateRequired):
		return session.ErrUpdateRequired
	case errors.Is(err, syncprotocol.ErrUnauthenticated):
		return session.ErrUnauthenticated
	case errors.Is(err, syncprotocol.ErrUnreachable):
		return session.ErrUnreachable
	}
	return err
}

// GetChainReadChallenge asks for a challenge of the root key, with the checks of GetChallenge. Its message is the
// one the recovery key signs to read the chain of its account.
func (s *Server) GetChainReadChallenge(ctx context.Context, rootSignKey []byte) (session.Challenge, error) {
	challenge, err := s.client.GetChainReadChallenge(ctx, rootSignKey)
	if err != nil {
		return session.Challenge{}, readError(err)
	}
	return session.Challenge{ID: challenge.ID, Message: challenge.Message}, nil
}

// GetAccountChainByRoot returns the chain the server stores for the account of the root key, and false when it
// has no such account.
func (s *Server) GetAccountChainByRoot(ctx context.Context, challengeID, rootSignKey, signature []byte) ([]byte, bool, error) {
	chain, found, err := s.account.GetAccountChainByRoot(ctx, challengeID, rootSignKey, signature)
	return chain, found, readError(err)
}

// GetAccountDeleteChallenge asks for a challenge of the device key, with the checks of GetChallenge. Its message is the
// one the device signs to delete its account.
func (s *Server) GetAccountDeleteChallenge(ctx context.Context, deviceSignKey []byte) (session.Challenge, error) {
	challenge, err := s.client.GetAccountDeleteChallenge(ctx, deviceSignKey)
	if err != nil {
		return session.Challenge{}, readError(err)
	}
	return session.Challenge{ID: challenge.ID, Message: challenge.Message}, nil
}
