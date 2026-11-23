package syncprotocol

import (
	"errors"

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
// ErrNoVault, ErrVaultExists, QuotaExceededError and BlobNotReadyError, keeping the Connect error where there is one.
func ReadCallError(err error) error {
	err = ReadError(err)
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return err
	}
	switch failure.Code() {
	case connect.CodePermissionDenied:
		return refusalError{sentinel: ErrWriteNotAllowed, cause: failure}
	case connect.CodeNotFound:
		return refusalError{sentinel: ErrNoVault, cause: failure}
	case connect.CodeAlreadyExists:
		return refusalError{sentinel: ErrVaultExists, cause: failure}
	}
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		switch refusal := value.(type) {
		case *syncv1.QuotaExceeded:
			return QuotaExceededError{Used: refusal.GetUsedBytes(), Limit: refusal.GetLimitBytes()}
		case *syncv1.BlobNotReady:
			return BlobNotReadyError{Index: refusal.GetRevisionIndex()}
		}
	}
	return err
}
