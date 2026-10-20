package service

import (
	"errors"
	"fmt"
	"syscall"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/ports"
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

// describeCode is one error code in the words a tool answers with.
func describeCode(reason v1.ErrorCode) string {
	switch reason {
	case v1.ErrorCode_ERROR_CODE_MISSING:
		return "the vault holds no note at this path"
	case v1.ErrorCode_ERROR_CODE_NOT_A_NOTE:
		return "this is not a note the vault holds"
	case v1.ErrorCode_ERROR_CODE_NOT_TEXT:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case v1.ErrorCode_ERROR_CODE_TOO_LARGE:
		return fmt.Sprintf("it is larger than the %d bytes this reads; open the file instead", edit.MaxBytes)
	case v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE:
		return "this text opens with a frontmatter delimiter, so it cannot be written into a note"
	case v1.ErrorCode_ERROR_CODE_UNREADABLE:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	case v1.ErrorCode_ERROR_CODE_OCCUPIED:
		return "a file is already where this note would go"
	case v1.ErrorCode_ERROR_CODE_UNNAMEABLE:
		return "a note cannot be called this"
	case v1.ErrorCode_ERROR_CODE_NOT_A_STENCIL:
		return "this note is not a stencil"
	case v1.ErrorCode_ERROR_CODE_NOT_A_DECK:
		return "this note is not a deck"
	case v1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE:
		return "this deck is larger than a deck is read at"
	case v1.ErrorCode_ERROR_CODE_NOT_A_PRESET:
		return "this note is not a preset"
	case v1.ErrorCode_ERROR_CODE_STALE:
		return "this file is no longer the one that was read, and nothing was written: read it again and write from what is there now"
	default:
		return reason.String()
	}
}
