package ai

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type fakeHost struct {
	numenv1connect.UnimplementedHostServiceHandler
	code   numenv1.CompletionErrorCode
	answer *numenv1.CompletionAnswer
	got    *numenv1.CompletionQuestion
	auth   string
}

func (f *fakeHost) Complete(_ context.Context, req *connect.Request[numenv1.HostServiceCompleteRequest]) (*connect.Response[numenv1.HostServiceCompleteResponse], error) {
	f.got = req.Msg.GetQuestion()
	f.auth = req.Header().Get("Authorization")
	return connect.NewResponse(&numenv1.HostServiceCompleteResponse{Answer: f.answer, Error: f.code}), nil
}

func startHost(t *testing.T, h *fakeHost) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newQuestion() *numenv1.CompletionQuestion {
	return &numenv1.CompletionQuestion{
		Purpose: "proofreading",
		Messages: []*numenv1.CompletionMessage{
			{Role: "user", Text: "fix this"},
		},
	}
}

func TestClientCompleteSendsQuestionAndTokenAndReturnsText(t *testing.T) {
	host := &fakeHost{answer: &numenv1.CompletionAnswer{Text: "fixed"}}
	srv := startHost(t, host)

	text, err := NewClient(srv.Client(), srv.URL, "secret").Complete(context.Background(), newQuestion())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "fixed" {
		t.Errorf("got %q, want %q", text, "fixed")
	}
	if host.auth != "Bearer secret" {
		t.Errorf("got authorization %q, want %q", host.auth, "Bearer secret")
	}
	if host.got.GetPurpose() != "proofreading" || host.got.GetMessages()[0].GetText() != "fix this" {
		t.Errorf("host received %v", host.got)
	}
}

func TestClientCompleteOmitsAuthorizationWithoutToken(t *testing.T) {
	host := &fakeHost{answer: &numenv1.CompletionAnswer{Text: "ok"}}
	srv := startHost(t, host)

	if _, err := NewClient(srv.Client(), srv.URL, "").Complete(context.Background(), newQuestion()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.auth != "" {
		t.Errorf("got authorization %q, want none", host.auth)
	}
}

func TestClientCompleteReturnsErrNoAnswerWhenHostSendsNone(t *testing.T) {
	srv := startHost(t, &fakeHost{})

	_, err := NewClient(srv.Client(), srv.URL, "t").Complete(context.Background(), newQuestion())

	if !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("got %v, want ErrNoAnswer", err)
	}
}

func TestClientCompleteReturnsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)

	_, err := NewClient(srv.Client(), srv.URL, "t").Complete(context.Background(), newQuestion())

	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("got %v, want a connect error", err)
	}
}

func TestCompleteReadsHostFromEnvironment(t *testing.T) {
	host := &fakeHost{answer: &numenv1.CompletionAnswer{Text: "env"}}
	srv := startHost(t, host)
	t.Setenv(HostPortEnv, strconv.Itoa(srv.Listener.Addr().(*net.TCPAddr).Port))
	t.Setenv(PluginTokenEnv, "tok")

	text, err := Complete(context.Background(), newQuestion())

	if err != nil || text != "env" {
		t.Fatalf("got %q, %v", text, err)
	}
	if host.auth != "Bearer tok" {
		t.Errorf("got authorization %q", host.auth)
	}
}

func TestCompleteFailsWithoutHostPort(t *testing.T) {
	t.Setenv(HostPortEnv, "")

	_, err := Complete(context.Background(), newQuestion())

	if !errors.Is(err, ErrNoHost) {
		t.Fatalf("got %v, want ErrNoHost", err)
	}
}

func TestClientCompleteReturnsRefusalOfHost(t *testing.T) {
	cases := map[numenv1.CompletionErrorCode]error{
		numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE:    ErrNoRoute,
		numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE: ErrUnavailable,
		numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT:  ErrNoConsent,
	}
	for code, want := range cases {
		srv := startHost(t, &fakeHost{code: code})

		_, err := NewClient(srv.Client(), srv.URL, "t").Complete(context.Background(), newQuestion())

		if !errors.Is(err, want) {
			t.Errorf("%v: got %v, want %v", code, err, want)
		}
	}
}

func TestReadRefusalWalksEveryCode(t *testing.T) {
	seen := map[error]numenv1.CompletionErrorCode{}
	for value := range numenv1.CompletionErrorCode_name {
		code := numenv1.CompletionErrorCode(value)
		err := readRefusal(code)
		if code == numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED {
			if err != nil {
				t.Errorf("unspecified must be no refusal, got %v", err)
			}
			continue
		}
		if err == nil || err.Error() == "" {
			t.Fatalf("%v has no error", code)
		}
		if other, dup := seen[err]; dup {
			t.Errorf("%v and %v share an error", code, other)
		}
		seen[err] = code
	}
	if err := readRefusal(numenv1.CompletionErrorCode(99)); err == nil {
		t.Error("an unknown code must be an error")
	}
}

// A client made from the variables of a linked plugin asks the host they name, and reads nothing
// from the environment.
func TestClientFromVarsReadsTheHostFromItsVariables(t *testing.T) {
	host := &fakeHost{answer: &numenv1.CompletionAnswer{Text: "fixed"}}
	srv := startHost(t, host)
	t.Setenv(HostPortEnv, "")
	t.Setenv(PluginTokenEnv, "process-token")

	client, err := NewClientFromVars(map[string]string{
		HostPortEnv:    strconv.Itoa(srv.Listener.Addr().(*net.TCPAddr).Port),
		PluginTokenEnv: "linked-token",
	})
	if err != nil {
		t.Fatalf("NewClientFromVars: %v", err)
	}
	text, err := client.Complete(t.Context(), newQuestion())
	if err != nil || text != "fixed" {
		t.Fatalf("Complete = %q, %v", text, err)
	}
	if host.auth != "Bearer linked-token" {
		t.Errorf("authorization = %q, want the token of the variables", host.auth)
	}
}

func TestClientFromVarsFailsWithoutHostPort(t *testing.T) {
	if _, err := NewClientFromVars(map[string]string{PluginTokenEnv: "tok"}); !errors.Is(err, ErrNoHost) {
		t.Fatalf("got %v, want ErrNoHost", err)
	}
}
