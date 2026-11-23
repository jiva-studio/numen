package hostendpoint

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type entitlementSource struct {
	mu    sync.Mutex
	calls int
	rows  []identitydomain.Entitlement
	err   error
}

func (s *entitlementSource) GetEntitlements(context.Context) ([]identitydomain.Entitlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.rows, s.err
}

func (s *entitlementSource) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type entitlementsHost struct {
	source  *entitlementSource
	callers *callers
	client  func(token string) numenv1connect.HostAccountServiceClient
}

func newEntitlementsHost(t *testing.T, withSource bool) entitlementsHost {
	t.Helper()
	source := &entitlementSource{rows: []identitydomain.Entitlement{
		{Product: "link", ValidUntil: time.Unix(4_900_000_000, 0), Phase: "active"},
		{Product: "sync", ValidUntil: time.Unix(1_800_000_000, 0), Phase: "grace", QuotaBytes: 100 << 30, IsTrial: true},
	}}
	fake := newCallers()
	fake.confined = true
	fake.granted = []string{manifest.PermissionIdentity}
	hosted := fake.asCallers()
	if withSource {
		hosted.Entitlements = source
	}
	handler := &accountEndpoint{endpoint: &Endpoint{callers: hosted}}
	_, served := numenv1connect.NewHostAccountServiceHandler(handler)
	server := httptest.NewServer(served)
	t.Cleanup(server.Close)
	return entitlementsHost{source: source, callers: fake, client: func(token string) numenv1connect.HostAccountServiceClient {
		return numenv1connect.NewHostAccountServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
	}}
}

func TestAPluginWithTheIdentityPermissionReadsTheEntitlementsOfTheAccount(t *testing.T) {
	h := newEntitlementsHost(t, true)

	response, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := response.Msg.GetEntitlements()
	if len(got) != 2 || got[0].GetProduct() != "link" || got[0].GetValidUntil() != 4_900_000_000 || got[0].GetPhase() != "active" || got[0].GetQuotaBytes() != 0 || got[0].GetIsTrial() {
		t.Errorf("entitlements = %v, want link first, active, valid until 4900000000, no quota", got)
	}
	if got[1].GetProduct() != "sync" || got[1].GetPhase() != "grace" || got[1].GetQuotaBytes() != 100<<30 || !got[1].GetIsTrial() {
		t.Errorf("second entitlement = %v, want sync in grace with 100 GiB and a trial", got[1])
	}
	if checked := h.callers.checked; len(checked) != 1 || checked[0] != "md.numen.a identity" {
		t.Errorf("permissions checked = %v, want identity once", checked)
	}
}

func TestAPluginWithoutTheIdentityPermissionIsRefusedTheEntitlementsAndTheServerIsNotAsked(t *testing.T) {
	h := newEntitlementsHost(t, true)
	h.callers.granted = []string{manifest.PermissionVaultRead}

	_, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("error = %v, want permission denied", err)
	}
	if h.source.callCount() != 0 {
		t.Error("the sync server was asked for a plugin without the permission")
	}
}

func TestAnUnknownTokenIsUnauthenticatedForTheEntitlements(t *testing.T) {
	h := newEntitlementsHost(t, true)

	_, err := h.client("token-of-nobody").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

func TestAHostWithoutAnEntitlementSourceAnswersUnavailable(t *testing.T) {
	h := newEntitlementsHost(t, false)

	_, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("error = %v, want unavailable", err)
	}
}

func TestADeviceWithoutAnIdentityAnswersFailedPreconditionWithTheReasonNoIdentity(t *testing.T) {
	h := newEntitlementsHost(t, true)
	h.source.err = identitydomain.ErrNoIdentity

	_, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition || readReason(err) != noIdentityReason {
		t.Errorf("error = %v, want failed precondition with the reason NO_IDENTITY", err)
	}
}

func TestAnUnreachableSyncServerAnswersUnavailableWithTheReasonSyncUnavailable(t *testing.T) {
	h := newEntitlementsHost(t, true)
	h.source.err = identitydomain.ErrSyncUnavailable

	_, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodeUnavailable || readReason(err) != "SYNC_UNAVAILABLE" {
		t.Errorf("error = %v, want unavailable with the reason SYNC_UNAVAILABLE", err)
	}
}

func TestAFailureOfTheSourceShowsNoDetailOfTheServerToThePlugin(t *testing.T) {
	h := newEntitlementsHost(t, true)
	h.source.err = context.DeadlineExceeded

	_, err := h.client("token-a").GetEntitlements(t.Context(), connect.NewRequest(&v1.GetEntitlementsRequest{}))

	if connect.CodeOf(err) != connect.CodeInternal || err.Error() != "internal: "+errAccountFailure.Error() {
		t.Errorf("error = %v, want the generic failure of the account", err)
	}
}
