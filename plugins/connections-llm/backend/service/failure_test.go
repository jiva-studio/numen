package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestTheStatusOfAnAnswerDecidesTheKindOfTheFailure(t *testing.T) {
	cases := []struct {
		status     int
		retryAfter string
		want       failureWant
	}{
		{http.StatusUnauthorized, "", failureWant{isTryNext: true, reason: connection.ReasonRejectedKey}},
		{http.StatusForbidden, "", failureWant{isTryNext: true, reason: connection.ReasonRejectedKey}},
		{http.StatusPaymentRequired, "", failureWant{isTryNext: true, reason: connection.ReasonPaymentRequired}},
		{http.StatusTooManyRequests, "", failureWant{isTryNext: true, reason: connection.ReasonRateLimited}},
		{http.StatusTooManyRequests, "30", failureWant{isTryNext: true, reason: connection.ReasonRateLimited, retryAfter: 30 * time.Second}},
		{http.StatusBadGateway, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusServiceUnavailable, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusGatewayTimeout, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusNotFound, "", failureWant{reason: connection.ReasonInvalidRequest}},
		{http.StatusBadRequest, "", failureWant{reason: connection.ReasonInvalidRequest}},
		{http.StatusUnprocessableEntity, "", failureWant{reason: connection.ReasonInvalidRequest}},
		{http.StatusInternalServerError, "", failureWant{reason: connection.ReasonUnavailable}},
		{http.StatusTeapot, "", failureWant{reason: connection.ReasonInvalidRequest}},
	}
	for _, one := range cases {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		fake.Fail("/chat/completions", one.status, one.retryAfter)
		fake.Fail("/embeddings", one.status, one.retryAfter)

		_, err := newTestService(t).Complete(newContext(t), newCall("openrouter", address, testKey, "m"), oneUserMessage("hi"))
		checkFailure(t, err, one.want)
		_, err = newTestService(t).Embed(newContext(t), newCall("openrouter", address, testKey, "m"), []string{"a"})
		checkFailure(t, err, one.want)
	}
}

func TestARetryAfterGivenAsADateIsReadAsTheWaitFromNow(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	now := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)
	fake.Fail("/chat/completions", http.StatusTooManyRequests, now.Add(90*time.Second).Format(http.TimeFormat))
	svc := newTestServiceWith(t, newFakeClock(now))
	_, err := svc.Complete(newContext(t), newCall("openrouter", address, testKey, "m"), oneUserMessage("hi"))
	retryAfter, ok := connection.IsTryNext(err)
	if !ok || retryAfter != 90*time.Second {
		t.Errorf("retryAfter = %v, ok = %v, want 90 s", retryAfter, ok)
	}
}

func TestARetryAfterThatIsNotAWaitIsLeftOut(t *testing.T) {
	for _, value := range []string{"soon", "-5", ""} {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		fake.Fail("/chat/completions", http.StatusTooManyRequests, value)
		_, err := newTestService(t).Complete(newContext(t), newCall("openrouter", address, testKey, "m"), oneUserMessage("hi"))
		checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonRateLimited})
	}
}

func TestAServiceThatCannotBeReachedIsTryNext(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	address := ts.URL + "/v1"
	ts.Close()
	_, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", address, "", "m"), oneUserMessage("hi"))
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestAServiceThatAnswersTooSlowlyIsUnavailable(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 2*time.Second)
	fake.Delay("/embeddings", 2*time.Second)
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.1"})

	_, err := svc.Complete(newContext(t), newCall("openrouter", address, testKey, "m"), oneUserMessage("hi"))
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
	_, err = svc.Embed(newContext(t), newCall("openrouter", address, testKey, "m"), []string{"a"})
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestACanceledCallReturnsItsContextErrorAndEndsTheRequest(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 10*time.Second)
	ctx, cancel := context.WithCancel(newContext(t))
	result := make(chan error, 1)
	go func() {
		_, err := newTestService(t).Complete(ctx, newCall("openrouter", address, testKey, "m"), oneUserMessage("hi"))
		result <- err
	}()
	waitForRequest(t, fake)
	cancel()

	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	select {
	case <-fake.Aborted():
	case <-time.After(5 * time.Second):
		t.Fatal("the request to the service was not ended")
	}
}

func TestACallThatIsAlreadyCanceledNeverReachesTheService(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := newTestService(t).Embed(ctx, newCall("openrouter", address, testKey, "m"), []string{"a"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestARedirectIsNotFollowed(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the redirect target received a request")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/v1/chat/completions", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirector.Close)

	_, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", redirector.URL+"/v1", testKey, "m"), oneUserMessage("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonUnavailable})
}

func TestAFailureShowsNeitherTheKeyNorTheTextOfTheService(t *testing.T) {
	const leaked = "account acme-corp has 0 credits, key " + testKey
	for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusInternalServerError, http.StatusPaymentRequired} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			if _, err := w.Write([]byte(`{"error":{"message":"` + leaked + `"}}`)); err != nil {
				t.Error(err)
			}
		}))
		t.Cleanup(ts.Close)

		_, err := newTestService(t).Complete(newContext(t), newCall("openrouter", ts.URL+"/v1", testKey, "m"), oneUserMessage("hi"))
		if err == nil {
			t.Fatalf("status %d gave no error", status)
		}
		if text := err.Error(); strings.Contains(text, testKey) || strings.Contains(text, "acme-corp") {
			t.Errorf("status %d: error %q shows the key or the text of the service", status, text)
		}
	}
}

func waitForRequest(t *testing.T, fake *fakeopenai.Server) {
	t.Helper()
	for attempt := 0; len(fake.Requests()) == 0; attempt++ {
		if attempt == 1000 {
			t.Fatal("the service received no request")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
