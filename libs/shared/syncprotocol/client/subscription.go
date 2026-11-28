package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// SubscriptionClient calls the subscription service of one server with the bearer of the session.
type SubscriptionClient struct {
	rpc    syncv1connect.SubscriptionServiceClient
	server string
}

// NewSubscriptionClient returns the client of the server at address. token returns the session token and is
// read again for each call.
func NewSubscriptionClient(address string, httpClient *http.Client, token func() string) *SubscriptionClient {
	return &SubscriptionClient{rpc: syncv1connect.NewSubscriptionServiceClient(httpClient, address, newClientOptions(address, token)), server: address}
}

// GetSubscription returns the subscription of the account of the session. A response with no phase is an error.
func (c *SubscriptionClient) GetSubscription(ctx context.Context) (syncprotocol.Subscription, error) {
	response, err := c.rpc.GetSubscription(ctx, connect.NewRequest(&syncv1.GetSubscriptionRequest{}))
	if err != nil {
		return syncprotocol.Subscription{}, syncprotocol.ReadSubscriptionError(err)
	}
	return readSubscription(response.Msg)
}

func readSubscription(message *syncv1.GetSubscriptionResponse) (syncprotocol.Subscription, error) {
	if message.GetPhase() == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED {
		return syncprotocol.Subscription{}, errors.New("the sync server sent a subscription without a phase")
	}
	return syncprotocol.Subscription{
		Phase:      message.GetPhase(),
		PeriodEnd:  readUnixTime(message.GetPeriodEndAt()),
		WritesEnd:  readUnixTime(message.GetWritesEndAt()),
		DeleteAt:   readUnixTime(message.GetDeleteAt()),
		PlanID:     message.GetPlan(),
		QuotaBytes: message.GetQuotaBytes(),
		UsedBytes:  message.GetUsedBytes(),
		IsTrial:    message.GetIsTrial(),
	}, nil
}

// CreateCheckout returns the address of a payment page for the plan and the account of the session. An account with an
// active subscription gets ErrAlreadySubscribed. An address that syncprotocol.CheckPaymentAddress refuses is an
// error: the caller opens it in a browser.
func (c *SubscriptionClient) CreateCheckout(ctx context.Context, plan string) (string, error) {
	response, err := c.rpc.CreateCheckout(ctx, connect.NewRequest(&syncv1.CreateCheckoutRequest{Plan: plan}))
	if err != nil {
		return "", syncprotocol.ReadSubscriptionError(err)
	}
	address := response.Msg.GetCheckoutUrl()
	if err := syncprotocol.CheckPaymentAddress(c.server, address); err != nil {
		return "", err
	}
	return address, nil
}

func readUnixTime(seconds int64) time.Time {
	if seconds == 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}

// ListPlans returns the plans the server sells, ascending by quota. A server that sells none is
// ErrBillingUnavailable.
func (c *SubscriptionClient) ListPlans(ctx context.Context) ([]syncprotocol.Plan, error) {
	response, err := c.rpc.ListPlans(ctx, connect.NewRequest(&syncv1.ListPlansRequest{}))
	if err != nil {
		return nil, syncprotocol.ReadSubscriptionError(err)
	}
	plans := make([]syncprotocol.Plan, 0, len(response.Msg.GetPlans()))
	for _, plan := range response.Msg.GetPlans() {
		plans = append(plans, syncprotocol.Plan{
			ID: plan.GetId(), Name: plan.GetName(), QuotaBytes: plan.GetQuotaBytes(), PriceCents: plan.GetPriceCents(), Currency: plan.GetCurrency(),
		})
	}
	return plans, nil
}

// ChangePlan moves the active subscription of the account of the session to the plan and returns the subscription
// after the change. The refusals are ErrNoActiveSubscription, ErrSamePlan, ErrUnknownPlan and ErrBillingUnavailable.
func (c *SubscriptionClient) ChangePlan(ctx context.Context, plan string) (syncprotocol.Subscription, error) {
	response, err := c.rpc.ChangePlan(ctx, connect.NewRequest(&syncv1.ChangePlanRequest{Plan: plan}))
	if err != nil {
		return syncprotocol.Subscription{}, syncprotocol.ReadChangePlanError(err)
	}
	return readSubscription(response.Msg.GetSubscription())
}

// CreatePortalSession returns the address of the customer portal of the payment provider. The refusals are ErrNoActiveSubscription for an account that
// never paid by payment, ErrBillingUnavailable and ErrUnreachable when the payment provider fails.
func (c *SubscriptionClient) CreatePortalSession(ctx context.Context) (string, error) {
	response, err := c.rpc.CreatePortalSession(ctx, connect.NewRequest(&syncv1.CreatePortalSessionRequest{}))
	if err != nil {
		return "", syncprotocol.ReadChangePlanError(err)
	}
	address := response.Msg.GetPortalUrl()
	if err := syncprotocol.CheckPaymentAddress(c.server, address); err != nil {
		return "", err
	}
	return address, nil
}
