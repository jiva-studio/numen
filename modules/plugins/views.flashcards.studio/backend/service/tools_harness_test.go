package service_test

import (
	"encoding/json"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service"
)

// session is the tools the plugin offers, called the way the host calls them.
type session struct {
	registry *tools.Registry
	// asked is what the host tells each tool about the window the agent was started from.
	asked map[string]string
}

// newSession is the tools over a vault of the notes given.
func newSession(t *testing.T, notes map[string]string) (*session, domain.Vault) {
	t.Helper()
	env := vaulttest.New(t, notes)
	registry, err := service.New(service.Config{Vault: env.Host}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	return &session{registry: registry}, env.Vault
}

// invoke calls one tool and returns what it answered.
func (s *session) invoke(t *testing.T, name string, args any) *v1.CallToolResponse {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.registry.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: name, Arguments: raw, Context: s.asked,
	}))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.Msg.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("%s: rejected as %v", name, res.Msg.GetError())
	}
	return res.Msg
}

// list is the tools the plugin offers.
func (s *session) list(t *testing.T) []*v1.PluginTool {
	t.Helper()
	res, err := s.registry.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg.GetTools()
}

// call is a tool called and its answer read as T.
func call[T any](t *testing.T, s *session, name string, args any) T {
	t.Helper()
	res := s.invoke(t, name, args).GetResult()
	if res.GetIsRefusal() {
		t.Fatalf("%s: %s", name, res.GetText())
	}
	var out T
	if err := json.Unmarshal(res.GetStructured(), &out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

func getToolError(t *testing.T, s *session, name string, args any) string {
	t.Helper()
	res := s.invoke(t, name, args).GetResult()
	if !res.GetIsRefusal() {
		t.Fatalf("%s: expected an error, got %s", name, res.GetStructured())
	}
	return res.GetText()
}

// deckFingerprint is what a deck is at this moment, as card_read gives it.
func deckFingerprint(t *testing.T, s *session, path string) string {
	t.Helper()
	return call[struct {
		Fingerprint string `json:"fingerprint"`
	}](t, s, "card_read", map[string]any{"path": path}).Fingerprint
}
