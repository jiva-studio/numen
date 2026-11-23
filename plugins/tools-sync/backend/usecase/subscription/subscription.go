// Package subscription reads the subscription of the account, asks for a payment page and tells the person, in
// the health items of the plugin, what an ended subscription means for the changes.
package subscription

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the use cases of the subscription need.
type Deps struct {
	Subscriptions ports.Subscriptions
}

// ReadSubscription reads the subscription of the account from the server.
type ReadSubscription struct{ deps Deps }

// NewReadSubscription returns the use case over the dependencies.
func NewReadSubscription(deps Deps) *ReadSubscription { return &ReadSubscription{deps: deps} }

// Execute returns the subscription. A phase this code does not know is an error.
func (u *ReadSubscription) Execute(ctx context.Context) (domain.State, error) {
	read, err := u.deps.Subscriptions.GetSubscription(ctx)
	if err != nil {
		return domain.State{}, err
	}
	return getState(read)
}

func getState(read syncprotocol.Subscription) (domain.State, error) {
	phase, err := getPhase(read.Phase)
	if err != nil {
		return domain.State{}, err
	}
	return domain.State{
		Phase: phase, PeriodEnd: read.PeriodEnd, WritesEnd: read.WritesEnd, DeleteAt: read.DeleteAt,
		PlanID: read.PlanID, QuotaBytes: read.QuotaBytes, UsedBytes: read.UsedBytes, IsTrial: read.IsTrial,
	}, nil
}

func getPhase(wire syncprotocol.SubscriptionPhase) (domain.Phase, error) {
	switch wire {
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE:
		return domain.PhaseNone, nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE:
		return domain.PhaseActive, nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE:
		return domain.PhaseGrace, nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY:
		return domain.PhaseReadOnly, nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED:
		return domain.PhaseDeleted, nil
	case syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED:
	}
	return 0, fmt.Errorf("the sync server sent the subscription phase %v that this version does not know", wire)
}

// CreateCheckout asks the server for a payment page.
type CreateCheckout struct{ deps Deps }

// NewCreateCheckout returns the use case over the dependencies.
func NewCreateCheckout(deps Deps) *CreateCheckout { return &CreateCheckout{deps: deps} }

// Execute returns the address of the payment page for the plan. An account with an active subscription gets
// syncprotocol.ErrAlreadySubscribed.
func (u *CreateCheckout) Execute(ctx context.Context, plan string) (string, error) {
	return u.deps.Subscriptions.CreateCheckout(ctx, plan)
}

// ListPlans asks the server for the plans it sells.
type ListPlans struct{ deps Deps }

// NewListPlans returns the use case over the dependencies.
func NewListPlans(deps Deps) *ListPlans { return &ListPlans{deps: deps} }

// Execute returns the plans in the order of the server, ascending by quota.
func (u *ListPlans) Execute(ctx context.Context) ([]domain.Plan, error) {
	read, err := u.deps.Subscriptions.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]domain.Plan, 0, len(read))
	for _, plan := range read {
		plans = append(plans, domain.Plan{ID: plan.ID, Name: plan.Name, QuotaBytes: plan.QuotaBytes, PriceCents: plan.PriceCents, Currency: plan.Currency})
	}
	return plans, nil
}

// ChangePlan moves the active subscription to another plan.
type ChangePlan struct{ deps Deps }

// NewChangePlan returns the use case over the dependencies.
func NewChangePlan(deps Deps) *ChangePlan { return &ChangePlan{deps: deps} }

// Execute returns the subscription after the change. The refusals of the server are returned as it sent them:
// syncprotocol.ErrNoActiveSubscription, ErrSamePlan, ErrUnknownPlan and ErrBillingUnavailable.
func (u *ChangePlan) Execute(ctx context.Context, plan string) (domain.State, error) {
	read, err := u.deps.Subscriptions.ChangePlan(ctx, plan)
	if err != nil {
		return domain.State{}, err
	}
	return getState(read)
}

// ShopDeps are what the use cases of the products and the portal need.
type ShopDeps struct {
	Shop ports.Shop
}

// ListProducts asks the server for the products it sells that are no plans.
type ListProducts struct{ deps ShopDeps }

// NewListProducts returns the use case over the dependencies.
func NewListProducts(deps ShopDeps) *ListProducts { return &ListProducts{deps: deps} }

// Execute returns the products in the order of the server.
func (u *ListProducts) Execute(ctx context.Context) ([]domain.Product, error) {
	read, err := u.deps.Shop.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(read))
	for _, product := range read {
		products = append(products, domain.Product{
			ID: product.ID, Slug: product.Slug, Name: product.Name, PriceCents: product.PriceCents, Currency: product.Currency, OneTime: product.OneTime,
		})
	}
	return products, nil
}

// GetEntitlements asks the server for the products the account owns.
type GetEntitlements struct{ deps ShopDeps }

// NewGetEntitlements returns the use case over the dependencies.
func NewGetEntitlements(deps ShopDeps) *GetEntitlements { return &GetEntitlements{deps: deps} }

// Execute returns the slugs of the products the account owns, in the order of the server.
func (u *GetEntitlements) Execute(ctx context.Context) ([]string, error) {
	read, err := u.deps.Shop.GetEntitlements(ctx)
	if err != nil {
		return nil, err
	}
	products := make([]string, 0, len(read))
	for _, entitlement := range read {
		products = append(products, entitlement.Product)
	}
	return products, nil
}

// CreatePortalSession asks the server for the page where the person manages the subscription.
type CreatePortalSession struct{ deps ShopDeps }

// NewCreatePortalSession returns the use case over the dependencies.
func NewCreatePortalSession(deps ShopDeps) *CreatePortalSession {
	return &CreatePortalSession{deps: deps}
}

// Execute returns the address of the portal. An account that never paid by payment gets syncprotocol.ErrNoActiveSubscription.
func (u *CreatePortalSession) Execute(ctx context.Context) (string, error) {
	return u.deps.Shop.CreatePortalSession(ctx)
}
