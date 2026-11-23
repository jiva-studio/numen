package syncprotocol

import (
	"errors"
	"time"
)

var (
	// ErrChannelGone means the link channel is unknown, expired or closed; the server does not say which.
	ErrChannelGone = errors.New("the link channel is not there")
	// ErrLinkBusy means another wait on the same side of the channel is running.
	ErrLinkBusy = errors.New("the link channel is already waited on")
	// ErrRateLimited means the server refused the call to keep the limits of the relay.
	ErrRateLimited = errors.New("the sync server limits the link calls")
)

// ChainStaleError means the chain presented does not extend the chain the server stores. Head is the stored chain.
type ChainStaleError struct{ Head []byte }

func (ChainStaleError) Error() string { return "the sync server holds a newer chain" }

// DeviceRevokedError means the device proved its key but the account revoked it. Chain is the stored chain.
type DeviceRevokedError struct{ Chain []byte }

func (DeviceRevokedError) Error() string { return "this device was revoked" }

// AccountDeletingError means the account is deleting: the server refused the sign-in and stops syncing it until it is
// restored. DeleteAt is when the server deletes it for good.
type AccountDeletingError struct{ DeleteAt time.Time }

func (AccountDeletingError) Error() string { return "the account is being deleted" }
