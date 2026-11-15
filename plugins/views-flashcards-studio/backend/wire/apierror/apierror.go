// Package apierror turns the failures of the use cases into what a response carries: the ErrorCode
// of a note that could not be read or written, or the code of an API error.
package apierror

import (
	"errors"
	"syscall"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// ConvertOutcome is the ErrorCode of a read that did not return the note, and false for a read that did.
func ConvertOutcome(o edit.ReadOutcome) (api.ErrorCode, bool) {
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

// GetErrorCode is the ErrorCode of a write that failed on the note, and false for a failure that is
// the plugin's own.
func GetErrorCode(err error) (api.ErrorCode, bool) {
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

// GetCode is the code of the API error for a failure that is not about the note.
func GetCode(err error) server.ErrorCode {
	if errors.Is(err, host.ErrOutside) {
		return server.CodeInvalidArgument
	}
	return server.CodeInternal
}

// NewError is err as an API error of the code given.
func NewError(code server.ErrorCode, err error) error {
	return server.NewAPIError(code, err.Error())
}

// IsIndexOutdated reports whether a write reached the vault and the index did not follow.
func IsIndexOutdated(err error) bool { return errors.Is(err, edit.ErrIndexOutdated) }

func GetDeckError(o note.ReadOutcome, is domain.NoteType) (api.ErrorCode, bool) {
	if o == note.TooLarge {
		return api.ErrorCodeDeckTooLarge, true
	}
	if o == note.Ok && is != format.TypeDeck {
		return api.ErrorCodeNotADeck, true
	}
	return ConvertOutcome(o)
}

func GetCardModelError(o note.ReadOutcome, is domain.NoteType) (api.ErrorCode, bool) {
	if o == note.Ok && is != format.TypeCardModel {
		return api.ErrorCodeNotACardModel, true
	}
	return ConvertOutcome(o)
}

// DescribeCreate is why a command could not make the file, in words for the person who ran it.
func DescribeCreate(code api.ErrorCode) string {
	switch code {
	case api.ErrorCodeAlreadyExists:
		return "something of that name is filed there already, so nothing was written"
	case api.ErrorCodeInvalidName:
		return "a note cannot be called that"
	case api.ErrorCodeUnspecified, api.ErrorCodeNotFound, api.ErrorCodeNotANote, api.ErrorCodeNotText,
		api.ErrorCodeTooLarge, api.ErrorCodeBodyUnwritable, api.ErrorCodeUnreadable,
		api.ErrorCodeNotACardModel, api.ErrorCodeNotADeck, api.ErrorCodeDeckTooLarge,
		api.ErrorCodeNotAPreset, api.ErrorCodeConflict:
		// gomutants:disable-next-line BRANCH_CASE reason="Other error codes share the fallback message."
		return "the file could not be made: " + string(code)
	}
	return "the file could not be made: " + string(code)
}

// ClassifyCreate reads the end of a scenario that makes a note: the note made, the ErrorCode of a
// failure on the note, whether the index did not follow, and the API error of any other failure.
func ClassifyCreate(made cards.CreateNoteResult, err error) (cards.CreateNoteResult, api.ErrorCode, bool, error) {
	if err == nil || errors.Is(err, note.ErrIndexOutdated) {
		// The file is on disk under that name and nothing renumbers a second
		// attempt, so the path is the only way back to it.
		// gomutants:disable-next-line RETURN_ZERO reason="ErrorCodeUnspecified is the zero value of ErrorCode."
		return made, api.ErrorCodeUnspecified, IsIndexOutdated(err), nil
	}
	if errors.Is(err, cards.ErrNoFields) {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, NewError(server.CodeInvalidArgument, err)
	}
	reason, ok := GetErrorCode(err)
	if !ok {
		// gomutants:disable-next-line RETURN_ZERO reason="ErrorCodeUnspecified is the zero value of ErrorCode."
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, NewError(GetCode(err), err)
	}
	// gomutants:disable-next-line RETURN_ZERO reason="Empty struct literal equals the zero value."
	return cards.CreateNoteResult{Path: "", ID: "", Title: ""}, reason, false, nil
}

// NewFailure is err as an API error whose code is the one GetCode gives.
func NewFailure(err error) error { return NewError(GetCode(err), err) }

func NewInternal(err error) error { return NewError(server.CodeInternal, err) }

var errNoVault = errors.New("no vault is open")

// NewCurrentVaultError is why the vault in use could not be had: none is open, or the host failed.
func NewCurrentVaultError(err error) error {
	if errors.Is(err, host.ErrNoVault) {
		return NewError(server.CodeFailedPrecondition, errNoVault)
	}
	return NewInternal(err)
}

// NewVaultError is why a vault named could not be had: the installation holds none of that name, or the
// host failed.
func NewVaultError(err error) error {
	if errors.Is(err, host.ErrUnknownVault) {
		return NewError(server.CodeNotFound, err)
	}
	return NewInternal(err)
}
