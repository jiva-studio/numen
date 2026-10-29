package service

import (
	"errors"
	"syscall"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Conversions from an outcome or a write error to the ErrorCode of a response.

// convertOutcomeToErrorCode says which error code an outcome is, and whether it is one at all.
func convertOutcomeToErrorCode(o edit.ReadOutcome) (api.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return api.ErrorCodeMissing, true
	case edit.NotANote:
		return api.ErrorCodeNotANote, true
	case edit.NotText:
		return api.ErrorCodeNotText, true
	case edit.TooLarge:
		return api.ErrorCodeTooLarge, true
	case edit.Unreadable:
		return api.ErrorCodeUnreadable, true
	case edit.Ok:
		return api.ErrorCodeUnspecified, false
	}
	return api.ErrorCodeUnspecified, false
}

// getErrorCode says which error code a write's error is, and whether it is one at all. Anything
// else is the vault being out of reach.
func getErrorCode(err error) (api.ErrorCode, bool) {
	switch {
	case errors.Is(err, ports.ErrStale):
		return api.ErrorCodeStale, true
	case errors.Is(err, edit.ErrNoNote):
		return api.ErrorCodeMissing, true
	case errors.Is(err, edit.ErrTooLarge):
		return api.ErrorCodeTooLarge, true
	case errors.Is(err, domain.ErrInvalidName):
		return api.ErrorCodeInvalidName, true
	case errors.Is(err, edit.ErrUnreadable),
		errors.Is(err, edit.ErrInline),
		errors.Is(err, edit.ErrUnterminated),
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

// isUnlevelled says whether a write reached the vault and the index did not follow.
func isUnlevelled(err error) bool { return errors.Is(err, edit.ErrUnlevelled) }
