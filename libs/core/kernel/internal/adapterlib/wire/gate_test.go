package wire

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

const harborPhrase = "open sesame for the harbor"

func recordCreateProviderConnection(t *testing.T, wantFailure bool, serve func() error) []logs.Entry {
	t.Helper()
	var gate Gate
	var mu sync.Mutex
	var entries []logs.Entry
	gate.SetCallTrace(fixedClock{}, func(entry logs.Entry) {
		mu.Lock()
		defer mu.Unlock()
		entries = append(entries, entry)
	})
	handler := connect.NewUnaryHandler(
		numenv1connect.ProviderConnectionServiceCreateProviderConnectionProcedure,
		func(context.Context, *connect.Request[v1.CreateProviderConnectionRequest]) (*connect.Response[v1.CreateProviderConnectionResponse], error) {
			if err := serve(); err != nil {
				return nil, err
			}
			return connect.NewResponse(&v1.CreateProviderConnectionResponse{}), nil
		},
		gate.CountQuestions(),
	)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.ProviderConnectionServiceCreateProviderConnectionProcedure, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := numenv1connect.NewProviderConnectionServiceClient(server.Client(), server.URL)
	_, err := client.CreateProviderConnection(t.Context(), connect.NewRequest(&v1.CreateProviderConnectionRequest{
		TypeId:  "harbor",
		Name:    "Harbor",
		Secrets: map[string]string{"apiKey": harborPhrase},
	}))
	if (err != nil) != wantFailure {
		t.Fatalf("call error = %v, want failure %v", err, wantFailure)
	}
	mu.Lock()
	defer mu.Unlock()
	return entries
}

func assertNoSecret(t *testing.T, entries []logs.Entry) {
	t.Helper()
	if len(entries) != 1 {
		t.Fatalf("recorded %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if strings.Contains(entry.Message, harborPhrase) {
		t.Errorf("message holds the secret: %q", entry.Message)
	}
	for key, value := range entry.Properties {
		if strings.Contains(value, harborPhrase) {
			t.Errorf("property %s holds the secret: %q", key, value)
		}
	}
	for _, key := range []string{"request", "response"} {
		if _, found := entry.Properties[key]; found {
			t.Errorf("property %s is recorded, want no body", key)
		}
	}
}

func TestCallTraceOfSuccessfulCallRecordsNoSecret(t *testing.T) {
	entries := recordCreateProviderConnection(t, false, func() error { return nil })
	assertNoSecret(t, entries)
	if got := entries[0].Properties["method"]; got != `"CreateProviderConnection"` {
		t.Errorf("method = %s, want the method name", got)
	}
}

func TestCallTraceOfFailedCallRecordsNoSecretAndErrorCode(t *testing.T) {
	entries := recordCreateProviderConnection(t, true, func() error {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("rejected key "+harborPhrase))
	})
	assertNoSecret(t, entries)
	if got := entries[0].Properties["error"]; got != `"invalid_argument"` {
		t.Errorf("error = %s, want the error code", got)
	}
}
