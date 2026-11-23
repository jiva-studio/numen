package fakeserver

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	lifetimeYears = 100
	trialPeriod   = 14 * day
	portalFolder  = "/portal/"
	portalPage    = "<!doctype html><title>Portal</title><p>Manage the subscription</p>"
	storageSlug   = "sync"
)

// shop is what the server sold besides the subscription of the storage: the products it offers, the ones each account owns
// by the day of the purchase, and the accounts whose subscription is a trial.
type shop struct {
	products []syncprotocol.Product
	owned    map[string]map[string]time.Time
	trials   map[string]bool
}

func newShop(products []syncprotocol.Product) shop {
	return shop{products: products, owned: map[string]map[string]time.Time{}, trials: map[string]bool{}}
}

func (s *Server) findProduct(id string) (syncprotocol.Product, bool) {
	index := slices.IndexFunc(s.shop.products, func(one syncprotocol.Product) bool { return one.ID == id })
	if index < 0 {
		return syncprotocol.Product{}, false
	}
	return s.shop.products[index], true
}

func (s *Server) findPlan(id string) (syncprotocol.Plan, bool) {
	index := slices.IndexFunc(s.plans, func(one syncprotocol.Plan) bool { return one.ID == id })
	if index < 0 {
		return syncprotocol.Plan{}, false
	}
	return s.plans[index], true
}

func (s *Server) buy(account string, product syncprotocol.Product) {
	if s.shop.owned[account] == nil {
		s.shop.owned[account] = map[string]time.Time{}
	}
	s.shop.owned[account][product.Slug] = s.now()
}

func (s *Server) ListProducts(_ context.Context, req *connect.Request[syncv1.ListProductsRequest]) (*connect.Response[syncv1.ListProductsResponse], error) {
	if _, err := s.authenticate(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	products := make([]*syncv1.Product, 0, len(s.shop.products))
	for _, one := range s.shop.products {
		products = append(products, &syncv1.Product{
			Id: one.ID, Slug: one.Slug, Name: one.Name, PriceCents: one.PriceCents, Currency: one.Currency, OneTime: one.OneTime,
		})
	}
	return connect.NewResponse(&syncv1.ListProductsResponse{Products: products}), nil
}

func (s *Server) GetEntitlements(_ context.Context, req *connect.Request[syncv1.GetEntitlementsRequest]) (*connect.Response[syncv1.GetEntitlementsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entitlements := make([]*syncv1.Entitlement, 0, len(s.shop.owned[account])+1)
	for slug, boughtAt := range s.shop.owned[account] {
		entitlements = append(entitlements, &syncv1.Entitlement{
			Product: slug, ValidUntil: boughtAt.AddDate(lifetimeYears, 0, 0).Unix(), Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE,
		})
	}
	if entitlement := s.getStorageEntitlement(account); entitlement != nil {
		entitlements = append(entitlements, entitlement)
	}
	slices.SortFunc(entitlements, func(a, b *syncv1.Entitlement) int { return cmp.Compare(a.GetProduct(), b.GetProduct()) })
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

// serveRefund takes a product back from an account (POST, product=slug and an optional account).
func (s *Server) serveRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.listAccounts(r.URL.Query().Get("account")) {
		delete(s.shop.owned[account], r.URL.Query().Get("product"))
	}
	w.WriteHeader(http.StatusNoContent)
}

// serveTrial starts a trial of a plan (POST, plan=id and an optional account): the subscription runs to the end of the trial.
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

// serveUsage sets the bytes the cloud holds (POST, bytes=N and an optional account).
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
