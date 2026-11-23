package syncprotocol_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type recordingService struct {
	syncv1connect.UnimplementedSignInServiceHandler

	mu      sync.Mutex
	headers []string
}

func (s *recordingService) GetChallenge(_ context.Context, req *connect.Request[syncv1.GetChallengeRequest]) (*connect.Response[syncv1.GetChallengeResponse], error) {
	s.mu.Lock()
	s.headers = append(s.headers, req.Header().Get(syncprotocol.HeaderVersion))
	s.mu.Unlock()
	return connect.NewResponse(newChallenge()), nil
}

func (s *recordingService) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.headers...)
}

func serve(t *testing.T, serverVersion uint32) (*recordingService, string) {
	t.Helper()
	service := &recordingService{}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewSignInServiceHandler(service, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(serverVersion))))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return service, server.URL
}

func ask(url string, options ...connect.ClientOption) error {
	client := syncv1connect.NewSignInServiceClient(http.DefaultClient, url, options...)
	_, err := client.GetChallenge(context.Background(), connect.NewRequest(&syncv1.GetChallengeRequest{DeviceSignKey: make([]byte, 32)}))
	return err
}

func TestAClientSendsTheProtocolVersionOfItsPackageInTheHeaderOfEveryRequest(t *testing.T) {
	service, url := serve(t, syncprotocol.Version)

	if err := ask(url, connect.WithInterceptors(syncprotocol.NewVersionInterceptor())); err != nil {
		t.Fatal(err)
	}

	if got := service.calls(); len(got) != 1 || got[0] != "1" {
		t.Errorf("headers the service saw = %q, want [1]", got)
	}
}

func TestAServerOfAnotherVersionAnswersFailedPreconditionAndNeverRunsTheCall(t *testing.T) {
	service, url := serve(t, syncprotocol.Version+1)

	err := ask(url, connect.WithInterceptors(syncprotocol.NewVersionInterceptor()))

	if !errors.Is(err, syncprotocol.ErrUpdateRequired) {
		t.Fatalf("error = %v, want ErrUpdateRequired", err)
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want failed precondition", connect.CodeOf(err))
	}
	if got := service.calls(); len(got) != 0 {
		t.Errorf("the service ran %d calls, want none", len(got))
	}
}

func TestTheMismatchCarriesTheVersionOfTheServerInAnUpdateRequiredDetail(t *testing.T) {
	_, url := serve(t, 7)

	err := ask(url)

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition {
		t.Fatalf("error = %v, want a failed precondition", err)
	}
	var required *syncv1.UpdateRequired
	for _, detail := range failure.Details() {
		if value, valueErr := detail.Value(); valueErr == nil {
			if one, isRequired := value.(*syncv1.UpdateRequired); isRequired {
				required = one
			}
		}
	}
	if required == nil || required.GetServerVersion() != 7 {
		t.Errorf("detail = %v, want UpdateRequired with server version 7", required)
	}
}

func TestARequestWithoutTheHeaderOrWithAHeaderThatIsNoNumberIsAMismatch(t *testing.T) {
	service, url := serve(t, syncprotocol.Version)
	notANumber := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set(syncprotocol.HeaderVersion, "one")
			return next(ctx, req)
		}
	}))

	for name, option := range map[string]connect.ClientOption{"no header": connect.WithInterceptors(), "not a number": notANumber} {
		err := ask(url, option)
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("%s: error = %v, want failed precondition", name, err)
		}
	}
	if got := service.calls(); len(got) != 0 {
		t.Errorf("the service ran %d calls, want none", len(got))
	}
}

func TestAnErrorThatIsNotAMismatchIsLeftAsTheServerSentIt(t *testing.T) {
	_, url := serve(t, syncprotocol.Version)
	client := syncv1connect.NewSignInServiceClient(http.DefaultClient, url, connect.WithInterceptors(syncprotocol.NewVersionInterceptor()))

	_, err := client.SignIn(t.Context(), connect.NewRequest(&syncv1.SignInRequest{}))

	if errors.Is(err, syncprotocol.ErrUpdateRequired) || connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("error = %v, want the unimplemented of the service", err)
	}
}
