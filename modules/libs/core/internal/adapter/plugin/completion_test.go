package plugin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// answeringAgent is an agent plugin answering with whatever it was told to.
type answeringAgent struct {
	numenv1connect.UnimplementedCompletionServiceHandler
	got      *v1.CompleteRequest
	response *v1.CompleteResponse
	failure  error
}

func (a *answeringAgent) Complete(
	_ context.Context, req *connect.Request[v1.CompleteRequest],
) (*connect.Response[v1.CompleteResponse], error) {
	a.got = req.Msg
	if a.failure != nil {
		return nil, a.failure
	}
	return connect.NewResponse(a.response), nil
}

// completionHost serves one agent plugin and resolves ids to it.
type completionHost struct {
	endpoint  port.Endpoint
	plugins   map[string]port.PluginManifest
	byCap     string
	ensureErr error
	opened    []string
	released  int
}

func newCompletionHost(t *testing.T, agent *answeringAgent) *completionHost {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewCompletionServiceHandler(agent))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &completionHost{
		endpoint: port.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port},
		plugins: map[string]port.PluginManifest{
			"md.numen.agent": {ID: "md.numen.agent", Capabilities: []string{CapabilityAgentProvider}},
			"md.numen.other": {ID: "md.numen.other", Capabilities: []string{"proofreading"}},
		},
		byCap: "md.numen.agent",
	}
}

func (h *completionHost) FindPluginByCapability(capability string) (port.DiscoveredPlugin, bool) {
	if capability != CapabilityAgentProvider || h.byCap == "" {
		return port.DiscoveredPlugin{}, false
	}
	return port.DiscoveredPlugin{Manifest: h.plugins[h.byCap]}, true
}

func (h *completionHost) FindPluginByFileExtension(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{}, false
}

func (h *completionHost) GetPlugin(id string) (port.DiscoveredPlugin, bool) {
	manifest, found := h.plugins[id]
	return port.DiscoveredPlugin{Manifest: manifest}, found
}

func (h *completionHost) EnsureActive(_ context.Context, id string) (port.Endpoint, error) {
	if h.ensureErr != nil {
		return port.Endpoint{}, h.ensureErr
	}
	h.opened = append(h.opened, id)
	return h.endpoint, nil
}

func (h *completionHost) Release(string) { h.released++ }

func answerWith(text string) *v1.CompleteResponse {
	return &v1.CompleteResponse{Answer: &v1.CompletionAnswer{Text: text}}
}

// The agent is told the question and the model of the route, and its text comes back.
func TestTheAgentIsToldTheQuestionAndTheModel(t *testing.T) {
	agent := &answeringAgent{response: answerWith("fixed")}
	host := newCompletionHost(t, agent)
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "md.numen.agent")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	temperature := 0.5
	got, err := opened.Complete(t.Context(), "model-x", port.Question{
		Purpose:  "proofreading",
		Messages: []port.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: "hi"}},
		Hints:    port.Hints{CostPriority: 1, MaxTokens: 10, Temperature: &temperature},
	})
	if err != nil || got != "fixed" {
		t.Fatalf("answered %q, %v", got, err)
	}
	asked := agent.got
	if asked.GetModel() != "model-x" || asked.GetQuestion().GetPurpose() != "proofreading" ||
		len(asked.GetQuestion().GetMessages()) != 2 || asked.GetQuestion().GetMessages()[0].GetRole() != "system" ||
		asked.GetQuestion().GetHints().GetCostPriority() != 1 || asked.GetQuestion().GetHints().GetMaxTokens() != 10 ||
		asked.GetQuestion().GetHints().GetTemperature() != temperature {
		t.Errorf("the agent was asked %v", asked)
	}
}

// A question without a temperature reaches the agent without one.
func TestAnUnsetTemperatureStaysUnset(t *testing.T) {
	agent := &answeringAgent{response: answerWith("x")}
	host := newCompletionHost(t, agent)
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	if _, err := opened.Complete(t.Context(), "", port.Question{}); err != nil {
		t.Fatal(err)
	}
	if agent.got.GetQuestion().Hints.Temperature != nil {
		t.Error("a temperature nobody set reached the agent")
	}
}

// A refusal the agent answers with is the core's own error.
func TestAnAgentsRefusalIsTheCoresError(t *testing.T) {
	for _, one := range []struct {
		name string
		code v1.CompletionErrorCode
		want error
	}{
		{"no route", v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, port.ErrNoRoute},
		{"unavailable", v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, port.ErrUnavailable},
	} {
		t.Run(one.name, func(t *testing.T) {
			host := newCompletionHost(t, &answeringAgent{response: &v1.CompleteResponse{Error: one.code}})
			opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = opened.Close() })
			if _, err := opened.Complete(t.Context(), "", port.Question{}); !errors.Is(err, one.want) {
				t.Errorf("error = %v, want %v", err, one.want)
			}
		})
	}
}

// An answer that is not there, and a refusal of a kind this host does not know, are errors.
func TestAnAnswerThatIsNotThereIsAnError(t *testing.T) {
	for _, response := range []*v1.CompleteResponse{{}, {Error: v1.CompletionErrorCode(99)}} {
		host := newCompletionHost(t, &answeringAgent{response: response})
		opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		if text, err := opened.Complete(t.Context(), "", port.Question{}); err == nil {
			t.Errorf("answered %q for %v", text, response)
		}
		_ = opened.Close()
	}
}

// An agent that does not answer Complete, or is down, is unavailable.
func TestAnAgentThatCannotAnswerIsUnavailable(t *testing.T) {
	for _, one := range []struct {
		name    string
		failure error
	}{
		{"unimplemented", connect.NewError(connect.CodeUnimplemented, errors.New("no"))},
		{"unavailable", connect.NewError(connect.CodeUnavailable, errors.New("down"))},
	} {
		t.Run(one.name, func(t *testing.T) {
			host := newCompletionHost(t, &answeringAgent{failure: one.failure})
			opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = opened.Close() })
			if _, err := opened.Complete(t.Context(), "", port.Question{}); !errors.Is(err, port.ErrUnavailable) {
				t.Errorf("error = %v, want port.ErrUnavailable", err)
			}
		})
	}
}

// An agent that cannot be reached at all is unavailable.
func TestAnUnreachableAgentIsUnavailable(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	host.endpoint = port.Endpoint{Host: "127.0.0.1", Port: 1}
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	if _, err := opened.Complete(t.Context(), "", port.Question{}); !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v, want port.ErrUnavailable", err)
	}
}

// Any other failure of the agent is kept, and is not an unavailability.
func TestAnotherFailureOfTheAgentIsKept(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{failure: connect.NewError(connect.CodeInvalidArgument, errors.New("bad"))})
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	_, err = opened.Complete(t.Context(), "", port.Question{})
	if err == nil || errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v", err)
	}
}

// A cancelled context is the caller's own error, not an unavailability.
func TestACancelledCallIsNotAnUnavailability(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = opened.Complete(ctx, "", port.Question{})
	if !errors.Is(err, context.Canceled) || errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v, want context.Canceled alone", err)
	}
}

// The agent is the one the route names, or the one the host finds by capability.
func TestTheAgentIsFoundByIDOrByCapability(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	agents := NewCompletionAgents(host)

	for _, id := range []string{"", "md.numen.agent"} {
		opened, err := agents.OpenCompletionAgent(t.Context(), id)
		if err != nil {
			t.Fatalf("id %q: %v", id, err)
		}
		_ = opened.Close()
	}
	if len(host.opened) != 2 || host.opened[0] != "md.numen.agent" || host.opened[1] != "md.numen.agent" {
		t.Errorf("activated %v", host.opened)
	}
}

// A plugin that is absent, or is no agent, or whose capability no plugin
// provides, is unavailable and is not activated.
func TestAnAbsentAgentIsUnavailable(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	agents := NewCompletionAgents(host)

	for _, id := range []string{"md.numen.missing", "md.numen.other"} {
		if _, err := agents.OpenCompletionAgent(t.Context(), id); !errors.Is(err, port.ErrUnavailable) {
			t.Errorf("id %q: error = %v, want port.ErrUnavailable", id, err)
		}
	}
	host.byCap = ""
	if _, err := agents.OpenCompletionAgent(t.Context(), ""); !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("no provider: error = %v, want port.ErrUnavailable", err)
	}
	if len(host.opened) != 0 {
		t.Errorf("activated %v", host.opened)
	}
}

// A plugin that fails to start is unavailable; a cancelled start is not.
func TestAnAgentThatFailsToStartIsUnavailable(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	host.ensureErr = errors.New("crashed")
	agents := NewCompletionAgents(host)
	if _, err := agents.OpenCompletionAgent(t.Context(), ""); !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v, want port.ErrUnavailable", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := agents.OpenCompletionAgent(ctx, ""); !errors.Is(err, context.Canceled) || errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v, want context.Canceled alone", err)
	}
}

// Closing the agent gives its reference back once.
func TestTheAgentGivesItsReferenceBackOnce(t *testing.T) {
	host := newCompletionHost(t, &answeringAgent{response: answerWith("x")})
	opened, err := NewCompletionAgents(host).OpenCompletionAgent(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if host.released != 1 {
		t.Errorf("gave back %d references, want 1", host.released)
	}
}
