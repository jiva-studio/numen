// Package server is the sync server, reached over Connect.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
)

// Config is where the server is and what the adapter reaches it with.
type Config struct {
	// Address is the address of the server, such as https://sync.numen.md.
	Address string
	HTTP    *http.Client
	// Now reads the clock a challenge is checked against.
	Now func() time.Time
}

// Server is the sync server at an address.
type Server struct {
	address string
	now     func() time.Time
	client  syncv1connect.SignInServiceClient
}

// New returns the server at the address of the config.
func New(cfg Config) *Server {
	return &Server{
		address: cfg.Address, now: cfg.Now,
		client: syncv1connect.NewSignInServiceClient(cfg.HTTP, cfg.Address, connect.WithInterceptors(syncprotocol.NewVersionInterceptor())),
	}
}

// GetChallenge asks for a challenge. It returns one only if it names the address dialled, expires near
// the clock of this device and has the sizes of the protocol; any other is an error.
func (s *Server) GetChallenge(ctx context.Context, deviceSignKey []byte) (session.Challenge, error) {
	response, err := s.client.GetChallenge(ctx, connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: deviceSignKey}))
	if err != nil {
		return session.Challenge{}, readError(err)
	}
	if err := syncprotocol.CheckChallenge(response.Msg, s.address, s.now()); err != nil {
		return session.Challenge{}, err
	}
	return session.Challenge{
		ID:      response.Msg.GetChallengeId(),
		Message: syncprotocol.GetSignInMessage(response.Msg, deviceSignKey, syncprotocol.Version),
	}, nil
}

// SignIn sends the proof and returns the session the server gives.
func (s *Server) SignIn(ctx context.Context, proof session.Proof) (session.Session, error) {
	response, err := s.client.SignIn(ctx, connect.NewRequest(&syncv1.SignInRequest{
		ChallengeId: proof.ChallengeID, DeviceSignKey: proof.DeviceSignKey, Signature: proof.Signature, Chain: proof.Chain,
	}))
	if err != nil {
		return session.Session{}, readError(err)
	}
	return session.Session{
		Token:              response.Msg.GetSessionToken(),
		ExpiresAt:          time.Unix(response.Msg.GetExpiresAt(), 0),
		AccountFingerprint: response.Msg.GetAccountFingerprint(),
	}, nil
}

func readError(err error) error {
	switch {
	case errors.Is(err, syncprotocol.ErrUpdateRequired):
		return session.ErrUpdateRequired
	case connect.CodeOf(err) == connect.CodeUnauthenticated:
		return session.ErrUnauthenticated
	case connect.CodeOf(err) == connect.CodeUnavailable, connect.CodeOf(err) == connect.CodeDeadlineExceeded:
		return session.ErrUnreachable
	}
	return err
}
