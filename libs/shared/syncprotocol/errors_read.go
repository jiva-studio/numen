package syncprotocol

import (
	"errors"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

var (
	// ErrUnauthenticated means the server refused the sign-in.
	ErrUnauthenticated = errors.New("the sync server refused the sign-in")
	// ErrUnreachable means the server did not answer in time or was not there.
	ErrUnreachable = errors.New("the sync server is unreachable")
)

// ReadError maps the failure of a call to the server to the errors of the protocol: a server of another version,
// a refused sign-in, a server that is not there. Any other failure is returned as it came.
func ReadError(err error) error {
	if revoked, ok := readDeviceRevoked(err); ok {
		return revoked
	}
	if deleting, ok := readAccountDeleting(err); ok {
		return deleting
	}
	switch {
	case errors.Is(err, ErrUpdateRequired):
		return ErrUpdateRequired
	case connect.CodeOf(err) == connect.CodeUnauthenticated:
		return ErrUnauthenticated
	case connect.CodeOf(err) == connect.CodeUnavailable, connect.CodeOf(err) == connect.CodeDeadlineExceeded:
		return ErrUnreachable
	}
	return err
}

// ReadCallError is ReadError for the calls of a session: it also maps the refusals of the server to ErrWriteNotAllowed,
// ErrNoVault, ErrVaultExists, QuotaExceededError, BlobNotReadyError, BlobMissingError and MultipartIncompleteError, keeping the Connect error where there is one.
func ReadCallError(err error) error {
	err = ReadError(err)
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return err
	}
	if detailed := readDetail(failure); detailed != nil {
		return detailed
	}
	switch failure.Code() {
	case connect.CodePermissionDenied:
		return refusalError{sentinel: ErrWriteNotAllowed, cause: failure}
	case connect.CodeNotFound:
		return refusalError{sentinel: ErrNoVault, cause: failure}
	case connect.CodeAlreadyExists:
		return refusalError{sentinel: ErrVaultExists, cause: failure}
	}
	return err
}

func readDetail(failure *connect.Error) error {
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		switch refusal := value.(type) {
		case *syncv1.ChainStale:
			return refusalError{sentinel: ChainStaleError{Head: refusal.GetChain()}, cause: failure}
		case *syncv1.QuotaExceeded:
			return QuotaExceededError{Used: refusal.GetUsedBytes(), Limit: refusal.GetLimitBytes()}
		case *syncv1.BlobNotReady:
			return BlobNotReadyError{Index: refusal.GetRevisionIndex()}
		case *syncv1.BlobMissing:
			return BlobMissingError{IDs: refusal.GetBlobIds(), cause: failure}
		case *syncv1.MultipartIncomplete:
			return MultipartIncompleteError{Missing: refusal.GetMissing(), WrongSize: refusal.GetWrongSize()}
		}
	}
	return nil
}

// ReadSubscriptionError is ReadError for the calls of the subscription service: it also maps an account that has a
// subscription to ErrAlreadySubscribed and a server that sells none to ErrBillingUnavailable.
func ReadSubscriptionError(err error) error {
	if failure, ok := errors.AsType[*connect.Error](err); ok {
		switch failure.Code() {
		case connect.CodeAlreadyExists:
			return refusalError{sentinel: ErrAlreadySubscribed, cause: failure}
		case connect.CodeUnimplemented:
			return refusalError{sentinel: ErrBillingUnavailable, cause: failure}
		case connect.CodeInvalidArgument:
			return refusalError{sentinel: ErrUnknownPlan, cause: failure}
		}
	}
	return ReadError(err)
}

// ReadChangePlanError is ReadError for the call of ChangePlan: a server of another version stays ErrUpdateRequired, and it maps FAILED_PRECONDITION to ErrNoActiveSubscription,
// ALREADY_EXISTS to ErrSamePlan, INVALID_ARGUMENT to ErrUnknownPlan and a server that sells none to ErrBillingUnavailable.
func ReadChangePlanError(err error) error {
	if errors.Is(err, ErrUpdateRequired) {
		return ErrUpdateRequired
	}
	if failure, ok := errors.AsType[*connect.Error](err); ok {
		switch failure.Code() {
		case connect.CodeFailedPrecondition:
			return refusalError{sentinel: ErrNoActiveSubscription, cause: failure}
		case connect.CodeAlreadyExists:
			return refusalError{sentinel: ErrSamePlan, cause: failure}
		case connect.CodeInvalidArgument:
			return refusalError{sentinel: ErrUnknownPlan, cause: failure}
		case connect.CodeUnimplemented:
			return refusalError{sentinel: ErrBillingUnavailable, cause: failure}
		}
	}
	return ReadError(err)
}

func readDeviceRevoked(err error) (error, bool) {
	failure, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return nil, false
	}
	for _, detail := range failure.Details() {
		if value, valueErr := detail.Value(); valueErr == nil {
			if revoked, isRevoked := value.(*syncv1.DeviceRevoked); isRevoked {
				return refusalError{sentinel: DeviceRevokedError{Chain: revoked.GetChain()}, cause: failure}, true
			}
		}
	}
	return nil, false
}

// ReadShareError is ReadCallError for the calls of the share service: it maps NOT_FOUND to ErrNoShare, ALREADY_EXISTS
// to ErrShareExists, and the details ShareEnded and RateLimited to ShareEndedError and RateLimitedError.
func ReadShareError(err error) error {
	err = ReadError(err)
	failure, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return err
	}
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		switch refusal := value.(type) {
		case *syncv1.ShareEnded:
			return ShareEndedError{Version: refusal.GetVersion()}
		case *syncv1.RateLimited:
			return RateLimitedError{RetryAfter: time.Duration(refusal.GetRetryAfterSeconds()) * time.Second}
		}
	}
	switch failure.Code() {
	case connect.CodeNotFound:
		return refusalError{sentinel: ErrNoShare, cause: failure}
	case connect.CodeAlreadyExists:
		return refusalError{sentinel: ErrShareExists, cause: failure}
	}
	return ReadCallError(err)
}

func readAccountDeleting(err error) (error, bool) {
	failure, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return nil, false
	}
	for _, detail := range failure.Details() {
		if value, valueErr := detail.Value(); valueErr == nil {
			if deleting, isDeleting := value.(*syncv1.AccountDeleting); isDeleting {
				return refusalError{sentinel: AccountDeletingError{DeleteAt: time.Unix(deleting.GetDeleteAt(), 0)}, cause: failure}, true
			}
		}
	}
	return nil, false
}
