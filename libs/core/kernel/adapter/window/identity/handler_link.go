package identity

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// LinkUseCases are the use cases of a link a window starts, looks at, answers and ends.
type LinkUseCases struct {
	Start  *identity.StartLink
	Get    *identity.GetLink
	Decide *identity.DecideLink
	Cancel *identity.CancelLink
}

// AccountUseCases are the use cases of the devices of the account and of joining one.
type AccountUseCases struct {
	EnterCode   *identity.EnterLinkCode
	ConfirmJoin *identity.ConfirmJoin
	List        *identity.ListDevices
	Revoke      *identity.RevokeDevice
}

func (h *Handler) StartLink(
	ctx context.Context,
	r *connect.Request[v1.StartLinkRequest],
) (*connect.Response[v1.StartLinkResponse], error) {
	link, err := h.identity.Link.Start.Execute(ctx, readLinkRole(r.Msg.GetRole()), readLinkMode(r.Msg.GetMode()), r.Msg.GetScannedText())
	if err != nil {
		code, detail, served := getLinkRefusal(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.StartLinkResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.StartLinkResponse{Link: writeLink(link)}), nil
}

func (h *Handler) GetLink(
	ctx context.Context,
	_ *connect.Request[v1.GetLinkRequest],
) (*connect.Response[v1.GetLinkResponse], error) {
	link, err := h.identity.Link.Get.Execute(ctx)
	if err != nil {
		code, detail, served := getLinkRefusal(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.GetLinkResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.GetLinkResponse{Link: writeLink(link)}), nil
}

// DecideLink answers the new device.
func (h *Handler) DecideLink(
	ctx context.Context,
	r *connect.Request[v1.DecideLinkRequest],
) (*connect.Response[v1.DecideLinkResponse], error) {
	code, detail, err := serveLinkCall(h.identity.Link.Decide.Execute(ctx, r.Msg.GetApprove()))
	return connect.NewResponse(&v1.DecideLinkResponse{Error: code, Detail: detail}), err
}

// EnterLinkCode sends the confirmation code of the person.
func (h *Handler) EnterLinkCode(
	ctx context.Context,
	r *connect.Request[v1.EnterLinkCodeRequest],
) (*connect.Response[v1.EnterLinkCodeResponse], error) {
	code, detail, err := serveLinkCall(h.identity.Account.EnterCode.Execute(ctx, r.Msg.GetCode()))
	return connect.NewResponse(&v1.EnterLinkCodeResponse{Error: code, Detail: detail}), err
}

// ConfirmJoin answers the account the new device is offered.
func (h *Handler) ConfirmJoin(
	ctx context.Context,
	r *connect.Request[v1.ConfirmJoinRequest],
) (*connect.Response[v1.ConfirmJoinResponse], error) {
	code, detail, err := serveLinkCall(h.identity.Account.ConfirmJoin.Execute(ctx, r.Msg.GetApprove()))
	return connect.NewResponse(&v1.ConfirmJoinResponse{Error: code, Detail: detail}), err
}

// CancelLink ends the link in progress.
func (h *Handler) CancelLink(
	ctx context.Context,
	_ *connect.Request[v1.CancelLinkRequest],
) (*connect.Response[v1.CancelLinkResponse], error) {
	code, detail, err := serveLinkCall(h.identity.Link.Cancel.Execute(ctx))
	return connect.NewResponse(&v1.CancelLinkResponse{Error: code, Detail: detail}), err
}

// RevokeDevice removes a device from the account.
func (h *Handler) RevokeDevice(
	ctx context.Context,
	r *connect.Request[v1.RevokeDeviceRequest],
) (*connect.Response[v1.RevokeDeviceResponse], error) {
	fingerprint := r.Msg.GetFingerprint()
	err := h.identity.Account.Revoke.Execute(ctx, fingerprint)
	if err != nil {
		slog.WarnContext(ctx, "could not revoke the device", "fingerprint", fingerprint, "error", err)
	} else {
		slog.InfoContext(ctx, "the device waits to be revoked on the server", "fingerprint", fingerprint)
	}
	code, detail, err := serveLinkCall(err)
	return connect.NewResponse(&v1.RevokeDeviceResponse{Error: code, Detail: detail}), err
}

func (h *Handler) ListDevices(
	ctx context.Context,
	_ *connect.Request[v1.ListDevicesRequest],
) (*connect.Response[v1.ListDevicesResponse], error) {
	devices, err := h.identity.Account.List.Execute(ctx)
	if err != nil {
		code, detail, served := getLinkRefusal(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ListDevicesResponse{Error: code, Detail: detail}), nil
	}
	listed := make([]*v1.DeviceInfo, 0, len(devices))
	for _, device := range devices {
		listed = append(listed, &v1.DeviceInfo{
			Name:        device.Name,
			Fingerprint: device.Fingerprint,
			AddedAt:     device.AddedAt.UTC().Format(time.RFC3339),
			IsThis:      device.IsThis,
			IsRevoked:   device.IsRevoked,
		})
	}
	return connect.NewResponse(&v1.ListDevicesResponse{Devices: listed}), nil
}

func serveLinkCall(err error) (v1.IdentityErrorCode, string, error) {
	if err == nil {
		return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED, "", nil
	}
	code, detail, served := getLinkRefusal(err)
	if !served {
		return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED, "", connect.NewError(connect.CodeInternal, err)
	}
	return code, detail, nil
}

func getLinkRefusal(err error) (v1.IdentityErrorCode, string, bool) {
	code, served := getErrorCode(err)
	return code, err.Error(), served
}

func getLinkErrorCode(err error) (v1.IdentityErrorCode, bool) {
	refusals := []struct {
		target error
		code   v1.IdentityErrorCode
	}{
		{identitydomain.ErrLinkActive, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LINK_ACTIVE},
		{identitydomain.ErrNoLink, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_LINK},
		{identitylib.ErrLinkPayload, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LINK_TEXT_INVALID},
		{identitydomain.ErrLastDevice, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_LAST_DEVICE},
		{identitydomain.ErrRevokeRaced, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_REVOKE_RACED},
		{identitylib.ErrSelfRevoke, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ALLOWED_THIS_DEVICE},
		{identitydomain.ErrNoIdentity, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY},
		{identitydomain.ErrDeviceRevoked, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_DEVICE_REVOKED},
		{identitylib.ErrUnknownDevice, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNKNOWN_DEVICE},
		{identity.ErrCodeFormat, v1.IdentityErrorCode_IDENTITY_ERROR_CODE_CODE_INVALID},
	}
	for _, refusal := range refusals {
		if errors.Is(err, refusal.target) {
			return refusal.code, true
		}
	}
	return v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED, false
}

func readLinkRole(role v1.LinkRole) identitydomain.LinkRole {
	if role == v1.LinkRole_LINK_ROLE_HOST {
		return identitydomain.LinkRoleHost
	}
	return identitydomain.LinkRoleJoiner
}

func readLinkMode(mode v1.LinkMode) identitydomain.LinkMode {
	if mode == v1.LinkMode_LINK_MODE_SCAN {
		return identitydomain.LinkModeScan
	}
	return identitydomain.LinkModeShow
}

func writeLink(link identitydomain.Link) *v1.LinkInfo {
	return &v1.LinkInfo{
		Phase:              writeLinkPhase(link.Phase),
		Mode:               writeLinkMode(link.Mode),
		Role:               writeLinkRole(link.Role),
		QrText:             link.QRText,
		Code:               link.Code,
		PeerName:           link.PeerName,
		PeerFingerprint:    link.PeerFingerprint,
		AccountFingerprint: link.AccountFingerprint,
		TriesLeft:          uint32(max(link.TriesLeft, 0)),
		Reason:             writeLinkReason(link.Reason),
		DeviceName:         link.DeviceName,
		DeviceFingerprint:  link.DeviceFingerprint,
		IsApproved:         link.IsApproved,
	}
}

func writeLinkRole(role identitydomain.LinkRole) v1.LinkRole {
	if role == identitydomain.LinkRoleHost {
		return v1.LinkRole_LINK_ROLE_HOST
	}
	return v1.LinkRole_LINK_ROLE_JOINER
}

func writeLinkMode(mode identitydomain.LinkMode) v1.LinkMode {
	if mode == identitydomain.LinkModeScan {
		return v1.LinkMode_LINK_MODE_SCAN
	}
	return v1.LinkMode_LINK_MODE_SHOW
}

func writeLinkPhase(phase identitydomain.LinkPhase) v1.LinkPhase {
	phases := map[identitydomain.LinkPhase]v1.LinkPhase{
		identitydomain.LinkPhaseWaitingForChannel: v1.LinkPhase_LINK_PHASE_WAITING_FOR_CHANNEL,
		identitydomain.LinkPhaseWaitingForPeer:    v1.LinkPhase_LINK_PHASE_WAITING_FOR_PEER,
		identitydomain.LinkPhaseConfirmPeer:       v1.LinkPhase_LINK_PHASE_CONFIRM_PEER,
		identitydomain.LinkPhaseWaitingForAccept:  v1.LinkPhase_LINK_PHASE_WAITING_FOR_ACCEPT,
		identitydomain.LinkPhaseEnterCode:         v1.LinkPhase_LINK_PHASE_ENTER_CODE,
		identitydomain.LinkPhasePublishing:        v1.LinkPhase_LINK_PHASE_PUBLISHING,
		identitydomain.LinkPhaseConfirmAccount:    v1.LinkPhase_LINK_PHASE_CONFIRM_ACCOUNT,
		identitydomain.LinkPhaseLinked:            v1.LinkPhase_LINK_PHASE_LINKED,
		identitydomain.LinkPhaseFailed:            v1.LinkPhase_LINK_PHASE_FAILED,
	}
	return phases[phase]
}

func writeLinkReason(reason identitydomain.LinkReason) v1.LinkReason {
	reasons := map[identitydomain.LinkReason]v1.LinkReason{
		identitydomain.LinkReasonDeclined:          v1.LinkReason_LINK_REASON_DECLINED,
		identitydomain.LinkReasonDeviceHasIdentity: v1.LinkReason_LINK_REASON_DEVICE_HAS_IDENTITY,
		identitydomain.LinkReasonWrongCode:         v1.LinkReason_LINK_REASON_WRONG_CODE,
		identitydomain.LinkReasonTooManyWrongCodes: v1.LinkReason_LINK_REASON_TOO_MANY_WRONG_CODES,
		identitydomain.LinkReasonStaleChain:        v1.LinkReason_LINK_REASON_STALE_CHAIN,
		identitydomain.LinkReasonCanceled:          v1.LinkReason_LINK_REASON_CANCELED,
		identitydomain.LinkReasonExpired:           v1.LinkReason_LINK_REASON_EXPIRED,
		identitydomain.LinkReasonInvalid:           v1.LinkReason_LINK_REASON_INVALID,
	}
	return reasons[reason]
}
