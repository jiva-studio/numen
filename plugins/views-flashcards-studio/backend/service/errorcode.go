package service

import (
	"errors"
	"syscall"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Conversions from an outcome or a write error to the ErrorCode of a response.

// convertOutcomeToErrorCode returns the error code of an outcome, and whether the outcome is an error.
func convertOutcomeToErrorCode(o edit.ReadOutcome) (api.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return api.ErrorCodeNotFound, true
	case edit.NotANote:
		return api.ErrorCodeNotANote, true
	case edit.NotText:
		return api.ErrorCodeNotText, true
	case edit.TooLarge:
		return api.ErrorCodeTooLarge, true
	case edit.Unreadable:
		return api.ErrorCodeUnreadable, true
	case edit.Ok:
		// gomutants:disable-next-line BRANCH_CASE,RETURN_ZERO reason="Ok shares the fallback return and ErrorCodeUnspecified is the zero value."
		return api.ErrorCodeUnspecified, false
	}
	// gomutants:disable-next-line RETURN_ZERO reason="ErrorCodeUnspecified is the zero value of ErrorCode."
	return api.ErrorCodeUnspecified, false
}

// getErrorCode returns the error code of a write error, and whether the error
// has one. Any other error means the vault is unavailable.
func getErrorCode(err error) (api.ErrorCode, bool) {
	switch {
	case errors.Is(err, ports.ErrStale):
		return api.ErrorCodeConflict, true
	case errors.Is(err, edit.ErrNoNote):
		return api.ErrorCodeNotFound, true
	case errors.Is(err, edit.ErrTooLarge):
		return api.ErrorCodeTooLarge, true
	case errors.Is(err, domain.ErrInvalidName):
		return api.ErrorCodeInvalidName, true
	case errors.Is(err, markdown.ErrUnreadable),
		errors.Is(err, markdown.ErrInline),
		errors.Is(err, markdown.ErrUnterminated),
		errors.Is(err, edit.ErrAnchored):
		return api.ErrorCodeUnreadable, true
	case errors.Is(err, edit.ErrBodyUnwritable):
		return api.ErrorCodeBodyUnwritable, true
	case errors.Is(err, ports.ErrNotANote):
		return api.ErrorCodeNotANote, true
	// A file standing where a folder of the path must be is a file in the way, the same as a file
	// standing where the note itself would go.
	case errors.Is(err, ports.ErrAlreadyExists), errors.Is(err, syscall.ENOTDIR):
		return api.ErrorCodeAlreadyExists, true
	}
	// gomutants:disable-next-line RETURN_ZERO reason="ErrorCodeUnspecified is the zero value of ErrorCode."
	return api.ErrorCodeUnspecified, false
}

// getCode returns the API error code of an error that has no ErrorCode. A path outside the vault
// is invalid_argument; any other error is internal.
func getCode(err error) server.ErrorCode {
	if errors.Is(err, host.ErrOutside) {
		return server.CodeInvalidArgument
	}
	return server.CodeInternal
}

// newError creates the API error that answers a method failing with err under code.
func newError(code server.ErrorCode, err error) error {
	return server.NewAPIError(code, err.Error())
}

// isIndexOutdated reports whether a write reached the vault and the index was not updated.
func isIndexOutdated(err error) bool { return errors.Is(err, edit.ErrIndexOutdated) }
