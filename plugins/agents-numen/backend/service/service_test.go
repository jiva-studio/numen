package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const testToken = "plugin-token"

func startPlugin(t *testing.T) string {
	t.Helper()
	stdin, hold := io.Pipe()
	cleanup.CloseAfter(t, hold)
	s := server.New(server.WithPort(0), server.WithToken(testToken), server.WithStdin(stdin), server.WithStdout(io.Discard), server.WithSignals())
	if err := Mount(s); err != nil {
		t.Fatal(err)
	}
	port, err := s.Listen()
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(t.Context())
	isDone := make(chan error, 1)
	go func() { isDone <- s.ServeContext(ctx) }()
	t.Cleanup(func() {
		stop()
		if err := <-isDone; err != nil {
			t.Error(err)
		}
	})
	return "http://127.0.0.1:" + strconv.Itoa(port)
}

func TestThePluginAnswersPingWithTheProtocolVersion(t *testing.T) {
	pos := startPlugin(t)

	res, err := server.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).Ping(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if res.Protocol != server.ProtocolVersion {
		t.Errorf("the protocol is %d", res.Protocol)
	}
}

func TestConfigure_PropagatesSettingsToAgent(t *testing.T) {
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer host-key" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		encodeJSON(t, w, map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()
	pos := startPlugin(t)

	if _, err := server.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).
		Configure(t.Context(), map[string]string{"baseUrl": chat.URL}, hostKey()); err != nil {
		t.Fatal(err)
	}
	res, err := agentrun.NewClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).
		GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if programs := res; len(programs) != 1 || !programs[0].IsInstalled || len(programs[0].Models) != 2 {
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
		encodeJSON(t, w, map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if _, err := server.NewLifecycleClient(web.Client(), web.URL, hostcall.WithBearer(testToken)).
		Configure(t.Context(), map[string]string{"baseUrl": chat.URL}, hostKey()); err != nil {
		t.Fatal(err)
	}
	res, err := agentrun.NewClient(web.Client(), web.URL, hostcall.WithBearer(testToken)).
		GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if programs := res; len(programs) != 1 || len(programs[0].Models) != 2 {
		t.Errorf("the programs are %+v", programs)
	}
}

func hostKey() server.Secrets {
	return server.Secrets{server.SecretAPIKey: {
		{ID: "s00000001", Name: "Work", Group: "openrouter", Value: "host-key"},
		{ID: "s00000002", Name: "Home", Group: "openrouter", Value: "other-key"},
	}}
}

// The agent sends the first secret of the list and ignores the others; an empty list sends no key.
func TestTheFirstSecretOfTheListIsTheKeySentToTheChatService(t *testing.T) {
	var seen []string
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		encodeJSON(t, w, map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer chat.Close()
	pos := startPlugin(t)
	client := server.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken))

	if _, err := client.Configure(t.Context(), map[string]string{"baseUrl": chat.URL}, hostKey()); err != nil {
		t.Fatal(err)
	}
	if _, err := agentrun.NewClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).GetPrograms(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if len(seen) == 0 || seen[len(seen)-1] != "Bearer host-key" {
		t.Errorf("the chat service was asked with %v, want the first secret", seen)
	}

	if _, err := client.Configure(t.Context(), map[string]string{"baseUrl": chat.URL}, server.Secrets{server.SecretAPIKey: {}}); err != nil {
		t.Fatal(err)
	}
	if _, err := agentrun.NewClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).GetPrograms(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if last := seen[len(seen)-1]; last == "Bearer host-key" || last == "Bearer other-key" {
		t.Errorf("the chat service was asked with %q after an empty list", last)
	}
}
