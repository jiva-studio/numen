package fakeserver

import (
	"context"
	"net/http"
	"slices"
	"strconv"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	trialPeriod  = 14 * day
	portalFolder = "/portal/"
	portalPage   = "<!doctype html><title>Portal</title><p>Manage the subscription</p>"
	storageSlug  = "sync"
)

// shop is what the server keeps besides the subscription of the storage: the accounts whose subscription is a trial.
type shop struct {
	trials map[string]bool
}

func (s *Server) findPlan(id string) (syncprotocol.Plan, bool) {
	index := slices.IndexFunc(s.plans, func(one syncprotocol.Plan) bool { return one.ID == id })
	if index < 0 {
		return syncprotocol.Plan{}, false
	}
	return s.plans[index], true
}

func (s *Server) ListProducts(_ context.Context, req *connect.Request[syncv1.ListProductsRequest]) (*connect.Response[syncv1.ListProductsResponse], error) {
	if _, err := s.authenticate(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.ListProductsResponse{Products: nil}), nil
}

func (s *Server) GetEntitlements(_ context.Context, req *connect.Request[syncv1.GetEntitlementsRequest]) (*connect.Response[syncv1.GetEntitlementsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entitlements := make([]*syncv1.Entitlement, 0, 1)
	if entitlement := s.getStorageEntitlement(account); entitlement != nil {
		entitlements = append(entitlements, entitlement)
	}
	return connect.NewResponse(&syncv1.GetEntitlementsResponse{Entitlements: entitlements}), nil
}

func (s *Server) getStorageEntitlement(account string) *syncv1.Entitlement {
	if !s.needsSubscription {
		return nil
	}
	phase, periodEnd := s.getPhase(account)
	if phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE && phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE {
		return nil
	}
	return &syncv1.Entitlement{
		Product: storageSlug, ValidUntil: periodEnd.Unix(), Phase: phase, QuotaBytes: s.getQuota(account), IsTrial: s.shop.trials[account],
	}
}

func (s *Server) CreatePortalSession(_ context.Context, req *connect.Request[syncv1.CreatePortalSessionRequest]) (*connect.Response[syncv1.CreatePortalSessionResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, paid := s.subscriptions[account]; !paid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNeverPaid)
	}
	return connect.NewResponse(&syncv1.CreatePortalSessionResponse{PortalUrl: s.audience + portalFolder}), nil
}

func (s *Server) servePortal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(portalPage)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serveTrial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, found := s.findPlan(r.URL.Query().Get("plan")); !found {
		http.Error(w, "unknown plan", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.listAccounts(r.URL.Query().Get("account")) {
		s.subscriptions[account] = s.now().Add(trialPeriod)
		s.planOf[account] = r.URL.Query().Get("plan")
		s.shop.trials[account] = true
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) serveCancelTrial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, found := s.findPlan(r.URL.Query().Get("plan")); !found {
		http.Error(w, "unknown plan", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.listAccounts(r.URL.Query().Get("account")) {
		if s.planOf[account] == r.URL.Query().Get("plan") {
			delete(s.shop.trials, account)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) serveUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bytes, err := strconv.ParseUint(r.URL.Query().Get("bytes"), 10, 64)
	if err != nil {
		http.Error(w, "bytes is a number", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.listAccounts(r.URL.Query().Get("account")) {
		s.used[account] = bytes
	}
	w.WriteHeader(http.StatusNoContent)
}

func defaultPlans() []syncprotocol.Plan {
	const gib = 1 << 30
	return []syncprotocol.Plan{
		{ID: "basic", Name: "Basic", QuotaBytes: 10 * gib, PriceCents: 500, Currency: "USD"},
		{ID: "plus", Name: "Plus", QuotaBytes: 100 * gib, PriceCents: 1500, Currency: "USD"},
		{ID: "pro", Name: "Pro", QuotaBytes: 250 * gib, PriceCents: 3000, Currency: "USD"},
	}
}
