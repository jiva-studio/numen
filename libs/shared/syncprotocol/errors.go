package syncprotocol

import "errors"

var (
	// ErrUpdateRequired means the server needs another protocol version than the client has.
	ErrUpdateRequired = errors.New("the sync server needs another version of Numen")
	// ErrAudienceMismatch means the challenge names another server than the one the client dialed.
	ErrAudienceMismatch = errors.New("the challenge names another server")
	// ErrChallengeOutOfRange means the challenge expires too far from the clock of the client.
	ErrChallengeOutOfRange = errors.New("the challenge expires too far from the clock of this device")
	// ErrMalformedChallenge means the id or the nonce of the challenge has the wrong size.
	ErrMalformedChallenge = errors.New("the challenge is malformed")
)
