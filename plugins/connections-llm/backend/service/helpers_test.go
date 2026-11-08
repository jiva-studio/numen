package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

const testKey = "sk-test-key-1234"

func startFake(t *testing.T, config fakeopenai.Config) (*fakeopenai.Server, string) {
	t.Helper()
	fake := fakeopenai.New(config)
	ts := httptest.NewServer(fake)
	t.Cleanup(ts.Close)
	return fake, ts.URL + "/v1"
}

func newCall(typeID, address, key, model string) connection.Call {
	call := connection.Call{
		Type:    typeID,
		Fields:  map[string]string{"baseUrl": address},
		Secrets: map[string]connection.Secret{},
		Model:   model,
	}
	if key != "" {
		call.Secrets["apiKey"] = connection.Secret{Value: key}
	}
	return call
}

func newContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func oneUserMessage(text string) connection.ProviderCompleteRequest {
	return connection.ProviderCompleteRequest{Messages: []connection.Message{{Role: "user", Text: text}}}
}

type failureWant struct {
	isTryNext  bool
	reason     connection.Reason
	retryAfter time.Duration
}

func checkFailure(t *testing.T, err error, want failureWant) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %+v", want)
	}
	retryAfter, isTryNext := connection.IsTryNext(err)
	if isTryNext != want.isTryNext || connection.IsStop(err) == want.isTryNext {
		t.Fatalf("err = %v: try-next %v, stop %v, want try-next %v", err, isTryNext, connection.IsStop(err), want.isTryNext)
	}
	reason, ok := connection.GetReason(err)
	if !ok || reason != want.reason {
		t.Errorf("reason = %v, want %v", reason, want.reason)
	}
	if retryAfter != want.retryAfter {
		t.Errorf("retryAfter = %v, want %v", retryAfter, want.retryAfter)
	}
}
