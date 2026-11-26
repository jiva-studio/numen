package synctest

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/wiretest"
)

const (
	subscribePeriod  = 30 * 24 * time.Hour
	subscribeTimeout = 20 * time.Second
	subscribePoll    = 200 * time.Millisecond
	day              = 24 * time.Hour
)

// Subscribe buys the plan for the account of this computer through the payment stand-in of the stack, as a person does: the computer creates a checkout, the
// control API of the stand-in pays it and the server takes the signed webhook. It returns when the server reports the subscription as active.
func (c *Computer) Subscribe(t *testing.T, stack *Stack, plan string) {
	t.Helper()
	c.subscribeEnding(t, stack, plan, c.Clock.Now().Add(subscribePeriod), syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE)
}

// SetPhase buys the default plan with a period that ends so far back that the server finds the account in the phase: "active", "grace", "read_only" or "deleted".
// The phase "none" buys nothing. An account that holds a subscription that is not over cannot be moved by a second call.
func (c *Computer) SetPhase(t *testing.T, phase string) {
	t.Helper()
	now := c.Clock.Now()
	switch phase {
	case "none":
	case "active":
		c.subscribeEnding(t, c.Server.Stack, DefaultPlan, now.Add(subscribePeriod), syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE)
	case "grace":
		c.subscribeEnding(t, c.Server.Stack, DefaultPlan, now.Add(-day), syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE)
	case "read_only":
		c.subscribeEnding(t, c.Server.Stack, DefaultPlan, now.Add(-8*day), syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY)
	case "deleted":
		c.subscribeEnding(t, c.Server.Stack, DefaultPlan, now.Add(-61*day), syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED)
	default:
		t.Fatalf("unknown phase %q", phase)
	}
}

func (c *Computer) subscribeEnding(t *testing.T, stack *Stack, plan string, periodEnd time.Time, want syncv1.SubscriptionPhase) {
	t.Helper()
	if stack.PaymentURL == "" {
		t.Fatalf("no payment stand-in: set %s to the control API of the stack", EnvPaymentURL)
	}
	client := syncclient.NewSubscriptionClient(c.Server.URL, c.NewHTTP(), c.getToken)
	address, err := createCheckout(t, client, plan)
	if err != nil {
		t.Fatalf("create a checkout for %s: %v", plan, err)
	}
	CompleteCheckout(t, stack, address, periodEnd)

	deadline := GetNow().Add(subscribeTimeout)
	for {
		got, err := client.GetSubscription(t.Context())
		if err == nil && got.Phase == want && got.PlanID == plan {
			return
		}
		if GetNow().After(deadline) {
			t.Fatalf("the subscription to %s is not in the phase %v after %s: phase %v, error %v", plan, want, subscribeTimeout, got.Phase, err)
		}
		time.Sleep(subscribePoll)
	}
}

func CompleteCheckout(t *testing.T, stack *Stack, address string, periodEnd time.Time) {
	t.Helper()
	parsed, err := url.Parse(address)
	if err != nil || !strings.HasPrefix(parsed.Path, "/pay/") {
		t.Fatalf("the checkout address %q has no /pay/<id> path", address)
	}
	completeCheckout(t, stack.PaymentURL, strings.TrimPrefix(parsed.Path, "/pay/"), periodEnd)
}

func completeCheckout(t *testing.T, control, checkout string, periodEnd time.Time) {
	t.Helper()
	target := fmt.Sprintf("%s/pay?checkout=%s&period_end=%s", strings.TrimSuffix(control, "/"), url.QueryEscape(checkout), strconv.FormatInt(periodEnd.Unix(), 10))
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("pay the checkout: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close the answer of the control API: %v", err)
		}
	}()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<10))
	if err != nil {
		t.Fatalf("read the answer of the control API: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("the control API of the stand-in answered %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
}

func createCheckout(t *testing.T, client *syncclient.SubscriptionClient, plan string) (string, error) {
	t.Helper()
	deadline := GetNow().Add(subscribeTimeout)
	for {
		address, err := client.CreateCheckout(t.Context(), plan)
		if !wiretest.IsResourceExhausted(err) || GetNow().After(deadline) {
			return address, err
		}
		time.Sleep(subscribePoll)
	}
}
