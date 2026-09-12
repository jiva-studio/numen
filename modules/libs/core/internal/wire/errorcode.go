package wire

import (
	"errors"
	"syscall"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// An outcome or a write's error as the error code the schema carries.
//
// Every window that shows a note it could not read, and every window that could
// not write one, names the reason the same way: the translation is written once
// and not one per window.

// ErrorCodeOf says which error code an outcome is, and whether it is one at all.
func ErrorCodeOf(o note.ReadOutcome) (v1.ErrorCode, bool) {
	switch o {
	case note.Missing:
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	case note.NotANote:
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	case note.NotText:
		return v1.ErrorCode_ERROR_CODE_NOT_TEXT, true
	case note.TooLarge:
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case note.Unreadable:
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	default:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
}

// ErrorCodeBy says which error code a write's error is, and whether it is one at
// all. Anything else is the vault being out of reach.
func ErrorCodeBy(err error) (v1.ErrorCode, bool) {
	switch {
	case errors.Is(err, port.ErrStale):
		return v1.ErrorCode_ERROR_CODE_STALE, true
	case errors.Is(err, note.ErrNoNote):
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	case errors.Is(err, note.ErrTooLarge):
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case errors.Is(err, note.ErrUnnameable):
		return v1.ErrorCode_ERROR_CODE_UNNAMEABLE, true
	case errors.Is(err, note.ErrUnreadable),
		errors.Is(err, note.ErrInline),
		errors.Is(err, note.ErrUnterminated),
		errors.Is(err, note.ErrAnchored):
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, note.ErrBodyUnwritable):
		return v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE, true
	case errors.Is(err, port.ErrNotANote):
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	// A file standing where a folder of the path must be is a file in the way,
	// the same as a file standing where the note itself would go.
	case errors.Is(err, port.ErrOccupied), errors.Is(err, syscall.ENOTDIR):
		return v1.ErrorCode_ERROR_CODE_OCCUPIED, true
	default:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
}

// GetCode is the code an error the schema carries no error code for answers
// with. A path that does not stay in the vault is the client's to correct;
// anything else is the vault being out of reach.
func GetCode(err error) connect.Code {
	if errors.Is(err, port.ErrOutside) {
		return connect.CodeInvalidArgument
	}
	return connect.CodeInternal
}
