package openaiapi

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
)

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
		wait       time.Duration
	}{
		{http.StatusUnauthorized, "", true, 0},
		{http.StatusPaymentRequired, "", true, 0},
		{http.StatusTooManyRequests, "30", true, 30 * time.Second},
		{http.StatusTooManyRequests, now.Add(90 * time.Second).Format(http.TimeFormat), true, 90 * time.Second},
		{http.StatusTooManyRequests, "soon", true, 0},
		{http.StatusTooManyRequests, "-5", true, 0},
		{http.StatusServiceUnavailable, "", true, 0},
		{http.StatusBadRequest, "", false, 0},
		{http.StatusInternalServerError, "", false, 0},
	}
	for _, one := range cases {
		err := mapStatusCode(one.status, one.retryAfter, now)

		wait, isTryNext := connection.IsTryNext(err)
		if isTryNext != one.isTryNext || connection.IsStop(err) == one.isTryNext || wait != one.wait {
			t.Errorf("status %d, Retry-After %q: try-next %v after %v, stop %v", one.status, one.retryAfter, isTryNext, wait, connection.IsStop(err))
		}
	}
}
