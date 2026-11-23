package syncprotocol

import (
	"errors"
	"fmt"
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
