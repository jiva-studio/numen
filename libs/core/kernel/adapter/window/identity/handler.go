// Package identity serves a window's IdentityService: reading the identity of this device, making
// recovery words and recovery kits, creating the identity from words or sheets and deleting it. No key
// reaches the window.
package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var (
	errNoProof  = errors.New("give the recovery words or the sheets as the proof")
	errNoSheets = errors.New("give the sheets")
)

// UseCases are the use cases IdentityService is answered through.
type UseCases struct {
	Identity IdentityUseCases
	Kit      KitUseCases
	Link     LinkUseCases
	Account  AccountUseCases
}

// IdentityUseCases are the use cases that read, create and delete the identity.
type IdentityUseCases struct {
	Get           *identity.GetIdentity
	GenerateWords *identity.GenerateRecoveryWords
	Create        *identity.CreateIdentity
	Delete        *identity.DeleteIdentity
	StartRestore  *identity.StartRestore
}

// KitUseCases are the use cases that make, check and restore from a recovery kit.
type KitUseCases struct {
	Create  *identity.CreateRecoveryKit
	Check   *identity.CheckRecoveryKit
	Restore *identity.StartRestoreFromKit
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
	found, err := h.identity.Identity.Get.Execute(ctx)
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
	words, err := h.identity.Identity.GenerateWords.Execute(ctx)
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
	created, err := h.identity.Identity.Create.Execute(ctx, r.Msg.GetWords())
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

// StartRestore starts to restore the account of the recovery words on this device.
func (h *Handler) StartRestore(
	ctx context.Context,
	r *connect.Request[v1.StartRestoreRequest],
) (*connect.Response[v1.StartRestoreResponse], error) {
	if err := h.identity.Identity.StartRestore.Execute(ctx, r.Msg.GetWords()); err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.StartRestoreResponse{Error: code, Detail: getDetail(err)}), nil
	}
	return connect.NewResponse(&v1.StartRestoreResponse{}), nil
}

func (h *Handler) DeleteIdentity(
	ctx context.Context,
	_ *connect.Request[v1.DeleteIdentityRequest],
) (*connect.Response[v1.DeleteIdentityResponse], error) {
	if err := h.identity.Identity.Delete.Execute(ctx); err != nil {
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
		IsRevoked:          found.IsRevoked,
		KeyEpoch:           found.KeyEpoch,
	}
}

// CreateRecoveryKit makes a kit of three sheets for the account of this device once the caller has
// proved the account. It keeps nothing.
func (h *Handler) CreateRecoveryKit(
	ctx context.Context,
	r *connect.Request[v1.CreateRecoveryKitRequest],
) (*connect.Response[v1.CreateRecoveryKitResponse], error) {
	words := r.Msg.GetWords().GetWords()
	sheets := readSheets(r.Msg.GetSheets().GetSheets())
	if len(words) == 0 && len(sheets) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoProof)
	}
	made, err := h.identity.Kit.Create.Execute(ctx, words, sheets)
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.CreateRecoveryKitResponse{Error: code, Detail: getDetail(err)}), nil
	}
	return connect.NewResponse(&v1.CreateRecoveryKitResponse{Kit: writeKit(made)}), nil
}

// CheckRecoveryKit reports whether the sheets restore the account of this device.
func (h *Handler) CheckRecoveryKit(
	ctx context.Context,
	r *connect.Request[v1.CheckRecoveryKitRequest],
) (*connect.Response[v1.CheckRecoveryKitResponse], error) {
	sheets := readSheets(r.Msg.GetSheets())
	if len(sheets) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoSheets)
	}
	fingerprint, err := h.identity.Kit.Check.Execute(ctx, sheets)
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.CheckRecoveryKitResponse{Error: code, Detail: getDetail(err)}), nil
	}
	return connect.NewResponse(&v1.CheckRecoveryKitResponse{AccountFingerprint: fingerprint}), nil
}

// StartRestoreFromKit starts to restore the account of the sheets on this device.
func (h *Handler) StartRestoreFromKit(
	ctx context.Context,
	r *connect.Request[v1.StartRestoreFromKitRequest],
) (*connect.Response[v1.StartRestoreFromKitResponse], error) {
	sheets := readSheets(r.Msg.GetSheets())
	if len(sheets) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoSheets)
	}
	if err := h.identity.Kit.Restore.Execute(ctx, sheets); err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.StartRestoreFromKitResponse{Error: code, Detail: getDetail(err)}), nil
	}
	return connect.NewResponse(&v1.StartRestoreFromKitResponse{}), nil
}

func readSheets(sheets []*v1.RecoverySheet) [][]string {
	out := make([][]string, len(sheets))
	for i, sheet := range sheets {
		out[i] = sheet.GetWords()
	}
	return out
}

func writeKit(made identitydomain.Kit) *v1.RecoveryKit {
	sheets := make([]*v1.RecoverySheet, len(made.Sheets))
	for i, sheet := range made.Sheets {
		sheets[i] = &v1.RecoverySheet{Words: sheet.Words}
	}
	return &v1.RecoveryKit{KitId: made.ID, AccountFingerprint: made.AccountFingerprint, Sheets: sheets}
}

func getDetail(err error) string {
	if errors.Is(err, identitydomain.ErrInvalidSheets) {
		return getSheetDetail(err)
	}
	for _, refusal := range getRefusals() {
		if errors.Is(err, refusal.sentinel) {
			return strings.TrimPrefix(err.Error(), refusal.sentinel.Error()+": ")
		}
	}
	return err.Error()
}

func getSheetDetail(err error) string {
	var sheetErr *identitylib.SheetError
	if !errors.As(err, &sheetErr) {
		return identitydomain.ErrInvalidSheets.Error()
	}
	if errors.Is(err, slip39.ErrDuplicateMember) {
		return fmt.Sprintf("sheets 1 and %d are the same sheet", sheetErr.Position)
	}
	return fmt.Sprintf("sheet %d is not valid", sheetErr.Position)
}

type refusal struct {
	sentinel error
	code     v1.IdentityErrorCode
}

func getRefusals() []refusal {
	return []refusal{
		{identitydomain.ErrInvalidWords, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS},
		{identitydomain.ErrIdentityExists, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS},
		{identitydomain.ErrKeyStore, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED},
		{identitydomain.ErrInvalidSheets, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS},
		{identitydomain.ErrNotEnoughSheets, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS},
		{identitydomain.ErrOtherAccount, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT},
		{identitydomain.ErrNoIdentity, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY},
	}
}

func getErrorCode(err error) (v1.IdentityErrorCode, bool) {
	for _, refusal := range getRefusals() {
		if errors.Is(err, refusal.sentinel) {
			return refusal.code, true
		}
	}
	return getLinkErrorCode(err)
}
