package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// SubscriptionClient calls the subscription service of one server with the bearer of the session.
type SubscriptionClient struct {
	rpc syncv1connect.SubscriptionServiceClient
}

// NewSubscriptionClient returns the client of the server at address. token returns the session token and is
// read again for each call.
func NewSubscriptionClient(address string, httpClient *http.Client, token func() string) *SubscriptionClient {
	return &SubscriptionClient{rpc: syncv1connect.NewSubscriptionServiceClient(httpClient, address, newClientOptions(token))}
}

// GetSubscription returns the subscription of the account of the session. A response with no phase is an error.
func (c *SubscriptionClient) GetSubscription(ctx context.Context) (syncprotocol.Subscription, error) {
	response, err := c.rpc.GetSubscription(ctx, connect.NewRequest(&syncv1.GetSubscriptionRequest{}))
	if err != nil {
		return syncprotocol.Subscription{}, syncprotocol.ReadSubscriptionError(err)
	}
	if response.Msg.GetPhase() == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED {
		return syncprotocol.Subscription{}, errors.New("the sync server sent a subscription without a phase")
	}
	return syncprotocol.Subscription{
		Phase:     response.Msg.GetPhase(),
		PeriodEnd: readUnixTime(response.Msg.GetPeriodEndAt()),
		WritesEnd: readUnixTime(response.Msg.GetWritesEndAt()),
		DeleteAt:  readUnixTime(response.Msg.GetDeleteAt()),
	}, nil
}

// CreateCheckout returns the address of a payment page for the plan and the account of the session. An account with an
// active subscription gets ErrAlreadySubscribed. An address that is not https, or http on a loopback host, is
// an error: the caller opens it in a browser.
func (c *SubscriptionClient) CreateCheckout(ctx context.Context, plan string) (string, error) {
	response, err := c.rpc.CreateCheckout(ctx, connect.NewRequest(&syncv1.CreateCheckoutRequest{Plan: plan}))
	if err != nil {
		return "", syncprotocol.ReadSubscriptionError(err)
	}
	address := response.Msg.GetCheckoutUrl()
	if err := checkCheckoutAddress(address); err != nil {
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

func checkCheckoutAddress(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return errors.New("the sync server sent a payment address that is not an address")
	}
	host := parsed.Hostname()
	if host == "" || !isBrowserScheme(parsed.Scheme, host) {
		return fmt.Errorf("the sync server sent a payment address that is not https: %q", parsed.Scheme)
	}
	return nil
}

func isBrowserScheme(scheme, host string) bool {
	if scheme == "https" {
		return true
	}
	if scheme != "http" {
		return false
	}
	ip, err := netip.ParseAddr(host)
	return host == "localhost" || (err == nil && ip.IsLoopback())
}

// ListPlans returns the plans the server sells, ascending by quota. A server that sells none is
// ErrBillingUnavailable.
func (c *SubscriptionClient) ListPlans(_ context.Context) ([]syncprotocol.Plan, error) {
	return nil, errors.New("not implemented")
}

// ChangePlan moves the active subscription of the account of the session to the plan and returns the subscription
// after the change. The refusals are ErrNoActiveSubscription, ErrSamePlan, ErrUnknownPlan and ErrBillingUnavailable.
func (c *SubscriptionClient) ChangePlan(_ context.Context, _ string) (syncprotocol.Subscription, error) {
	return syncprotocol.Subscription{}, errors.New("not implemented")
}
