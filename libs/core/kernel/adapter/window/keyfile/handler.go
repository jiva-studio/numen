// Package keyfile serves a window's KeyFileService: how a computer without a system keychain keeps the
// keys of the account, and the passphrase that opens the file. The passphrase is never answered.
package keyfile

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// UseCases are the use cases KeyFileService is answered through.
type UseCases struct {
	GetState *keyfile.GetState
	Choose   *keyfile.Choose
	Unlock   *keyfile.Unlock
}

// Handler answers KeyFileService to a window.
type Handler struct {
	useCases UseCases
}

var _ numenv1connect.KeyFileServiceHandler = (*Handler)(nil)

// NewHandler answers over the use cases.
func NewHandler(useCases UseCases) *Handler { return &Handler{useCases: useCases} }

// NewRoute returns the path KeyFileService is served at and the handler that serves it.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewKeyFileServiceHandler(handler, counted)
}

// GetKeyFileState tells how the keys are kept in the file.
func (h *Handler) GetKeyFileState(
	ctx context.Context, _ *connect.Request[v1.GetKeyFileStateRequest],
) (*connect.Response[v1.GetKeyFileStateResponse], error) {
	state, err := h.useCases.GetState.Execute(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetKeyFileStateResponse{
		IsInUse:        state.IsInUse,
		Mode:           writeMode(state.Mode),
		IsChoiceNeeded: state.IsInUse && state.Mode == domainkeyfile.Unset,
		IsLocked:       state.IsLocked,
		Warning:        state.Warning,
	}), nil
}

// ChooseKeyFile sets how the file keeps the keys.
func (h *Handler) ChooseKeyFile(
	ctx context.Context, r *connect.Request[v1.ChooseKeyFileRequest],
) (*connect.Response[v1.ChooseKeyFileResponse], error) {
	err := h.useCases.Choose.Execute(ctx, readMode(r.Msg.GetMode()), r.Msg.GetPassphrase())
	code, err := getErrorCode(err)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ChooseKeyFileResponse{Error: code}), nil
}

// UnlockKeyFile opens the encrypted file for this run.
func (h *Handler) UnlockKeyFile(
	ctx context.Context, r *connect.Request[v1.UnlockKeyFileRequest],
) (*connect.Response[v1.UnlockKeyFileResponse], error) {
	code, err := getErrorCode(h.useCases.Unlock.Execute(ctx, r.Msg.GetPassphrase()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UnlockKeyFileResponse{Error: code}), nil
}

func getErrorCode(err error) (v1.ErrorCode, error) {
	switch {
	case err == nil:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, nil
	case errors.Is(err, domainkeyfile.ErrWrongPassphrase):
		return v1.ErrorCode_ERROR_CODE_WRONG_PASSPHRASE, nil
	case errors.Is(err, domainkeyfile.ErrEmptyPassphrase):
		return v1.ErrorCode_ERROR_CODE_EMPTY_PASSPHRASE, nil
	case errors.Is(err, keyfile.ErrNotSaved):
		return v1.ErrorCode_ERROR_CODE_INTERNAL, nil
	case errors.Is(err, keyfile.ErrInvalidMode):
		return v1.ErrorCode_ERROR_CODE_INVALID, nil
	}
	return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, err
}

func writeMode(mode domainkeyfile.Mode) v1.KeyFileMode {
	switch mode {
	case domainkeyfile.Plaintext:
		return v1.KeyFileMode_KEY_FILE_MODE_PLAINTEXT
	case domainkeyfile.Passphrase:
		return v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE
	case domainkeyfile.Unset:
		return v1.KeyFileMode_KEY_FILE_MODE_UNSET
	}
	return v1.KeyFileMode_KEY_FILE_MODE_UNSET
}

func readMode(mode v1.KeyFileMode) domainkeyfile.Mode {
	switch mode {
	case v1.KeyFileMode_KEY_FILE_MODE_PLAINTEXT:
		return domainkeyfile.Plaintext
	case v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE:
		return domainkeyfile.Passphrase
	case v1.KeyFileMode_KEY_FILE_MODE_UNSET, v1.KeyFileMode_KEY_FILE_MODE_UNSPECIFIED:
		return domainkeyfile.Unset
	}
	return domainkeyfile.Unset
}
