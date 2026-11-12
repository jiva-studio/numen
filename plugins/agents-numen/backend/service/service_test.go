package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
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

	res, err := server.NewLifecycleClient(http.DefaultClient, pos, hostcall.WithBearer(testToken)).Ping(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if res.Protocol != server.ProtocolVersion {
		t.Errorf("the protocol is %d", res.Protocol)
	}
}

// callTask asks the handler for acp.task over Connect and returns the body of the answer.
func callTask(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	payload, err := proto.Marshal(&acp.TaskRequest{Prompt: "When does the ferry leave?"})
	if err != nil {
		t.Fatal(err)
	}
	message, err := json.Marshal(map[string]any{"operation": acp.OperationTask, "call": map[string]any{"type": "numen"}, "payload": payload})
	if err != nil {
		t.Fatal(err)
	}
	envelope := binary.BigEndian.AppendUint32([]byte{0}, uint32(len(message)))
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/numen.v1.ProviderService/Call", bytes.NewReader(append(envelope, message...)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/connect+json")
	request.Header.Set("Authorization", "Bearer "+testToken)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("the plugin answered %d %q", response.StatusCode, body)
	}
	return string(body)
}

// A task while the settings hold no row of llm.chat stops as unavailable, before the host is reached.
func TestThePluginServesAgentTasksAsAProvider(t *testing.T) {
	pos := startPlugin(t)

	body := callTask(t, http.DefaultClient, pos)

	if !strings.Contains(body, "FAILURE_KIND_STOP") || !strings.Contains(body, "FAILURE_REASON_UNAVAILABLE") {
		t.Errorf("the plugin answered %q", body)
	}
}

func TestTheLinkedHandlerServesAgentTasksAsAProvider(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken, "NUMEN_HOST_PORT": "4000"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	body := callTask(t, web.Client(), web.URL)

	if !strings.Contains(body, "FAILURE_KIND_STOP") {
		t.Errorf("the plugin answered %q", body)
	}
}

func TestNewHandlerNeedsTheHostOfTheCore(t *testing.T) {
	if _, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": testToken}); err == nil {
		t.Error("NewHandler accepted variables with no host port")
	}
}
