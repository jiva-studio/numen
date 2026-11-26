// Package client is the Go client of the sync server: the sign-in, catalog, revision, blob and account services and
// the transfer of a blob over a presigned URL. It reaches the network, so a use case does not import it.
package client

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// Challenge is a challenge the client has checked: its id and the bytes to sign.
type Challenge struct {
	ID      []byte
	Message []byte
}

// Proof is what answers a challenge: the signature over its message, with the key and the chain that
// vouch for the device.
type Proof struct {
	ChallengeID    []byte
	DeviceSignKey  []byte
	Signature      []byte
	Chain          []byte
	RestoreAccount bool
}

// Session is the session the server opened for a sign-in.
type Session struct {
	Token              string
	ExpiresAt          time.Time
	AccountFingerprint string
}

// SignInClient calls the sign-in service of one server.
type SignInClient struct {
	address string
	now     func() time.Time
	rpc     syncv1connect.SignInServiceClient
}

// NewSignInClient returns the client of the server at address, such as https://sync.numen.md. now is the
// clock a challenge is checked against.
func NewSignInClient(address string, httpClient *http.Client, now func() time.Time) *SignInClient {
	return &SignInClient{
		address: address, now: now,
		rpc: syncv1connect.NewSignInServiceClient(httpClient, address, connect.WithInterceptors(syncprotocol.NewVersionInterceptor())),
	}
}

// GetChallenge asks for a challenge of the device key. It returns one only if it names the address of the
// client, expires near the clock and has the sizes of the protocol; any other is an error. A server of
// another version gives ErrUpdateRequired, one that is not there ErrUnreachable.
func (c *SignInClient) GetChallenge(ctx context.Context, deviceSignKey []byte) (Challenge, error) {
	return c.getChallenge(ctx, deviceSignKey, syncprotocol.GetSignInMessage)
}

// GetChainReadChallenge asks for a challenge of the root key, with the checks of GetChallenge. The message
// of the challenge is the one the recovery key signs to read the chain of its account.
func (c *SignInClient) GetChainReadChallenge(ctx context.Context, rootSignKey []byte) (Challenge, error) {
	return c.getChallenge(ctx, rootSignKey, syncprotocol.GetChainReadMessage)
}

// GetAccountDeleteChallenge runs the checks of GetChallenge. A signature over its message deletes the account and
// never signs in.
func (c *SignInClient) GetAccountDeleteChallenge(ctx context.Context, deviceSignKey []byte) (Challenge, error) {
	return c.getChallenge(ctx, deviceSignKey, syncprotocol.GetAccountDeleteMessage)
}

// GetVaultDeleteChallenge runs the checks of GetChallenge. A signature over its message deletes the vault and nothing else.
func (c *SignInClient) GetVaultDeleteChallenge(ctx context.Context, deviceSignKey, vaultID []byte) (Challenge, error) {
	return c.getChallenge(ctx, deviceSignKey, func(challenge *syncv1.GetChallengeResponse, key []byte, version uint32) []byte {
		return syncprotocol.GetVaultDeleteMessage(challenge, key, vaultID, version)
	})
}

func (c *SignInClient) getChallenge(ctx context.Context, deviceSignKey []byte, getMessage func(*syncv1.GetChallengeResponse, []byte, uint32) []byte) (Challenge, error) {
	response, err := c.rpc.GetChallenge(ctx, connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: deviceSignKey}))
	if err != nil {
		return Challenge{}, syncprotocol.ReadError(err)
	}
	if err := syncprotocol.CheckChallenge(response.Msg, c.address, c.now()); err != nil {
		return Challenge{}, err
	}
	return Challenge{ID: response.Msg.GetChallengeId(), Message: getMessage(response.Msg, deviceSignKey, syncprotocol.Version)}, nil
}

// SignIn sends the proof and returns the session the server opens. A refusal is ErrUnauthenticated.
func (c *SignInClient) SignIn(ctx context.Context, proof Proof) (Session, error) {
	response, err := c.rpc.SignIn(ctx, connect.NewRequest(&syncv1.SignInRequest{
		ChallengeId: proof.ChallengeID, DeviceSignKey: proof.DeviceSignKey, Signature: proof.Signature, Chain: proof.Chain, RestoreAccount: proof.RestoreAccount,
	}))
	if err != nil {
		return Session{}, syncprotocol.ReadError(err)
	}
	return Session{
		Token:              response.Msg.GetSessionToken(),
		ExpiresAt:          time.Unix(response.Msg.GetExpiresAt(), 0),
		AccountFingerprint: response.Msg.GetAccountFingerprint(),
	}, nil
}
