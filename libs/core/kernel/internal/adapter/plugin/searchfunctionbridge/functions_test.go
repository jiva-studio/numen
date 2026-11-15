package searchfunctionbridge_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/searchfunctionbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

const shelfPlugin = "md.test.shelf"

// shelfService is a plugin that answers shelved(shelf) for the paths on each shelf.
type shelfService struct {
	numenv1connect.UnimplementedSearchFunctionServiceHandler
	shelves map[string][]string
	calls   chan *v1.SearchFunctionCall
	block   chan struct{}
}

func (s *shelfService) MatchDocuments(ctx context.Context, req *connect.Request[v1.MatchDocumentsRequest]) (*connect.Response[v1.MatchDocumentsResponse], error) {
	s.calls <- req.Msg.GetCall()
	shelf := req.Msg.GetCall().GetArguments()[0].GetStringValue()
	if shelf == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("shelved needs a shelf"))
	}
	if s.block != nil {
		select {
		case <-s.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	held := s.shelves[shelf]
	paths := slices.DeleteFunc(slices.Clone(req.Msg.GetPaths()), func(p string) bool { return !slices.Contains(held, p) })
	return connect.NewResponse(&v1.MatchDocumentsResponse{Paths: paths, Warning: "late"}), nil
}

func (s *shelfService) FindDocuments(_ context.Context, req *connect.Request[v1.FindDocumentsRequest]) (*connect.Response[v1.FindDocumentsResponse], error) {
	s.calls <- req.Msg.GetCall()
	held := s.shelves[req.Msg.GetCall().GetArguments()[0].GetStringValue()]
	return connect.NewResponse(&v1.FindDocumentsResponse{Paths: held[:min(int(req.Msg.GetLimit()), len(held))], EstimatedCount: int64(len(held))}), nil
}

// activator serves one plugin from a test server and counts the use leases taken and released.
type activator struct {
	port     int
	mu       sync.Mutex
	taken    int
	released int
	fail     error
}

func (a *activator) EnsureActive(_ context.Context, pluginID string, _ plugin.Holder) (transport.Endpoint, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.fail != nil {
		return transport.Endpoint{}, a.fail
	}
	a.taken++
	return transport.Endpoint{Host: "127.0.0.1", Port: a.port, Token: "t-" + pluginID}, nil
}

func (a *activator) Release(string, string, plugin.Holder) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.released++
}

func (a *activator) EnsureActiveForQuestion(context.Context, string, plugin.Holder) (transport.Endpoint, error) {
	return transport.Endpoint{}, errors.New("a search function takes a use lease")
}

func (a *activator) ReleaseQuestion(string, string, plugin.Holder) {}

func serve(t *testing.T, service *shelfService) *activator {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewSearchFunctionServiceHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	addr, isTCP := server.Listener.Addr().(*net.TCPAddr)
	if !isTCP {
		t.Fatalf("expected *net.TCPAddr, got %T", server.Listener.Addr())
	}
	return &activator{port: addr.Port}
}

func newShelves() *shelfService {
	return &shelfService{shelves: map[string][]string{"east": {"a.md", "b.md", "c.md"}}, calls: make(chan *v1.SearchFunctionCall, 8)}
}

func callShelved(shelf string) searchquery.FunctionCall {
	return searchquery.FunctionCall{
		PluginID: shelfPlugin, Function: "shelved", VaultID: "v1", IndexGeneration: "g3",
		Arguments: []searchquery.Argument{{Name: "shelf", Type: searchquery.TypeString, String: shelf}},
	}
}

func TestACallCarriesTheFunctionItsArgumentsTheVaultAndTheGenerationUnderAUseLease(t *testing.T) {
	service := newShelves()
	host := serve(t, service)

	answer, err := searchfunctionbridge.Open(host).MatchDocuments(t.Context(), callShelved("east"), []string{"a.md", "z.md"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(answer.Paths, []string{"a.md"}) || answer.Warning != "late" {
		t.Errorf("answer = %+v, want a.md with the warning", answer)
	}
	call := <-service.calls
	if call.GetFunction() != "shelved" || call.GetVaultId() != "v1" || call.GetIndexGeneration() != "g3" ||
		call.GetArguments()[0].GetName() != "shelf" || call.GetArguments()[0].GetStringValue() != "east" {
		t.Errorf("the plugin received %v", call)
	}
	if host.taken != 1 || host.released != 1 {
		t.Errorf("leases taken %d, released %d, want one of each", host.taken, host.released)
	}
}

func TestFindReturnsAtMostTheLimitAndTheEstimatedCount(t *testing.T) {
	host := serve(t, newShelves())

	answer, err := searchfunctionbridge.Open(host).FindDocuments(t.Context(), callShelved("east"), 2)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(answer.Paths, []string{"a.md", "b.md"}) || answer.EstimatedCount != 3 {
		t.Errorf("answer = %+v, want two paths of three", answer)
	}
}

func TestInvalidArgumentFromThePluginIsInvalidFunctionArguments(t *testing.T) {
	host := serve(t, newShelves())

	_, err := searchfunctionbridge.Open(host).MatchDocuments(t.Context(), callShelved(""), []string{"a.md"})

	if !errors.Is(err, searchquery.ErrInvalidFunctionArguments) {
		t.Errorf("err = %v, want ErrInvalidFunctionArguments", err)
	}
}

func TestAPluginThatCannotStartIsAnErrorNamingIt(t *testing.T) {
	host := serve(t, newShelves())
	host.fail = errors.New("the process exited")

	_, err := searchfunctionbridge.Open(host).FindDocuments(t.Context(), callShelved("east"), 10)

	var unavailable *plugin.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.PluginID != shelfPlugin {
		t.Errorf("err = %v, want the plugin unavailable", err)
	}
}

func TestCancelingTheSearchCancelsAnOpenCall(t *testing.T) {
	service := newShelves()
	service.block = make(chan struct{})
	host := serve(t, service)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		_, err := searchfunctionbridge.Open(host).MatchDocuments(ctx, callShelved("east"), []string{"a.md"})
		done <- err
	}()
	<-service.calls
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the call did not end after the cancel")
	}
}
