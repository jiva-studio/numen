package server

import (
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// A linked handler answers with the token its variables hold, and hands Configure to the server
// it was mounted on.
func TestALinkedHandlerAnswersWithTheTokenOfItsVariables(t *testing.T) {
	var configured map[string]string
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "linked-token"}, func(s *Server) error {
		s.SetUpdateConfig(func(values map[string]string) { configured = values })
		return nil
	})
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	refused := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "another-token"}))
	_, err = refused.Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("Configure with another token = %v, want unauthenticated", err)
	}

	client := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "linked-token"}))
	_, err = client.Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{"model": "small-model"},
	}))
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if configured["model"] != "small-model" {
		t.Fatalf("configured %v, want model small-model", configured)
	}
}

// A linked handler takes its token from its variables alone.
func TestALinkedHandlerReadsNoTokenFromTheEnvironment(t *testing.T) {
	t.Setenv(tokenVar, "process-token")
	handler, err := NewLinkedHandler(map[string]string{}, func(*Server) error { return nil })
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL)
	if _, err := client.Ping(t.Context(), connect.NewRequest(&numenv1.PingRequest{})); err != nil {
		t.Fatalf("Ping with no token to a server holding none = %v", err)
	}
}

// A mount that fails gives no handler.
func TestALinkedHandlerWhoseMountFailsIsNotMade(t *testing.T) {
	failure := errors.New("the backend cannot start")
	handler, err := NewLinkedHandler(map[string]string{}, func(*Server) error { return failure })
	if !errors.Is(err, failure) || handler != nil {
		t.Fatalf("NewLinkedHandler = %v, %v; want the failure and no handler", handler, err)
	}
}
