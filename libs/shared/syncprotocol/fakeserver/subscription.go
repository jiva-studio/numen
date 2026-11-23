package fakeserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	day                = 24 * time.Hour
	subscriptionPeriod = 30 * day
	writeGrace         = 7 * day
	retention          = 60 * day
	payFolder          = "/pay/"
	payPage            = "<!doctype html><title>Payment</title><p>Payment received</p>"
)

var (
	errNeedsSubscription = errors.New("the account has no subscription that allows writes")
	errNeverPaid         = errors.New("the account never paid")
	errUnknownPlan       = errors.New("no such plan or product")
	errSamePlan          = errors.New("the account has this plan")
	errOwned             = errors.New("the account owns this product")
)

func (s *Server) getPhase(account string) (syncv1.SubscriptionPhase, time.Time) {
	if !s.needsSubscription {
		return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, s.now().Add(subscriptionPeriod)
	}
	periodEnd, found := s.subscriptions[account]
	if !found {
		return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE, time.Time{}
	}
	now := s.now()
	switch {
	case now.Before(periodEnd):
		return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, periodEnd
	case now.Before(periodEnd.Add(writeGrace)):
		return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, periodEnd
	case now.Before(periodEnd.Add(retention)):
		return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, periodEnd
	}
	return syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, periodEnd
}

func (s *Server) checkWriteAllowed(account string) error {
	switch phase, _ := s.getPhase(account); phase {
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE:
		return nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED, syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE,
		syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED:
	}
	return connect.NewError(connect.CodePermissionDenied, errNeedsSubscription)
}

func (s *Server) GetSubscription(_ context.Context, req *connect.Request[syncv1.GetSubscriptionRequest]) (*connect.Response[syncv1.GetSubscriptionResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return connect.NewResponse(s.getSubscriptionMessage(account)), nil
}

func (s *Server) CreateCheckout(_ context.Context, req *connect.Request[syncv1.CreateCheckoutRequest]) (*connect.Response[syncv1.CreateCheckoutResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	offer := req.Msg.GetPlan()
	if err := s.checkOffer(account, offer); err != nil {
		return nil, err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	id := hex.EncodeToString(raw)
	s.checkouts[id] = checkout{account: account, offer: offer}
	return connect.NewResponse(&syncv1.CreateCheckoutResponse{CheckoutUrl: s.audience + payFolder + id}), nil
}

func (s *Server) checkOffer(account, offer string) error {
	if product, isProduct := s.findProduct(offer); isProduct {
		if _, owned := s.shop.owned[account][product.Slug]; owned {
			return connect.NewError(connect.CodeAlreadyExists, errOwned)
		}
		return nil
	}
	if _, isPlan := s.findPlan(offer); !isPlan {
		return connect.NewError(connect.CodeInvalidArgument, errUnknownPlan)
	}
	if phase, _ := s.getPhase(account); phase == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		return connect.NewError(connect.CodeAlreadyExists, errors.New("the subscription is active"))
	}
	return nil
}

func (s *Server) servePay(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	paid, found := s.checkouts[strings.TrimPrefix(r.URL.Path, payFolder)]
	if found {
		s.settle(paid)
	}
	s.mu.Unlock()
	if !found {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(payPage)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) settle(paid checkout) {
	if product, isProduct := s.findProduct(paid.offer); isProduct {
		s.buy(paid.account, product)
		return
	}
	s.subscriptions[paid.account] = s.now().Add(subscriptionPeriod)
	s.planOf[paid.account] = paid.offer
	delete(s.shop.trials, paid.account)
}

type checkout struct {
	account string
	offer   string
}

func getPhaseAge(phase string) (time.Duration, bool) {
	switch phase {
	case "active":
		return -subscriptionPeriod, true
	case "grace":
		return day, true
	case "read_only":
		return 10 * day, true
	case "deleted":
		return 70 * day, true
	}
	return 0, false
}

func (s *Server) serveSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	phase := r.URL.Query().Get("phase")
	age, known := getPhaseAge(phase)
	if !known && phase != "none" {
		http.Error(w, "phase is none, active, grace, read_only or deleted", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.listAccounts(r.URL.Query().Get("account")) {
		if phase == "none" {
			delete(s.subscriptions, account)
			continue
		}
		s.subscriptions[account] = s.now().Add(-age)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listAccounts(named string) []string {
	if named != "" {
		return []string{named}
	}
	accounts := make([]string, 0, len(s.chains))
	for account := range s.chains {
		accounts = append(accounts, account)
	}
	return accounts
}

func (s *Server) ListPlans(_ context.Context, req *connect.Request[syncv1.ListPlansRequest]) (*connect.Response[syncv1.ListPlansResponse], error) {
	if _, err := s.authenticate(req.Header()); err != nil {
		return nil, err
	}
	plans := make([]*syncv1.Plan, 0, len(s.plans))
	for _, plan := range s.plans {
		plans = append(plans, &syncv1.Plan{Id: plan.ID, Name: plan.Name, QuotaBytes: plan.QuotaBytes, PriceCents: plan.PriceCents, Currency: plan.Currency})
	}
	return connect.NewResponse(&syncv1.ListPlansResponse{Plans: plans}), nil
}

func (s *Server) ChangePlan(_ context.Context, req *connect.Request[syncv1.ChangePlanRequest]) (*connect.Response[syncv1.ChangePlanResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.findPlan(req.Msg.GetPlan()); !found {
		return nil, connect.NewError(connect.CodeInvalidArgument, errUnknownPlan)
	}
	if _, paid := s.subscriptions[account]; !paid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNeverPaid)
	}
	if phase, _ := s.getPhase(account); phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNeedsSubscription)
	}
	if s.planOf[account] == req.Msg.GetPlan() {
		return nil, connect.NewError(connect.CodeAlreadyExists, errSamePlan)
	}
	s.planOf[account] = req.Msg.GetPlan()
	return connect.NewResponse(&syncv1.ChangePlanResponse{Subscription: s.getSubscriptionMessage(account)}), nil
}

// getQuota returns the bytes the account may store: the quota of its plan, or the quota of the server for an account without one.
func (s *Server) getQuota(account string) uint64 {
	if plan, found := s.findPlan(s.planOf[account]); found {
		return plan.QuotaBytes
	}
	return s.quota
}

func (s *Server) getSubscriptionMessage(account string) *syncv1.GetSubscriptionResponse {
	phase, periodEnd := s.getPhase(account)
	response := &syncv1.GetSubscriptionResponse{Phase: phase, Plan: s.planOf[account], IsTrial: s.shop.trials[account]}
	if !periodEnd.IsZero() {
		response.PeriodEndAt = periodEnd.Unix()
		response.WritesEndAt = periodEnd.Add(writeGrace).Unix()
		response.DeleteAt = periodEnd.Add(retention).Unix()
	}
	if phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		response.QuotaBytes = s.getQuota(account)
	}
	if phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE && phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED {
		response.UsedBytes = s.used[account]
	}
	return response
}
