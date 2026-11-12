package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
)

// The handler a host that links the plugin in serves answers the agent service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheAgentService(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := agentrun.NewClient(web.Client(), web.URL)

	client.Header.Set("Authorization", "Bearer linked-token")
	if _, err := client.GetPrograms(t.Context(), nil); err != nil {
		t.Fatalf("call with the token: %v", err)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+"/numen.v1.PluginLifecycleService/Configure", strings.NewReader(`{"values":{"executablePath":"/usr/bin/my-acp"}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer linked-token")
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if closeErr := res.Body.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Configure = %d", res.StatusCode)
	}
	programs, err := client.GetPrograms(t.Context(), nil)
	if err != nil || len(programs) != 1 || programs[0].Name != "acp:acp" {
		t.Fatalf("GetPrograms after Configure = %v (%v)", programs, err)
	}

	client.Header.Del("Authorization")
	_, err = client.GetPrograms(t.Context(), nil)
	if rpcerror.GetCode(err) != rpcerror.Unauthenticated {
		t.Fatalf("call without the token = %v, want unauthenticated", err)
	}
}
