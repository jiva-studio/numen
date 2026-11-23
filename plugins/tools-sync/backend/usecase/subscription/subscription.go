// Package subscription reads the subscription of the account, asks for a payment page and tells the person, in
// the health items of the plugin, what an ended subscription means for the changes.
package subscription

import (
	"context"
	"errors"
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
	phase, err := getPhase(read.Phase)
	if err != nil {
		return domain.State{}, err
	}
	return domain.State{Phase: phase, PeriodEnd: read.PeriodEnd, WritesEnd: read.WritesEnd, DeleteAt: read.DeleteAt}, nil
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
func (u *ListPlans) Execute(_ context.Context) ([]domain.Plan, error) {
	return nil, errors.New("not implemented")
}

// ChangePlan moves the active subscription to another plan.
type ChangePlan struct{ deps Deps }

// NewChangePlan returns the use case over the dependencies.
func NewChangePlan(deps Deps) *ChangePlan { return &ChangePlan{deps: deps} }

// Execute returns the subscription after the change. The refusals of the server are returned as it sent them:
// syncprotocol.ErrNoActiveSubscription, ErrSamePlan, ErrUnknownPlan and ErrBillingUnavailable.
func (u *ChangePlan) Execute(_ context.Context, _ string) (domain.State, error) {
	return domain.State{}, errors.New("not implemented")
}
