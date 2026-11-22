// Package identity serves a window's IdentityService: reading the identity of this device, making
// recovery words, creating the identity from them and deleting it. No key reaches the window.
package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// UseCases are the use cases IdentityService is answered through.
type UseCases struct {
	Get           *identity.GetIdentity
	GenerateWords *identity.GenerateRecoveryWords
	Create        *identity.CreateIdentity
	Delete        *identity.DeleteIdentity
}

// Handler serves IdentityService over the identity use cases.
type Handler struct {
	identity UseCases
}

// NewHandler returns the handler over the identity use cases of this installation.
func NewHandler(useCases UseCases) *Handler {
	return &Handler{identity: useCases}
}

// GetIdentity returns the identity of this device, or none.
func (h *Handler) GetIdentity(
	ctx context.Context,
	_ *connect.Request[v1.GetIdentityRequest],
) (*connect.Response[v1.GetIdentityResponse], error) {
	found, err := h.identity.Get.Execute(ctx)
	if errors.Is(err, identitydomain.ErrNoIdentity) {
		return connect.NewResponse(&v1.GetIdentityResponse{}), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetIdentityResponse{Identity: writeIdentity(found)}), nil
}

// GenerateRecoveryWords returns 12 new recovery words and keeps nothing.
func (h *Handler) GenerateRecoveryWords(
	ctx context.Context,
	_ *connect.Request[v1.GenerateRecoveryWordsRequest],
) (*connect.Response[v1.GenerateRecoveryWordsResponse], error) {
	words, err := h.identity.GenerateWords.Execute(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GenerateRecoveryWordsResponse{Words: words}), nil
}

// CreateIdentity keeps an identity on this device for the recovery words.
func (h *Handler) CreateIdentity(
	ctx context.Context,
	r *connect.Request[v1.CreateIdentityRequest],
) (*connect.Response[v1.CreateIdentityResponse], error) {
	created, err := h.identity.Create.Execute(ctx, r.Msg.GetWords())
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.CreateIdentityResponse{
			Error:  code,
			Detail: getDetail(err),
		}), nil
	}
	return connect.NewResponse(&v1.CreateIdentityResponse{Identity: writeIdentity(created)}), nil
}

func (h *Handler) DeleteIdentity(
	ctx context.Context,
	_ *connect.Request[v1.DeleteIdentityRequest],
) (*connect.Response[v1.DeleteIdentityResponse], error) {
	if err := h.identity.Delete.Execute(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteIdentityResponse{}), nil
}

func writeIdentity(found identitydomain.Identity) *v1.IdentityInfo {
	return &v1.IdentityInfo{
		AccountFingerprint: found.AccountFingerprint,
		DeviceFingerprint:  found.DeviceFingerprint,
		DeviceName:         found.DeviceName,
		DeviceAddedAt:      found.DeviceAddedAt.UTC().Format(time.RFC3339),
	}
}

func getDetail(err error) string {
	sentinels := []error{
		identitydomain.ErrInvalidWords,
		identitydomain.ErrIdentityExists,
		identitydomain.ErrKeyStore,
	}
	for _, sentinel := range sentinels {
		if errors.Is(err, sentinel) {
			return strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
		}
	}
	return err.Error()
}

func getErrorCode(err error) (v1.IdentityErrorCode, bool) {
	switch {
	case errors.Is(err, identitydomain.ErrInvalidWords):
		return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS, true
	case errors.Is(err, identitydomain.ErrIdentityExists):
		return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS, true
	case errors.Is(err, identitydomain.ErrKeyStore):
		return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED, true
	}
	return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED, false
}
