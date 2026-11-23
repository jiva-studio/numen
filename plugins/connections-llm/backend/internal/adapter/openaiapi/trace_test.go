package openaiapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/logs"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
)

func TestAFailureIsClassifiedByItsCause(t *testing.T) {
	for want, err := range map[string]error{
		"timeout":          context.DeadlineExceeded,
		"canceled":         context.Canceled,
		"connection_reset": fmt.Errorf("read: %w", syscall.ECONNRESET),
		"unexpected_eof":   io.ErrUnexpectedEOF,
		"eof":              io.EOF,
		"other":            errors.New("boom"),
	} {
		if got := classifyFailure(err); got != want {
			t.Errorf("classifyFailure(%v) = %q, want %q", err, got, want)
		}
	}
	if got := classifyFailure(&net.DNSError{IsTimeout: true}); got != "timeout" {
		t.Errorf("a network timeout = %q", got)
	}
}

func TestLoggedBodyIsTruncatedAndKeyFree(t *testing.T) {
	got := truncateBody([]byte(`{"k":"sk-secret","h":"Bearer abc.def"}`+strings.Repeat("x", 5000)), "sk-secret")

	if strings.Contains(got, "sk-secret") || strings.Contains(got, "abc.def") || !strings.HasSuffix(got, "…") ||
		len(got) > maxLoggedBodyBytes+len("…")+2*len(redacted) {
		t.Errorf("body = %q", got)
	}
}

type entry struct {
	level      logs.Level
	message    string
	properties map[string]any
}

func traceServer(t *testing.T, handler http.HandlerFunc) (*Gateway, *[]entry, connection.Call) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	gateway := New(clock.System{})
	t.Cleanup(gateway.Close)
	var entries []entry
	gateway.SetTracer(func(_ context.Context, level logs.Level, message string, properties map[string]any) {
		entries = append(entries, entry{level, message, properties})
	})
	call := connection.Call{
		Type: servicetype.OpenAI, Fields: map[string]string{servicetype.FieldBaseURL: server.URL},
		Secrets: map[string]connection.Secret{servicetype.SecretAPIKey: {Value: "sk-secret"}},
	}
	return gateway, &entries, call
}

func TestAnswerTraceKeepsOnlyChosenHeaders(t *testing.T) {
	gateway, entries, call := traceServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-ID", "req-1")
		w.Header().Set("Set-Cookie", "session=private")
		if _, err := io.WriteString(w, `{"echo":"sk-secret"}`); err != nil {
			t.Error(err)
		}
	})
	var out map[string]any

	err := gateway.send(t.Context(), call, time.Second, endpoint{http.MethodPost, "/chat/completions"}, map[string]string{"model": "m"}, &out)

	if err != nil {
		t.Fatal(err)
	}
	var response entry
	for _, one := range *entries {
		if strings.Contains(fmt.Sprint(one.properties), "sk-secret") {
			t.Errorf("an entry holds the key: %v", one)
		}
		if one.message == "Connection response received" {
			response = one
		}
	}
	if response.properties["status"] != 200 || response.properties["x-request-id"] != "req-1" ||
		response.properties["response_bytes"] != len(`{"echo":"sk-secret"}`) || response.properties["limit_ms"] != int64(1000) {
		t.Errorf("response entry = %v", response)
	}
	if _, has := response.properties["set-cookie"]; has {
		t.Error("a header that is not kept was logged")
	}
	if len(*entries) != 4 || (*entries)[0].message != "Connection request sent" || (*entries)[0].level != logs.Info ||
		(*entries)[1].level != logs.Debug || (*entries)[3].level != logs.Debug {
		t.Errorf("entries = %v", *entries)
	}
}

func TestABodyThatEndsEarlyIsTracedAsAFailureOfTheRead(t *testing.T) {
	gateway, entries, call := traceServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		if _, err := io.WriteString(w, "{"); err != nil {
			t.Error(err)
		}
	})
	var out map[string]any

	err := gateway.send(t.Context(), call, time.Second, endpoint{http.MethodGet, "/models"}, nil, &out)

	last := (*entries)[len(*entries)-1]
	if err == nil || last.level != logs.Warning || last.properties["stage"] != "read_response" ||
		last.properties["failure"] != "unexpected_eof" || last.properties["status"] != 200 {
		t.Errorf("err = %v, entries = %v", err, *entries)
	}
}
