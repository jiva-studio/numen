package syncprotocol

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrUpdateRequired means the server needs another protocol version than the client has.
	ErrUpdateRequired = errors.New("the sync server needs another version of Numen")
	// ErrAudienceMismatch means the challenge names another server than the one the client dialed.
	ErrAudienceMismatch = errors.New("the challenge names another server")
	// ErrChallengeOutOfRange means the challenge expires too far from the clock of the client.
	ErrChallengeOutOfRange = errors.New("the challenge expires too far from the clock of this device")
	// ErrMalformedChallenge means the id or the nonce of the challenge has the wrong size.
	ErrMalformedChallenge = errors.New("the challenge is malformed")
	// ErrWriteNotAllowed means the server refused a write, for example because the plan of the account has none.
	ErrWriteNotAllowed = errors.New("the sync server does not allow this write")
	// ErrNoVault means the server has no such vault in the account.
	ErrNoVault = errors.New("the sync server has no such vault")
	// ErrVaultExists means the account has a vault with this id already.
	ErrVaultExists = errors.New("the vault exists on the sync server")
	// ErrAlreadySubscribed means the account has an active subscription, so there is nothing to buy.
	ErrAlreadySubscribed = errors.New("the account has an active subscription")
	// ErrBillingUnavailable means the sync server does not sell subscriptions.
	ErrBillingUnavailable = errors.New("the sync server does not sell subscriptions")
	// ErrNoActiveSubscription means a plan change was asked for an account without an active subscription bought by payment.
	ErrNoActiveSubscription = errors.New("the account has no active subscription to change")
	// ErrSamePlan means the account has the plan asked for already.
	ErrSamePlan = errors.New("the account has this plan already")
	// ErrUnknownPlan means the server sells no plan with this id.
	ErrUnknownPlan = errors.New("the sync server sells no such plan")
)

// QuotaExceededError means the account has no room for the bytes asked for.
type QuotaExceededError struct {
	Used  uint64
	Limit uint64
}

func (e QuotaExceededError) Error() string {
	return fmt.Sprintf("the sync quota is exceeded: %d of %d bytes used", e.Used, e.Limit)
}

// BlobNotReadyError means the blob of a revision of a batch was not uploaded. Index is the position of the
// revision in the batch.
type BlobNotReadyError struct {
	Index uint32
}

func (e BlobNotReadyError) Error() string {
	return fmt.Sprintf("the blob of revision %d is not uploaded", e.Index)
}

// refusalError reads as its sentinel and keeps the Connect error of the server, so connect.CodeOf still answers.
type refusalError struct {
	sentinel error
	cause    error
}

func (r refusalError) Error() string { return r.sentinel.Error() }

func (r refusalError) Unwrap() []error { return []error{r.sentinel, r.cause} }

// ErrBlobMissing means the store of the server no longer holds the body of a blob.
var ErrBlobMissing = errors.New("the sync server lost the body of a file")

// BlobMissingError names the blobs the server lost the bodies of. It reads as ErrBlobMissing and unwraps to the Connect error of the refusal.
type BlobMissingError struct {
	IDs   [][]byte
	cause error
}

func (e BlobMissingError) Error() string { return ErrBlobMissing.Error() }

func (e BlobMissingError) Is(target error) bool { return target == ErrBlobMissing }

func (e BlobMissingError) Unwrap() error { return e.cause }

// MultipartIncompleteError means the parts the storage holds are not the parts the upload needs. Each list holds
// the first 100 part numbers.
type MultipartIncompleteError struct {
	Missing   []uint32
	WrongSize []uint32
}

func (e MultipartIncompleteError) Error() string {
	return fmt.Sprintf("the multipart upload is incomplete: %d parts missing, %d of the wrong size", len(e.Missing), len(e.WrongSize))
}

// NewLinkRefusal returns an error that reads as the sentinel and keeps the Connect error of the relay.
func NewLinkRefusal(sentinel error, cause error) error {
	return refusalError{sentinel: sentinel, cause: cause}
}

var (
	// ErrNoShare means the server has no such share root, or the caller was never a member of it.
	ErrNoShare = errors.New("the sync server has no such share")
	// ErrShareExists means a share root with this id exists already.
	ErrShareExists = errors.New("the share exists on the sync server")
	// ErrReaderRefused means the server refused to add a reader. It gives no reason: the same answer covers a root no
	// account has, the owner itself, a member already and a share that holds the most readers, so the call does not reveal
	// whether an account has a root key.
	ErrReaderRefused = errors.New("the sync server refused to add the reader")
)

// ShareEndedError means the owner removed the caller from a share, or closed it. Version is the version of the
// entry that did.
type ShareEndedError struct {
	Version uint64
}

func (e ShareEndedError) Error() string {
	return fmt.Sprintf("the share ended at version %d", e.Version)
}

// RateLimitedError means the server refused download URLs of a share for now. RetryAfter is how long to wait.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e RateLimitedError) Error() string {
	return fmt.Sprintf("the share is rate limited for %s", e.RetryAfter)
}
