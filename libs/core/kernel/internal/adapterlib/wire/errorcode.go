package wire

import (
	"errors"
	"syscall"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// An outcome or a write's error as the error code the schema carries.
//
// Every window that shows a note it could not read, and every window that could
// not write one, names the reason the same way: the translation is written once
// and not one per window.

func ConvertOutcomeToErrorCode(o edit.ReadOutcome) (v1.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return v1.ErrorCode_ERROR_CODE_NOT_FOUND, true
	case edit.NotANote:
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	case edit.NotText:
		return v1.ErrorCode_ERROR_CODE_NOT_TEXT, true
	case edit.TooLarge:
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case edit.Unreadable:
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	default:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
}

// GetErrorCode returns which error code a write's error is, and whether it is one at
// all. Anything else is the vault being unreachable.
func GetErrorCode(err error) (v1.ErrorCode, bool) {
	switch {
	case errors.Is(err, edit.ErrStale):
		return v1.ErrorCode_ERROR_CODE_CONFLICT, true
	case errors.Is(err, edit.ErrNoNote):
		return v1.ErrorCode_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, edit.ErrTooLarge):
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case errors.Is(err, domain.ErrInvalidName):
		return v1.ErrorCode_ERROR_CODE_INVALID_NAME, true
	case errors.Is(err, edit.ErrUnreadable),
		errors.Is(err, edit.ErrInline),
		errors.Is(err, edit.ErrUnterminated),
		errors.Is(err, edit.ErrAnchored):
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, edit.ErrBodyUnwritable):
		return v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE, true
	case errors.Is(err, edit.ErrNotANote):
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	// A file standing where a folder of the path must be is a file in the way,
	// the same as a file standing where the note itself would go.
	case errors.Is(err, vaultdomain.ErrAlreadyExists), errors.Is(err, syscall.ENOTDIR):
		return v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS, true
	default:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
}

// GetCode is the code an error the schema carries no error code for answers
// with. A path that does not stay in the vault is the client's to correct;
// anything else is the vault being out of reach.
func GetCode(err error) connect.Code {
	if errors.Is(err, vaultdomain.ErrOutside) {
		return connect.CodeInvalidArgument
	}
	return connect.CodeInternal
}
