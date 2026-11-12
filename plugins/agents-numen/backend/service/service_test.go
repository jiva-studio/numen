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
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

const testToken = "plugin-token"

func startPlugin(t *testing.T) string {
	t.Helper()
	t.Setenv(hostcall.HostPortEnv, "4000")
	t.Setenv(hostcall.PluginTokenEnv, testToken)
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

	res, err := servertest.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).Ping(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if res.Protocol != server.ProtocolVersion {
		t.Errorf("the protocol is %d", res.Protocol)
	}
}

func TestConfigurePropagatesTheRowToTheAgent(t *testing.T) {
	pos := startPlugin(t)

	if _, err := servertest.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).
		Configure(t.Context(), map[string]string{"connections": rowsJSON(row{connection: "c1", model: "m1"})}); err != nil {
		t.Fatal(err)
	}
	res, err := agentrun.NewClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if programs := res; len(programs) != 1 || !programs[0].IsInstalled || len(programs[0].Models) != 1 || programs[0].Models[0].Name != "m1" {
		t.Errorf("the programs are %+v", programs)
	}
}

// The handler a host that links the plugin in serves hands the rows Configure receives to the agent.
func TestNewHandlerPropagatesTheRowToTheAgent(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken, "NUMEN_HOST_PORT": "4000"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if _, err := servertest.NewLifecycleClient(web.Client(), web.URL, hostcall.WithBearer(testToken)).
		Configure(t.Context(), map[string]string{"connections": rowsJSON(row{connection: "c1", model: "m1"})}); err != nil {
		t.Fatal(err)
	}
	res, err := agentrun.NewClient(web.Client(), web.URL, hostcall.WithBearer(testToken)).GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if programs := res; len(programs) != 1 || !programs[0].IsInstalled {
		t.Errorf("the programs are %+v", programs)
	}
}

func TestNewHandlerNeedsTheHostOfTheCore(t *testing.T) {
	if _, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken}); err == nil {
		t.Error("NewHandler accepted variables with no host port")
	}
}
