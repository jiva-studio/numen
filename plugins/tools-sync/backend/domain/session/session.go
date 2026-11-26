// Package session is the sign-in of this device to the sync server: the identity it signs in with, the
// challenge it signs, and the session the server gives back.
package session

import (
	"errors"
	"time"
)

var (
	// ErrNoIdentity means this device holds no identity yet.
	ErrNoIdentity = errors.New("no identity on this device")
	// ErrUpdateRequired means the server needs another protocol version.
	ErrUpdateRequired = errors.New("the sync server needs another version of Numen")
	// ErrUnauthenticated means the server refused the sign-in.
	ErrUnauthenticated = errors.New("the sync server refused the sign-in")
	// ErrUnreachable means the server did not answer.
	ErrUnreachable = errors.New("the sync server could not be reached")
	// ErrNoServer means no sync server address is set.
	ErrNoServer = errors.New("no sync server is set")
)

// DeviceRevokedError means the server refused the sign-in because the account revoked this device. Chain is the chain the
// server stores, which says so.
type DeviceRevokedError struct{ Chain []byte }

func (DeviceRevokedError) Error() string { return "this device was revoked" }

// Identity is what is public of the identity of this device.
type Identity struct {
	AccountFingerprint string
	DeviceSignKey      []byte
	Chain              []byte
	KeyEpoch           uint32
	ChainLength        uint32
	IsRevoked          bool
}

// Challenge is what the server asks this device to sign: its id, and the bytes to sign.
type Challenge struct {
	ID      []byte
	Message []byte
}

// Proof is what the server checks to give a session.
type Proof struct {
	ChallengeID   []byte
	DeviceSignKey []byte
	Signature     []byte
	Chain         []byte
}

// Session is a session of the server. It lives in memory only and prints as [session], so its token
// reaches no log.
type Session struct {
	Token              string
	ExpiresAt          time.Time
	AccountFingerprint string
}

func (Session) String() string { return "[session]" }

func (s Session) GoString() string { return s.String() }
