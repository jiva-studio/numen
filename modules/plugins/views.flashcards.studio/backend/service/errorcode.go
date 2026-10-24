package service

import (
	"errors"
	"syscall"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

// An outcome or a write's error as the error code the schema carries.

// convertOutcomeToErrorCode says which error code an outcome is, and whether it is one at all.
func convertOutcomeToErrorCode(o edit.ReadOutcome) (v1.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	case edit.NotANote:
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	case edit.NotText:
		return v1.ErrorCode_ERROR_CODE_NOT_TEXT, true
	case edit.TooLarge:
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case edit.Unreadable:
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case edit.Ok:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
	return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
}

// getErrorCode says which error code a write's error is, and whether it is one at all. Anything
// else is the vault being out of reach.
func getErrorCode(err error) (v1.ErrorCode, bool) {
	switch {
	case errors.Is(err, ports.ErrStale):
		return v1.ErrorCode_ERROR_CODE_STALE, true
	case errors.Is(err, edit.ErrNoNote):
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	case errors.Is(err, edit.ErrTooLarge):
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case errors.Is(err, domain.ErrUnnameable):
		return v1.ErrorCode_ERROR_CODE_UNNAMEABLE, true
	case errors.Is(err, edit.ErrUnreadable),
		errors.Is(err, edit.ErrInline),
		errors.Is(err, edit.ErrUnterminated),
		errors.Is(err, edit.ErrAnchored):
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, edit.ErrBodyUnwritable):
		return v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE, true
	case errors.Is(err, ports.ErrNotANote):
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	// A file standing where a folder of the path must be is a file in the way, the same as a file
	// standing where the note itself would go.
	case errors.Is(err, ports.ErrOccupied), errors.Is(err, syscall.ENOTDIR):
		return v1.ErrorCode_ERROR_CODE_OCCUPIED, true
	}
	return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
}

// getCode is the code an error the schema carries no error code for answers with. A path that does
// not stay in the vault is the client's to correct; anything else is the vault being out of reach.
func getCode(err error) connect.Code {
	if errors.Is(err, host.ErrOutside) {
		return connect.CodeInvalidArgument
	}
	return connect.CodeInternal
}

// isUnlevelled says whether a write reached the vault and the index did not follow.
func isUnlevelled(err error) bool { return errors.Is(err, edit.ErrUnlevelled) }
