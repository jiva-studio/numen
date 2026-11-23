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

var errNeedsSubscription = errors.New("the account has no subscription that allows writes")

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
	phase, periodEnd := s.getPhase(account)
	response := &syncv1.GetSubscriptionResponse{Phase: phase}
	if !periodEnd.IsZero() {
		response.PeriodEndAt = periodEnd.Unix()
		response.WritesEndAt = periodEnd.Add(writeGrace).Unix()
		response.DeleteAt = periodEnd.Add(retention).Unix()
	}
	return connect.NewResponse(response), nil
}

func (s *Server) CreateCheckout(_ context.Context, req *connect.Request[syncv1.CreateCheckoutRequest]) (*connect.Response[syncv1.CreateCheckoutResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if phase, _ := s.getPhase(account); phase == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("the subscription is active"))
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	id := hex.EncodeToString(raw)
	s.checkouts[id] = account
	return connect.NewResponse(&syncv1.CreateCheckoutResponse{CheckoutUrl: s.audience + payFolder + id}), nil
}

func (s *Server) servePay(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	account, found := s.checkouts[strings.TrimPrefix(r.URL.Path, payFolder)]
	if found {
		s.subscriptions[account] = s.now().Add(subscriptionPeriod)
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

func (s *Server) ListPlans(_ context.Context, _ *connect.Request[syncv1.ListPlansRequest]) (*connect.Response[syncv1.ListPlansResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (s *Server) ChangePlan(_ context.Context, _ *connect.Request[syncv1.ChangePlanRequest]) (*connect.Response[syncv1.ChangePlanResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (s *Server) serveUsage(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
