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
	none := api.GetSubscriptionResponse{HasIdentity: false, Phase: "", PeriodEndAt: "", WritesEndAt: "", DeleteAt: ""}
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
	return api.GetSubscriptionResponse{
		HasIdentity: true, Phase: getPhase(state.Phase), PeriodEndAt: formatTime(state.PeriodEnd),
		WritesEndAt: formatTime(state.WritesEnd), DeleteAt: formatTime(state.DeleteAt),
	}, nil
}

// CreateCheckout returns the address of a payment page. A device with no identity gets ErrNoIdentity and the
// server is not asked.
func (b *Backend) CreateCheckout(ctx context.Context, _ api.CreateCheckoutRequest) (api.CreateCheckoutResponse, error) {
	if _, err := b.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return api.CreateCheckoutResponse{CheckoutURL: ""}, err
	}
	address, err := usecase.NewCreateCheckout(usecase.Deps{Subscriptions: b.deps.Subscriptions}).Execute(ctx)
	return api.CreateCheckoutResponse{CheckoutURL: address}, err
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
