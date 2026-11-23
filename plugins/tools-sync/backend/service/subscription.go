package service

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
)

// GetSubscription returns the subscription of the account. A device with no identity gets an empty answer and
// the server is not asked.
func (b *Backend) GetSubscription(ctx context.Context, _ api.GetSubscriptionRequest) (api.GetSubscriptionResponse, error) {
	none := api.GetSubscriptionResponse{HasIdentity: false, Phase: "", PeriodEndAt: "", WritesEndAt: "", DeleteAt: "", Plan: "", QuotaBytes: 0, UsedBytes: 0, IsTrial: false}
	_, err := b.deps.Identity.GetPublicIdentity(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		return none, nil
	}
	if err != nil {
		return none, err
	}
	state, err := usecase.NewReadSubscription(usecase.Deps{Subscriptions: b.deps.Subscriptions}).Execute(ctx)
	if err != nil {
		return none, err
	}
	return getSubscriptionResponse(state), nil
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
func (b *Backend) ListPlans(ctx context.Context, _ api.ListPlansRequest) (api.ListPlansResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.ListPlansResponse{Plans: nil}, err
	}
	plans, err := usecase.NewListPlans(usecase.Deps{Subscriptions: b.deps.Subscriptions}).Execute(ctx)
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
func (b *Backend) CreateCheckout(ctx context.Context, request api.CreateCheckoutRequest) (api.CreateCheckoutResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.CreateCheckoutResponse{CheckoutURL: ""}, err
	}
	address, err := usecase.NewCreateCheckout(usecase.Deps{Subscriptions: b.deps.Subscriptions}).Execute(ctx, request.Plan)
	return api.CreateCheckoutResponse{CheckoutURL: address}, err
}

// ChangePlan moves the active subscription to the plan and returns the subscription after the change. A device
// with no identity gets ErrNoIdentity and the server is not asked.
func (b *Backend) ChangePlan(ctx context.Context, request api.ChangePlanRequest) (api.ChangePlanResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.ChangePlanResponse{Subscription: getSubscriptionResponse(subscription.State{})}, err
	}
	state, err := usecase.NewChangePlan(usecase.Deps{Subscriptions: b.deps.Subscriptions}).Execute(ctx, request.Plan)
	if err != nil {
		return api.ChangePlanResponse{Subscription: getSubscriptionResponse(subscription.State{})}, err
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

// ListProducts returns the products the server sells that are no plans. A device with no identity gets ErrNoIdentity
// and the server is not asked.
func (b *Backend) ListProducts(ctx context.Context, _ api.ListProductsRequest) (api.ListProductsResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.ListProductsResponse{Products: nil}, err
	}
	products, err := usecase.NewListProducts(usecase.ShopDeps{Shop: b.deps.Shop}).Execute(ctx)
	if err != nil {
		return api.ListProductsResponse{Products: nil}, err
	}
	response := api.ListProductsResponse{Products: make([]api.Product, 0, len(products))}
	for _, product := range products {
		response.Products = append(response.Products, api.Product{
			ID: product.ID, Slug: product.Slug, Name: product.Name, PriceCents: product.PriceCents, Currency: product.Currency, OneTime: product.OneTime,
		})
	}
	return response, nil
}

// GetEntitlements returns the products the account owns. A device with no identity gets ErrNoIdentity and the server is
// not asked.
func (b *Backend) GetEntitlements(ctx context.Context, _ api.GetEntitlementsRequest) (api.GetEntitlementsResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.GetEntitlementsResponse{Entitlements: nil}, err
	}
	products, err := usecase.NewGetEntitlements(usecase.ShopDeps{Shop: b.deps.Shop}).Execute(ctx)
	if err != nil {
		return api.GetEntitlementsResponse{Entitlements: nil}, err
	}
	response := api.GetEntitlementsResponse{Entitlements: make([]api.Entitlement, 0, len(products))}
	for _, product := range products {
		response.Entitlements = append(response.Entitlements, api.Entitlement{Product: product})
	}
	return response, nil
}

// CreatePortalSession returns the address of the page where the person manages the subscription. A device with no
// identity gets ErrNoIdentity and the server is not asked.
func (b *Backend) CreatePortalSession(ctx context.Context, _ api.CreatePortalSessionRequest) (api.CreatePortalSessionResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.CreatePortalSessionResponse{PortalURL: ""}, err
	}
	address, err := usecase.NewCreatePortalSession(usecase.ShopDeps{Shop: b.deps.Shop}).Execute(ctx)
	return api.CreatePortalSessionResponse{PortalURL: address}, err
}
