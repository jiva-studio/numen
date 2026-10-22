package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const testToken = "plugin-token"

// startPlugin serves the plugin on a free port and returns where it answers.
func startPlugin(t *testing.T) string {
	t.Helper()
	stdin, hold := io.Pipe()
	t.Cleanup(func() { _ = hold.Close() })
	s := server.New(server.WithPort(0), server.WithToken(testToken), server.WithStdin(stdin), server.WithStdout(io.Discard), server.WithSignals())
	if err := Mount(s); err != nil {
		t.Fatal(err)
	}
	port, err := s.Listen()
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	isDone := make(chan error, 1)
	go func() { isDone <- s.ServeContext(ctx) }()
	t.Cleanup(func() { stop(); <-isDone })
	return "http://127.0.0.1:" + strconv.Itoa(port)
}

func withToken() connect.ClientOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+testToken)
			return next(ctx, req)
		}
	}))
}

func TestThePluginAnswersPingWithTheProtocolVersion(t *testing.T) {
	at := startPlugin(t)

	res, err := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, at, withToken()).
		Ping(t.Context(), connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetProtocol() != server.ProtocolVersion {
		t.Errorf("the protocol is %d", res.Msg.GetProtocol())
	}
}

func TestConfigure_PropagatesSettingsToAgent(t *testing.T) {
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer host-key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()
	at := startPlugin(t)

	if _, err := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, at, withToken()).
		Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{Values: map[string]string{"baseUrl": chat.URL, "apiKey": "host-key"}})); err != nil {
		t.Fatal(err)
	}
	res, err := numenv1connect.NewAgentServiceClient(http.DefaultClient, at, withToken()).
		GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := res.Msg.GetPrograms(); len(programs) != 1 || !programs[0].GetIsInstalled() || len(programs[0].GetModels()) != 1 {
		t.Errorf("the programs are %+v", programs)
	}
}

// The handler a host that links the plugin in serves hands Configure's base URL and key to the agent.
func TestNewHandler_PropagatesSettingsToAgent(t *testing.T) {
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer host-key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if _, err := numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL, withToken()).
		Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{Values: map[string]string{"baseUrl": chat.URL, "apiKey": "host-key"}})); err != nil {
		t.Fatal(err)
	}
	res, err := numenv1connect.NewAgentServiceClient(web.Client(), web.URL, withToken()).
		GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := res.Msg.GetPrograms(); len(programs) != 1 || len(programs[0].GetModels()) != 1 {
		t.Errorf("the programs are %+v", programs)
	}
}
