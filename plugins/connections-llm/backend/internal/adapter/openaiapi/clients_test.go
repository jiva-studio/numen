package openaiapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
)

func TestRequestFailureReportsStatusWithoutResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "private response", http.StatusTooManyRequests)
	}))
	defer server.Close()
	gateway := New(clock.System{})
	defer gateway.Close()
	var operation, failure string
	var status int
	gateway.SetFailureReporter(func(_ context.Context, op, kind string, code int) {
		operation, failure, status = op, kind, code
	})
	call := connection.Call{Type: servicetype.OpenAI, Fields: map[string]string{servicetype.FieldBaseURL: server.URL}}
	var response map[string]any
	err := gateway.send(t.Context(), call, time.Second, endpoint{http.MethodGet, "/models"}, nil, &response)
	if err == nil || operation != "/models" || failure != "http_status" || status != http.StatusTooManyRequests {
		t.Fatalf("error %v, operation %q, failure %q, status %d", err, operation, failure, status)
	}
}

func newContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestTheClientRefusesToConnectToAForbiddenAddress(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(newContext(t), http.MethodGet, "http://0.0.0.0:"+port+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := newClients().get(false).Do(request)
	if err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("the client connected to 0.0.0.0")
	}
	if !address.IsError(err) {
		t.Errorf("err = %v, want a refusal", err)
	}
}

func TestTheClientConnectsToTheLoopbackWhenAKeyTravelsOverHTTP(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)
	ctx := context.WithValue(newContext(t), loopbackOnlyKey{}, true)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := newClients().get(true).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
}

func TestTheStatusOfAnAnswerDecidesTheKindOfFailureAndTheWaitOfARetry(t *testing.T) {
	now := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		status     int
		retryAfter string
		isTryNext  bool
		reason     connection.Reason
		wait       time.Duration
	}{
		{http.StatusUnauthorized, "", true, connection.ReasonRejectedKey, 0},
		{http.StatusForbidden, "", true, connection.ReasonRejectedKey, 0},
		{http.StatusPaymentRequired, "", true, connection.ReasonPaymentRequired, 0},
		{http.StatusTooManyRequests, "  30  ", true, connection.ReasonRateLimited, 30 * time.Second},
		{http.StatusTooManyRequests, "2147483647", true, connection.ReasonRateLimited, 2147483647 * time.Second},
		{http.StatusTooManyRequests, "2147483648", true, connection.ReasonRateLimited, 0},
		{http.StatusTooManyRequests, now.Add(90 * time.Second).Format(http.TimeFormat), true, connection.ReasonRateLimited, 90 * time.Second},
		{http.StatusTooManyRequests, now.Format(http.TimeFormat), true, connection.ReasonRateLimited, 0},
		{http.StatusTooManyRequests, "soon", true, connection.ReasonRateLimited, 0},
		{http.StatusTooManyRequests, "-1", true, connection.ReasonRateLimited, 0},
		{http.StatusTooManyRequests, "-5", true, connection.ReasonRateLimited, 0},
		{http.StatusBadGateway, "", true, connection.ReasonUnavailable, 0},
		{http.StatusServiceUnavailable, "", true, connection.ReasonUnavailable, 0},
		{http.StatusGatewayTimeout, "", true, connection.ReasonUnavailable, 0},
		{399, "", false, connection.ReasonUnavailable, 0},
		{http.StatusBadRequest, "", false, connection.ReasonInvalidRequest, 0},
		{499, "", false, connection.ReasonInvalidRequest, 0},
		{http.StatusInternalServerError, "", false, connection.ReasonUnavailable, 0},
	}
	for _, one := range cases {
		err := mapStatusCode(one.status, one.retryAfter, now)

		wait, isTryNext := connection.IsTryNext(err)
		if isTryNext != one.isTryNext || connection.IsStop(err) == one.isTryNext || wait != one.wait || err.Error() != one.reason.String() {
			t.Errorf("status %d, Retry-After %q: try-next %v after %v, stop %v, err %q, want reason %q",
				one.status, one.retryAfter, isTryNext, wait, connection.IsStop(err), err.Error(), one.reason)
		}
	}

	if got := parseRetryAfter("soon", now); got != 0 {
		t.Errorf("parseRetryAfter(soon) = %v, want 0", got)
	}
	if got := parseRetryAfter("-1", now); got != 0 {
		t.Errorf("parseRetryAfter(-1) = %v, want 0", got)
	}
	if got := parseRetryAfter(now.Format(http.TimeFormat), now); got != 0 {
		t.Errorf("parseRetryAfter(now) = %v, want 0", got)
	}
	if got := parseRetryAfter(now.Format(http.TimeFormat), now.Add(time.Nanosecond)); got != 0 {
		t.Errorf("parseRetryAfter(now - 1ns) = %v, want 0", got)
	}
}

func TestClientPoolSettingsAndTransportClassification(t *testing.T) {
	if maxResponseBytes != 64*1024*1024 {
		t.Errorf("maxResponseBytes = %d, want %d", maxResponseBytes, 64*1024*1024)
	}
	pool := newClients()
	defer pool.close()
	if pool.get(true) == nil || pool.get(false) == nil || pool.get(true) == pool.get(false) {
		t.Errorf("pool clients must be distinct non-nil instances: loopback=%p general=%p", pool.get(true), pool.get(false))
	}
	transport, ok := newClient().Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.TLSHandshakeTimeout != 30*time.Second || transport.IdleConnTimeout != 90*time.Second || transport.MaxIdleConnsPerHost != 8 {
		t.Errorf("transport timeouts = (%v, %v, %d), want (30s, 90s, 8)",
			transport.TLSHandshakeTimeout, transport.IdleConnTimeout, transport.MaxIdleConnsPerHost)
	}

	ctxWithLoopbackFalse := context.WithValue(t.Context(), loopbackOnlyKey{}, false)
	if err := controlDial(ctxWithLoopbackFalse, "tcp", "93.184.216.34:443", nil); err != nil {
		t.Errorf("controlDial(loopbackOnly=false, public IP) = %v, want nil", err)
	}
	ctxWithLoopbackTrue := context.WithValue(t.Context(), loopbackOnlyKey{}, true)
	if err := controlDial(ctxWithLoopbackTrue, "tcp", "93.184.216.34:443", nil); !address.IsError(err) {
		t.Errorf("controlDial(loopbackOnly=true, public IP) = %v, want address error", err)
	}

	if _, isTryNext := connection.IsTryNext(classifyTransportError(t.Context(), errors.New("eof"))); !isTryNext {
		t.Error("classifyTransportError with live context = not try-next, want try-next")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := classifyTransportError(canceled, errors.New("eof")); !errors.Is(err, context.Canceled) {
		t.Errorf("classifyTransportError with canceled context = %v, want context.Canceled", err)
	}
}
