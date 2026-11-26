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
)

const (
	subscribePeriod  = 30 * 24 * time.Hour
	subscribeTimeout = 20 * time.Second
	subscribePoll    = 200 * time.Millisecond
)

// Subscribe buys the plan for the account of this computer through the payment stand-in of the stack, as a person does: the computer creates a checkout, the
// control API of the stand-in pays it and the server takes the signed webhook. It returns when the server reports the subscription as active.
func (c *Computer) Subscribe(t *testing.T, stack *Stack, plan string) {
	t.Helper()
	if stack.PaymentURL == "" {
		t.Fatalf("no payment stand-in: set %s to the control API of the stack", EnvPaymentURL)
	}
	client := syncclient.NewSubscriptionClient(c.Server.URL, c.NewHTTP(), c.getToken)
	address, err := client.CreateCheckout(t.Context(), plan)
	if err != nil {
		t.Fatalf("create a checkout for %s: %v", plan, err)
	}
	parsed, err := url.Parse(address)
	if err != nil || !strings.HasPrefix(parsed.Path, "/pay/") {
		t.Fatalf("the checkout address %q has no /pay/<id> path", address)
	}
	completeCheckout(t, stack.PaymentURL, strings.TrimPrefix(parsed.Path, "/pay/"), c.Clock.Now().Add(subscribePeriod))

	deadline := GetNow().Add(subscribeTimeout)
	for {
		got, err := client.GetSubscription(t.Context())
		if err == nil && got.Phase == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE && got.PlanID == plan {
			return
		}
		if GetNow().After(deadline) {
			t.Fatalf("the subscription to %s is not active after %s: phase %v, error %v", plan, subscribeTimeout, got.Phase, err)
		}
		time.Sleep(subscribePoll)
	}
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
