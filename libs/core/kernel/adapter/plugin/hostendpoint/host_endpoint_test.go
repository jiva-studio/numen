package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// callers is a host that resolves the plugins registered with it and
// with the calls on models it was given.
type callers struct {
	mu         sync.Mutex
	plugins    map[string]string
	allowed    map[string]bool
	modelCalls ModelCalls
	checked    []string
	vault      *vaultapi.Access
	texts      TextVersions
	search     *SearchAccess
	work       WorkQueue
	window     WindowControl
	list       vaultapi.List
	granted    []string
	confined   bool
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
		return plugin.ErrPermissionDenied
	}
	if c.confined && !slices.Contains(c.granted, permission) {
		return plugin.ErrPermissionDenied
	}
	return nil
}

func (c *callers) over(list vaultapi.List, granted ...string) Callers {
	c.list, c.granted, c.confined = list, granted, true
	return c.asCallers()
}

func (c *callers) asCallers() Callers {
	lent := NewServices()
	lent.SetModelCalls(c.modelCalls)
	lent.SetPluginVault(c.vault)
	lent.SetPluginTexts(c.texts)
	lent.SetPluginSearch(c.search)
	lent.SetPluginWork(c.work)
	lent.SetPluginWindow(c.window)
	lent.SetPluginVaultList(c.list)
	return Callers{
		Identity: c, Leases: c, Lent: lent,
		Tools: func() PluginTools { return c.tools },
	}
}

func capability[T any](fake any) T {
	found, isFound := fake.(T)
	if !isFound {
		var none T
		return none
	}
	return found
}

func vaultOf(fake any) *vaultapi.Access {
	return &vaultapi.Access{
		Notes: capability[vaultapi.NoteLookup](fake), Links: capability[vaultapi.LinkLookup](fake),
		Files: capability[vaultapi.FileReader](fake), Writer: capability[vaultapi.FileWriter](fake),
		Arranger: capability[vaultapi.Arranger](fake), LinkEditor: capability[vaultapi.LinkEditor](fake),
		Watch: capability[vaultapi.Watch](fake), Switch: capability[vaultapi.Switch](fake),
		Stores: capability[vaultapi.StoreOpener](fake), Cache: capability[vaultapi.CacheStore](fake),
	}
}

func searchOf(fake any) *SearchAccess {
	if fake == nil {
		return nil
	}
	return &SearchAccess{
		SearchIndex: capability[SearchIndex](fake), PassageSearch: capability[PassageSearch](fake),
		SearchFloors: capability[SearchFloors](fake),
	}
}

func (c *callers) Acquire(pluginID string, _ plugin.Holder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.acquired == nil {
		c.acquired = map[string]int{}
	}
	c.acquired[pluginID]++
}

func (c *callers) Release(pluginID string, _ plugin.Holder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acquired[pluginID]--
	if c.releases != nil {
		c.releases <- pluginID
	}
}

func (c *callers) getAcquireCount(pluginID string) int {
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

func TestAHeaderThatIsNoBearerTokenIsRefused(t *testing.T) {
	fakeCallers := newCallers()
	endpoint := &Endpoint{callers: fakeCallers.asCallers()}
	for _, header := range []string{"", "token-a", "Basic token-a", "bearer token-a"} {
		if _, err := endpoint.Authorize(header, manifest.PermissionVaultRead); readCode(t, err) != connect.CodeUnauthenticated {
			t.Errorf("header %q was accepted", header)
		}
	}
}

// The endpoint binds the loopback interface and responds for a token.
func TestHostEndpoint_BindsLoopbackAndServes(t *testing.T) {
	fakeCallers := newCallers()
	fakeCallers.modelCalls = &scriptedCalls{done: "ok"}
	endpoint, err := Open(t.Context(), Address, fakeCallers.asCallers())
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

	client := numenv1connect.NewHostConnectionServiceClient(http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	request := &v1.HostConnectionServiceCallRequest{Operation: "llm.complete", Payload: []byte("u")}
	got, err := call(t.Context(), client, request)
	if err != nil || string(got.done) != "ok" {
		t.Fatalf("answered %+v, %v", got, err)
	}

	if err := endpoint.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := call(t.Context(), client, request); err == nil {
		t.Error("a closed endpoint answered")
	}
}
