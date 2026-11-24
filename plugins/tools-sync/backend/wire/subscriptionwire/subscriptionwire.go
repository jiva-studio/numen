// Package subscriptionwire answers the methods of the settings view about the subscription of the account.
package subscriptionwire

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

// Subscription answers the methods of the settings view about the subscription of the account.
type Subscription struct {
	identity      ports.Identity
	subscriptions ports.Subscriptions
	shop          ports.Shop
}

// New returns the methods over the dependencies of a pass.
func New(deps syncvault.Deps) *Subscription {
	return &Subscription{identity: deps.Identity, subscriptions: deps.Subscriptions, shop: deps.Shop}
}

// GetSubscription returns the subscription of the account. A device with no identity gets an empty answer and
// the server is not asked.
func (s *Subscription) GetSubscription(ctx context.Context, _ api.GetSubscriptionRequest) (api.GetSubscriptionResponse, error) {
	none := getNoSubscriptionResponse()
	_, err := s.identity.GetPublicIdentity(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		return none, nil
	}
	if err != nil {
		return none, err
	}
	state, err := usecase.NewReadSubscription(usecase.Deps{Subscriptions: s.subscriptions}).Execute(ctx)
	if err != nil {
		return none, err
	}
	return getSubscriptionResponse(state), nil
}

func getNoSubscriptionResponse() api.GetSubscriptionResponse {
	return api.GetSubscriptionResponse{HasIdentity: false, Phase: "", PeriodEndAt: "", WritesEndAt: "", DeleteAt: "", Plan: "", QuotaBytes: 0, UsedBytes: 0, IsTrial: false}
}

func getSubscriptionResponse(state subscription.State) api.GetSubscriptionResponse {
	return api.GetSubscriptionResponse{
		HasIdentity: true, Phase: getPhase(state.Phase), PeriodEndAt: formatTime(state.PeriodEnd),
		WritesEndAt: formatTime(state.WritesEnd), DeleteAt: formatTime(state.DeleteAt),
		Plan: state.PlanID, QuotaBytes: state.QuotaBytes, UsedBytes: state.UsedBytes, IsTrial: state.IsTrial,
	}
}

// ListPlans returns the plans the server sells, ascending by quota. A device with no identity gets ErrNoIdentity
// and the server is not asked.
func (s *Subscription) ListPlans(ctx context.Context, _ api.ListPlansRequest) (api.ListPlansResponse, error) {
	if _, err := s.identity.GetPublicIdentity(ctx); err != nil {
		return api.ListPlansResponse{Plans: nil}, err
	}
	plans, err := usecase.NewListPlans(usecase.Deps{Subscriptions: s.subscriptions}).Execute(ctx)
	if err != nil {
		return api.ListPlansResponse{Plans: nil}, err
	}
	response := api.ListPlansResponse{Plans: make([]api.Plan, 0, len(plans))}
	for _, plan := range plans {
		response.Plans = append(response.Plans, api.Plan{ID: plan.ID, Name: plan.Name, QuotaBytes: plan.QuotaBytes, PriceCents: plan.PriceCents, Currency: plan.Currency})
	}
	return response, nil
}

// CreateCheckout returns the address of a payment page for the plan. A device with no identity gets ErrNoIdentity
// and the server is not asked.
func (s *Subscription) CreateCheckout(ctx context.Context, request api.CreateCheckoutRequest) (api.CreateCheckoutResponse, error) {
	if _, err := s.identity.GetPublicIdentity(ctx); err != nil {
		return api.CreateCheckoutResponse{CheckoutURL: ""}, err
	}
	address, err := usecase.NewCreateCheckout(usecase.Deps{Subscriptions: s.subscriptions}).Execute(ctx, request.Plan)
	return api.CreateCheckoutResponse{CheckoutURL: address}, err
}

// ChangePlan moves the active subscription to the plan and returns the subscription after the change. A device
// with no identity gets ErrNoIdentity and the server is not asked.
func (s *Subscription) ChangePlan(ctx context.Context, request api.ChangePlanRequest) (api.ChangePlanResponse, error) {
	none := api.ChangePlanResponse{Subscription: getNoSubscriptionResponse()}
	if _, err := s.identity.GetPublicIdentity(ctx); err != nil {
		return none, err
	}
	state, err := usecase.NewChangePlan(usecase.Deps{Subscriptions: s.subscriptions}).Execute(ctx, request.Plan)
	if err != nil {
		return none, err
	}
	return api.ChangePlanResponse{Subscription: getSubscriptionResponse(state)}, nil
}

func getPhase(phase subscription.Phase) api.Phase {
	switch phase {
	case subscription.PhaseActive:
		return api.PhaseActive
	case subscription.PhaseGrace:
		return api.PhaseGrace
	case subscription.PhaseReadOnly:
		return api.PhaseReadOnly
	case subscription.PhaseDeleted:
		return api.PhaseDeleted
	case subscription.PhaseNone:
	}
	return api.PhaseNone
}

func formatTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}

// CreatePortalSession returns the address of the page where the person manages the subscription. A device with no
// identity gets ErrNoIdentity and the server is not asked.
func (s *Subscription) CreatePortalSession(ctx context.Context, _ api.CreatePortalSessionRequest) (api.CreatePortalSessionResponse, error) {
	if _, err := s.identity.GetPublicIdentity(ctx); err != nil {
		return api.CreatePortalSessionResponse{PortalURL: ""}, err
	}
	address, err := usecase.NewCreatePortalSession(usecase.ShopDeps{Shop: s.shop}).Execute(ctx)
	return api.CreatePortalSessionResponse{PortalURL: address}, err
}
