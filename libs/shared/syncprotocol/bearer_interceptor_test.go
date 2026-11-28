package syncprotocol_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type authorizationRecorder struct {
	syncv1connect.UnimplementedSignInServiceHandler

	mu   sync.Mutex
	seen []string
}

func (s *authorizationRecorder) GetChallenge(_ context.Context, req *connect.Request[syncv1.GetChallengeRequest]) (*connect.Response[syncv1.GetChallengeResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, req.Header().Get("Authorization"))
	return connect.NewResponse(newChallenge()), nil
}

func TestNewBearerInterceptor_SendsTheTokenAsABearerInEveryRequestAndReadsItAgainEachTime(t *testing.T) {
	t.Parallel()
	recorder := &authorizationRecorder{}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewSignInServiceHandler(recorder))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	tokens := []string{"first", "second"}
	next := 0
	client := syncv1connect.NewSignInServiceClient(http.DefaultClient, server.URL, connect.WithInterceptors(syncprotocol.NewBearerInterceptor(func() string {
		token := tokens[next]
		next++
		return token
	})))

	for range tokens {
		if _, err := client.GetChallenge(t.Context(), connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: make([]byte, 32)})); err != nil {
			t.Fatal(err)
		}
	}

	if len(recorder.seen) != 2 || recorder.seen[0] != "Bearer first" || recorder.seen[1] != "Bearer second" {
		t.Errorf("authorization headers = %q, want [Bearer first Bearer second]", recorder.seen)
	}
}

func TestNewBearerInterceptor_RemovesTheBearerOfAReusedRequestWhenTheTokenBecomesEmpty(t *testing.T) {
	t.Parallel()
	tokens := []string{"first", ""}
	next := 0
	interceptor := syncprotocol.NewBearerInterceptor(func() string {
		token := tokens[next]
		next++
		return token
	})
	req := connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: make([]byte, 32)})
	var seen []string
	call := interceptor.WrapUnary(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		seen = append(seen, req.Header().Get("Authorization"))
		return nil, nil
	})

	for range tokens {
		if _, err := call(t.Context(), req); err != nil {
			t.Fatal(err)
		}
	}

	if len(seen) != 2 || seen[0] != "Bearer first" || seen[1] != "" {
		t.Errorf("Authorization seen = %q, want [\"Bearer first\" \"\"]", seen)
	}
}
