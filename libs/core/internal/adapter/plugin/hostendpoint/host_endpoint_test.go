package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// callers is a host that resolves the plugins registered with it.
type callers struct {
	mu         sync.Mutex
	plugins    map[string]string
	allowed    map[string]bool
	checked    []string
	vault      pluginport.PluginVault
	modelCalls connectionport.ModelCalls
	texts      pluginport.PluginTexts
	search     pluginport.PluginSearch
	work       pluginport.PluginWork
	tools      PluginTools
	acquired   map[string]int
	releases   chan string
}

func (c *callers) GetPluginIDByToken(token string) (string, bool) {
	id, found := c.plugins[token]
	return id, found && token != ""
}

func (c *callers) CheckPluginPermission(pluginID, permission string) error {
	c.mu.Lock()
	c.checked = append(c.checked, pluginID+" "+permission)
	c.mu.Unlock()
	if !c.allowed[pluginID] {
		return pluginport.ErrPermissionDenied
	}
	return nil
}

func (c *callers) GetModelCalls() connectionport.ModelCalls { return c.modelCalls }

func (c *callers) GetPluginVault() pluginport.PluginVault { return c.vault }

func (c *callers) GetPluginTexts() pluginport.PluginTexts        { return c.texts }
func (c *callers) GetPluginSearch() pluginport.PluginSearch      { return c.search }
func (c *callers) GetDependencies() pluginport.DependencyCatalog { return nil }
func (c *callers) GetPluginWork() pluginport.PluginWork          { return c.work }
func (c *callers) GetPluginTools() PluginTools                   { return c.tools }

func (c *callers) Acquire(pluginID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.acquired == nil {
		c.acquired = map[string]int{}
	}
	c.acquired[pluginID]++
}

func (c *callers) Release(pluginID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acquired[pluginID]--
	if c.releases != nil {
		c.releases <- pluginID
	}
}

func (c *callers) getAcquired(pluginID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.acquired[pluginID]
}

func newCallers() *callers {
	return &callers{
		plugins: map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b", "token-c": "md.numen.c"},
		allowed: map[string]bool{"md.numen.a": true, "md.numen.c": true},
	}
}

func readCode(t *testing.T, err error) connect.Code {
	t.Helper()
	var failure *connect.Error
	if !errors.As(err, &failure) {
		t.Fatalf("error %v is not a status", err)
	}
	return failure.Code()
}

// A header that is no bearer token is not read as one.
func TestAHeaderThatIsNoBearerTokenIsRefused(t *testing.T) {
	fakeCallers := newCallers()
	endpoint := &Endpoint{callers: fakeCallers}
	for _, header := range []string{"", "token-a", "Basic token-a", "bearer token-a"} {
		if _, err := endpoint.authorize(header, manifest.PermissionVaultRead); readCode(t, err) != connect.CodeUnauthenticated {
			t.Errorf("header %q was accepted", header)
		}
	}
}

// The endpoint binds the loopback interface and responds for a token.
func TestHostEndpoint_BindsLoopbackAndServes(t *testing.T) {
	fakeCallers := newCallers()
	fakeCallers.modelCalls = &scriptedCalls{complete: connectionport.CompleteResult{Text: "ok"}}
	endpoint, err := Open(t.Context(), Address, fakeCallers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	if endpoint.GetPort() <= 0 {
		t.Fatalf("port = %d", endpoint.GetPort())
	}

	client := numenv1connect.NewHostModelServiceClient(http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	request := connect.NewRequest(&v1.HostModelServiceCompleteRequest{Messages: []*v1.HostModelMessage{{Role: "user", Text: "u"}}})
	res, err := client.Complete(t.Context(), request)
	if err != nil || res.Msg.GetText() != "ok" {
		t.Fatalf("answered %v, %v", res, err)
	}

	if err := endpoint.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(t.Context(), request); err == nil {
		t.Error("a closed endpoint answered")
	}
}
