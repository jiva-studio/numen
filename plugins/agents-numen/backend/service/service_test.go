package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp/acptest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

const testToken = "plugin-token"

func newEnvHandler(t *testing.T, vars map[string]string) (http.Handler, error) {
	t.Helper()
	t.Setenv(hostcall.HostPortEnv, "")
	for name, value := range vars {
		t.Setenv(name, value)
	}
	return servertest.NewHandler(vars, Mount)
}

func startPlugin(t *testing.T) string {
	t.Helper()
	t.Setenv(hostcall.HostPortEnv, "4000")
	t.Setenv(hostcall.PluginTokenEnv, testToken)
	handler, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: testToken}, Mount)
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web.URL
}

func TestThePluginAnswersPingWithTheProtocolVersion(t *testing.T) {
	pos := startPlugin(t)

	protocol, err := servertest.NewLifecycleClient(http.DefaultClient, pos, testToken).Ping(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if protocol != server.ProtocolVersion {
		t.Errorf("the protocol is %d", protocol)
	}
}

func callTask(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	return callTaskWithOptions(t, client, baseURL, nil)
}

func callTaskWithOptions(t *testing.T, client *http.Client, baseURL string, options map[string]string) string {
	t.Helper()
	payload, err := acptest.EncodeTaskRequest(&acp.TaskRequest{Prompt: "When does the ferry leave?"})
	if err != nil {
		t.Fatal(err)
	}
	call := map[string]any{"type": "numen"}
	if options != nil {
		call["options"] = options
	}
	message, err := json.Marshal(map[string]any{"operation": acp.OperationTask, "call": call, "payload": payload})
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

// A task whose row names no model stops as an invalid request, before the host is reached.
func TestThePluginServesAgentTasksAsAProvider(t *testing.T) {
	pos := startPlugin(t)

	body := callTask(t, http.DefaultClient, pos)

	if !strings.Contains(body, "FAILURE_KIND_STOP") || !strings.Contains(body, "FAILURE_REASON_INVALID_REQUEST") {
		t.Errorf("the plugin answered %q", body)
	}
}

func TestTheLinkedHandlerServesAgentTasksAsAProvider(t *testing.T) {
	handler, err := newEnvHandler(t, map[string]string{"NUMEN_PLUGIN_TOKEN": testToken, "NUMEN_HOST_PORT": "4000"})
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

func TestMountNeedsTheHostOfTheCore(t *testing.T) {
	t.Setenv(hostcall.HostPortEnv, "")
	if _, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: testToken}, Mount); err == nil {
		t.Error("Mount accepted an environment with no host port")
	}
}

func TestMountRejectsMissingHostParts(t *testing.T) {
	if _, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: testToken}, func(s *server.Server) error {
		return mount(s, host{})
	}); err == nil {
		t.Error("mount accepted an empty host")
	}
}
