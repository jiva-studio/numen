package server

import (
	"errors"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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

	unauthClient := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "another-token"}))
	_, err = unauthClient.Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{}))
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
	if err == nil || handler != nil {
		t.Fatalf("NewLinkedHandler with no token = %v, %v; want empty auth token and nil handler", handler, err)
	}
	if !strings.Contains(err.Error(), "empty auth token") {
		t.Fatalf("error = %v, want empty auth token", err)
	}
}

// A mount that fails gives no handler.
func TestALinkedHandlerWhoseMountFailsIsNotMade(t *testing.T) {
	failure := errors.New("the backend cannot start")
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "linked-token"}, func(*Server) error { return failure })
	if !errors.Is(err, failure) || handler != nil {
		t.Fatalf("NewLinkedHandler = %v, %v; want the failure and no handler", handler, err)
	}
}

// The Shutdown RPC runs what the mount added with OnClose, last added first, once.
func TestALinkedHandlerRunsItsClosersWhenTheHostShutsItDown(t *testing.T) {
	var closed []string
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "linked-token"}, func(s *Server) error {
		s.OnClose(func() { closed = append(closed, "first") })
		s.OnClose(func() { closed = append(closed, "second") })
		return nil
	})
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "linked-token"}))

	for range 2 {
		if _, err := client.Shutdown(t.Context(), connect.NewRequest(&numenv1.ShutdownRequest{})); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	}

	if want := []string{"second", "first"}; !slices.Equal(closed, want) {
		t.Fatalf("closed %v, want %v", closed, want)
	}
}

// A mount that fails runs the closers it added.
func TestALinkedHandlerWhoseMountFailsClosesWhatItAdded(t *testing.T) {
	closed := false
	_, err := NewLinkedHandler(map[string]string{tokenVar: "linked-token"}, func(s *Server) error {
		s.OnClose(func() { closed = true })
		return errors.New("the backend cannot start")
	})
	if err == nil || !closed {
		t.Fatalf("NewLinkedHandler = %v, closed %v", err, closed)
	}
}
