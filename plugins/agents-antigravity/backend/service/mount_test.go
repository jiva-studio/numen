package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// The handler a host that links the plugin in serves answers the agent service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheAgentService(t *testing.T) {
	agenttest.CheckLinkedHandler(t, NewHandler)

	withoutAgentsAnywhere(t)
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := agentrun.NewClient(web.Client(), web.URL)
	client.Header.Set("Authorization", "Bearer tok")

	state, err := client.GetState(t.Context())
	if err != nil || state.Unreachable == "" {
		t.Fatalf("GetState without antigravity = (%+v, %v), want unreachable", state, err)
	}
	progs, err := client.GetPrograms(t.Context(), nil)
	if err != nil || len(progs) != 1 || progs[0].IsInstalled {
		t.Fatalf("GetPrograms without antigravity = (%+v, %v), want not installed", progs, err)
	}

	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	lifecycle := server.NewLifecycleClient(http.DefaultClient, web.URL, hostcall.WithBearer("tok"))
	if _, err := lifecycle.Configure(t.Context(), map[string]string{"executablePath": "  " + shPath + "  "}, nil); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	state, err = client.GetState(t.Context())
	if err != nil || state.Unreachable != "" {
		t.Fatalf("GetState after Configure = (%+v, %v), want reachable", state, err)
	}
	if _, err := lifecycle.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Direct server config, customPath GetPrograms, and Ask first-send error.
	s := &agentServer{}
	s.applyConfig(map[string]string{"executablePath": " " + shPath + " "})
	s.applyConfig(map[string]string{"other": "ignored"})
	if got := s.getExecutablePath(); got != shPath {
		t.Fatalf("getExecutablePath = %q, want %q", got, shPath)
	}
	s.getModelCache().Put(shPath, []antigravityModel{{Name: "m-custom", Title: "Custom"}})
	progsCustom, err := s.GetPrograms(t.Context(), nil)
	if err != nil || len(progsCustom) != 1 || !progsCustom[0].IsInstalled || len(progsCustom[0].Models) != 1 {
		t.Fatalf("GetPrograms with customPath = (%+v, %v)", progsCustom, err)
	}
	sendErr := errors.New("send failed")
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(*agentrun.Response) error {
		return sendErr
	}); !errors.Is(err, sendErr) {
		t.Errorf("Ask when first send fails = %v, want %v", err, sendErr)
	}
}
