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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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
	pos := startPlugin(t)

	res, err := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, pos, withToken()).
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
	pos := startPlugin(t)

	if _, err := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, pos, withToken()).
		Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{Values: map[string]string{"baseUrl": chat.URL, "apiKey": "host-key"}})); err != nil {
		t.Fatal(err)
	}
	res, err := numenv1connect.NewAgentServiceClient(http.DefaultClient, pos, withToken()).
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

func TestConfigure_SetsTemperatureAndMaxTokens(t *testing.T) {
	var receivedBody map[string]any
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "hi"}},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()

	s := &Server{Client: chat.Client()}
	s.Configure(map[string]string{
		"baseUrl":     chat.URL,
		"temperature": "0.5",
		"max_tokens":  "2048",
	})
	c := s.getChat()
	_, _, err := c.send(t.Context(), "m1", effort{}, []message{{Role: "user", Content: newText("hello")}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if temp, ok := receivedBody["temperature"].(float64); !ok || temp != 0.5 {
		t.Errorf("expected temperature 0.5, got %v", receivedBody["temperature"])
	}
	if maxT, ok := receivedBody["max_tokens"].(float64); !ok || int(maxT) != 2048 {
		t.Errorf("expected max_tokens 2048, got %v", receivedBody["max_tokens"])
	}
}
