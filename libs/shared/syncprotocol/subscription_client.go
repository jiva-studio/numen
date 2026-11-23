package syncprotocol

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// SubscriptionPhase is the phase of a subscription as the server tells it.
type SubscriptionPhase = syncv1.SubscriptionPhase

// Subscription is the subscription of an account. A time the server did not send is the zero time.
type Subscription struct {
	Phase     SubscriptionPhase
	PeriodEnd time.Time
	WritesEnd time.Time
	DeleteAt  time.Time
}

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
func (c *SubscriptionClient) GetSubscription(ctx context.Context) (Subscription, error) {
	response, err := c.rpc.GetSubscription(ctx, connect.NewRequest(&syncv1.GetSubscriptionRequest{}))
	if err != nil {
		return Subscription{}, readSubscriptionError(err)
	}
	if response.Msg.GetPhase() == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED {
		return Subscription{}, errors.New("the sync server sent a subscription without a phase")
	}
	return Subscription{
		Phase:     response.Msg.GetPhase(),
		PeriodEnd: readUnixTime(response.Msg.GetPeriodEndAt()),
		WritesEnd: readUnixTime(response.Msg.GetWritesEndAt()),
		DeleteAt:  readUnixTime(response.Msg.GetDeleteAt()),
	}, nil
}

// CreateCheckout returns the address of a payment page for the account of the session. An account with an
// active subscription gets ErrAlreadySubscribed. An address that is not https, or http on a loopback host, is
// an error: the caller opens it in a browser.
func (c *SubscriptionClient) CreateCheckout(ctx context.Context) (string, error) {
	response, err := c.rpc.CreateCheckout(ctx, connect.NewRequest(&syncv1.CreateCheckoutRequest{}))
	if err != nil {
		return "", readSubscriptionError(err)
	}
	address := response.Msg.GetCheckoutUrl()
	if err := checkCheckoutAddress(address); err != nil {
		return "", err
	}
	return address, nil
}

func readSubscriptionError(err error) error {
	if failure, ok := errors.AsType[*connect.Error](err); ok {
		switch failure.Code() {
		case connect.CodeAlreadyExists:
			return refusalError{sentinel: ErrAlreadySubscribed, cause: failure}
		case connect.CodeUnimplemented:
			return refusalError{sentinel: ErrBillingUnavailable, cause: failure}
		}
	}
	return readError(err)
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
