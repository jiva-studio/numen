package identity_test

import (
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func newLinkHandler() *windowidentity.Handler {
	clock := testsupport.NewClockAt(time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC))
	deps := identity.NewDeps(newKeys(), names{}, clock).WithChains(&chainFile{})
	return windowidentity.NewHandler(windowidentity.UseCases{
		Identity: windowidentity.IdentityUseCases{
			Get:           identity.NewGetIdentity(deps),
			GenerateWords: identity.NewGenerateRecoveryWords(deps),
			Create:        identity.NewCreateIdentity(deps),
			Delete:        identity.NewDeleteIdentity(deps),
			StartRestore:  identity.NewStartRestore(deps),
		},
		Kit: windowidentity.KitUseCases{
			Save:    identity.NewSaveRecoveryKit(deps, &kitSaver{}),
			Restore: identity.NewStartRestoreFromKit(deps),
		},
		Link: windowidentity.LinkUseCases{
			Start: identity.NewStartLink(deps), Get: identity.NewGetLink(deps),
			Decide: identity.NewDecideLink(deps), Cancel: identity.NewCancelLink(deps),
		},
		Account: windowidentity.AccountUseCases{
			ConfirmJoin: identity.NewConfirmJoin(deps),
			List:        identity.NewListDevices(deps), Revoke: identity.NewRevokeDevice(deps),
		},
	})
}

func startLink(t *testing.T, h *windowidentity.Handler, role v1.LinkRole, mode v1.LinkMode, text string) *v1.StartLinkResponse {
	t.Helper()
	answer, err := h.StartLink(t.Context(), connect.NewRequest(&v1.StartLinkRequest{Role: role, Mode: mode, ScannedText: text}))
	if err != nil {
		t.Fatalf("StartLink() error = %v", err)
	}
	return answer.Msg
}

func TestStartLink_AHostThatShowsAnswersALinkThatWaitsForTheCourier(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)

	started := startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")

	link := started.GetLink()
	if started.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED || link.GetPhase() != v1.LinkPhase_LINK_PHASE_WAITING_FOR_CHANNEL ||
		link.GetRole() != v1.LinkRole_LINK_ROLE_HOST || link.GetMode() != v1.LinkMode_LINK_MODE_SHOW {
		t.Fatalf("StartLink() = %v", started)
	}
	if link.GetDeviceName() != "Laptop" || link.GetDeviceFingerprint() == "" {
		t.Errorf("link = %v, want the name and the fingerprint of this device", link)
	}
}

func TestStartLink_RefusesAHostWithoutAnIdentity(t *testing.T) {
	t.Parallel()
	started := startLink(t, newLinkHandler(), v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")
	if started.GetError() != v1.ErrorCode_ERROR_CODE_NO_IDENTITY || started.GetLink() != nil {
		t.Errorf("StartLink() = %v, want NO_IDENTITY", started)
	}
}

func TestStartLink_RefusesAJoinerThatHasAnIdentity(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)
	started := startLink(t, h, v1.LinkRole_LINK_ROLE_JOINER, v1.LinkMode_LINK_MODE_SHOW, "")
	if started.GetError() != v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS {
		t.Errorf("StartLink() = %v, want IDENTITY_EXISTS", started)
	}
}

func TestStartLink_RefusesATextThatIsNotALink(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)
	started := startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SCAN, "https://example.org")
	if started.GetError() != v1.ErrorCode_ERROR_CODE_LINK_TEXT_INVALID {
		t.Errorf("StartLink() = %v, want LINK_TEXT_INVALID", started)
	}
}

func TestStartLink_RefusesASecondLinkWhileOneRuns(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)
	startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")
	started := startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")
	if started.GetError() != v1.ErrorCode_ERROR_CODE_LINK_ACTIVE {
		t.Errorf("StartLink() = %v, want LINK_ACTIVE", started)
	}
}

func TestGetLink_AnswersNoLinkBeforeOneStartedAndTheLinkAfter(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)
	none, err := h.GetLink(t.Context(), connect.NewRequest(&v1.GetLinkRequest{}))
	if err != nil || none.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_LINK {
		t.Fatalf("GetLink() = %v, %v, want NO_LINK", none, err)
	}
	startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")
	got, err := h.GetLink(t.Context(), connect.NewRequest(&v1.GetLinkRequest{}))
	if err != nil || got.Msg.GetLink().GetPhase() != v1.LinkPhase_LINK_PHASE_WAITING_FOR_CHANNEL {
		t.Errorf("GetLink() = %v, %v, want a link that waits for the channel", got, err)
	}
}

func TestCancelLink_EndsTheLinkSoThatTheNextOneCanStart(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	createWithNewWords(t, h)
	startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, "")

	canceled, err := h.CancelLink(t.Context(), connect.NewRequest(&v1.CancelLinkRequest{}))
	if err != nil || canceled.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("CancelLink() = %v, %v", canceled, err)
	}

	got, err := h.GetLink(t.Context(), connect.NewRequest(&v1.GetLinkRequest{}))
	if err != nil || got.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_LINK {
		t.Errorf("GetLink() = %v, %v, want NO_LINK after the cancel", got, err)
	}
	if started := startLink(t, h, v1.LinkRole_LINK_ROLE_HOST, v1.LinkMode_LINK_MODE_SHOW, ""); started.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Errorf("StartLink() after the cancel = %v", started)
	}
}

func TestCancelLink_AnswersNoLinkWhereNoneRuns(t *testing.T) {
	t.Parallel()
	answer, err := newLinkHandler().CancelLink(t.Context(), connect.NewRequest(&v1.CancelLinkRequest{}))
	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_LINK {
		t.Errorf("CancelLink() = %v, %v, want NO_LINK", answer, err)
	}
}

func TestDecideLink_AnswersNoLinkWhereNoneRuns(t *testing.T) {
	t.Parallel()
	answer, err := newLinkHandler().DecideLink(t.Context(), connect.NewRequest(&v1.DecideLinkRequest{Approve: true}))
	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_LINK {
		t.Errorf("DecideLink() = %v, %v, want NO_LINK", answer, err)
	}
}

func TestConfirmJoin_AnswersNoLinkWhereNoneRuns(t *testing.T) {
	t.Parallel()
	answer, err := newLinkHandler().ConfirmJoin(t.Context(), connect.NewRequest(&v1.ConfirmJoinRequest{Approve: true}))
	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_LINK {
		t.Errorf("ConfirmJoin() = %v, %v, want NO_LINK", answer, err)
	}
}

func TestListDevices_AnswersThisDeviceWithItsFingerprintAndDate(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	created := createWithNewWords(t, h)

	answer, err := h.ListDevices(t.Context(), connect.NewRequest(&v1.ListDevicesRequest{}))

	devices := answer.Msg.GetDevices()
	if err != nil || len(devices) != 1 {
		t.Fatalf("ListDevices() = %v, %v, want one device", answer, err)
	}
	if !devices[0].GetIsThis() || devices[0].GetIsRevoked() || devices[0].GetName() != "Laptop" ||
		devices[0].GetFingerprint() != created.GetIdentity().GetDeviceFingerprint() || devices[0].GetAddedAt() != "2026-11-22T12:00:00Z" {
		t.Errorf("device = %v", devices[0])
	}
}

func TestListDevices_AnswersNoIdentityWithoutOne(t *testing.T) {
	t.Parallel()
	answer, err := newLinkHandler().ListDevices(t.Context(), connect.NewRequest(&v1.ListDevicesRequest{}))
	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_IDENTITY {
		t.Errorf("ListDevices() = %v, %v, want NO_IDENTITY", answer, err)
	}
}

func TestRevokeDevice_RefusesToRevokeTheLastDeviceOfTheAccount(t *testing.T) {
	t.Parallel()
	h := newLinkHandler()
	created := createWithNewWords(t, h)

	answer, err := h.RevokeDevice(t.Context(), connect.NewRequest(&v1.RevokeDeviceRequest{Fingerprint: created.GetIdentity().GetDeviceFingerprint()}))

	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_LAST_DEVICE {
		t.Errorf("RevokeDevice() = %v, %v, want LAST_DEVICE", answer, err)
	}
}

func TestRevokeDevice_AnswersNoIdentityWithoutOne(t *testing.T) {
	t.Parallel()
	answer, err := newLinkHandler().RevokeDevice(t.Context(), connect.NewRequest(&v1.RevokeDeviceRequest{Fingerprint: strings.Repeat("0", 8)}))
	if err != nil || answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_IDENTITY {
		t.Errorf("RevokeDevice() = %v, %v, want NO_IDENTITY", answer, err)
	}
}
