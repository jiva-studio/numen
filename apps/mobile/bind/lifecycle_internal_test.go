package bind

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const lifecycleService = "md.numen.test"

type noRequest struct{}

type noAnswer struct{}

// answeringPlugin is a linked plugin whose service answers Suspend and Resume, and records what it was asked.
type answeringPlugin struct {
	mu     sync.Mutex
	called []string
}

func (p *answeringPlugin) getCalled() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.called)
}

func (p *answeringPlugin) mount(failure error) func(*server.Server) error {
	return func(s *server.Server) error {
		service := s.NewAPI(lifecycleService)
		for _, method := range []string{"Suspend", "Resume"} {
			server.Handle(service, method, func(context.Context, noRequest) (noAnswer, error) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.called = append(p.called, method)
				return noAnswer{}, failure
			})
		}
		return service.Err()
	}
}

func newLifecycleCall(t *testing.T, plugin *answeringPlugin, failure error) lifecycleCall {
	t.Helper()
	handler, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: "plugin-token"}, plugin.mount(failure))
	if err != nil {
		t.Fatal(err)
	}
	return lifecycleCall{handler: handler, service: lifecycleService, token: "plugin-token"}
}

func newRunningWith(calls ...lifecycleCall) *Core {
	return &Core{running: &runningCore{lifecycles: calls}}
}

func TestSuspendAndResumeOfACoreThatIsNotRunningDoNothing(t *testing.T) {
	core := NewCore()
	if err := core.Suspend(); err != nil {
		t.Errorf("Suspend: %v", err)
	}
	if err := core.Resume(); err != nil {
		t.Errorf("Resume: %v", err)
	}
}

func TestSuspendAndResumeAskEveryPluginThatFollowsTheScreen(t *testing.T) {
	first, second := &answeringPlugin{}, &answeringPlugin{}
	core := newRunningWith(newLifecycleCall(t, first, nil), newLifecycleCall(t, second, nil))

	if err := core.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := core.Resume(); err != nil {
		t.Fatal(err)
	}

	for _, plugin := range []*answeringPlugin{first, second} {
		if got := plugin.getCalled(); !slices.Equal(got, []string{"Suspend", "Resume"}) {
			t.Errorf("a plugin was asked %v, want Suspend and then Resume", got)
		}
	}
}

func TestSuspendReturnsTheFailureOfAPluginAndStillAsksTheOthers(t *testing.T) {
	broken, other := &answeringPlugin{}, &answeringPlugin{}
	core := newRunningWith(newLifecycleCall(t, broken, errors.New("the server is away")), newLifecycleCall(t, other, nil))

	err := core.Suspend()

	if err == nil || !strings.Contains(err.Error(), lifecycleService) {
		t.Errorf("got %v, want the failure to name the service", err)
	}
	if got := other.getCalled(); !slices.Equal(got, []string{"Suspend"}) {
		t.Errorf("the plugin after the broken one was asked %v, want Suspend", got)
	}
}

func TestAPluginThatDoesNotKnowTheTokenIsNotAsked(t *testing.T) {
	plugin := &answeringPlugin{}
	call := newLifecycleCall(t, plugin, nil)
	call.token = "another-token"

	if err := newRunningWith(call).Suspend(); err == nil {
		t.Error("got nil, want the refusal of the plugin")
	}
	if got := plugin.getCalled(); len(got) != 0 {
		t.Errorf("a call with a wrong token reached the plugin: %v", got)
	}
}

type memoryKeys struct{ keys map[string]string }

func (m memoryKeys) Get(name string) (string, error) { return m.keys[name], nil }

func (m memoryKeys) Has(name string) bool {
	_, isThere := m.keys[name]
	return isThere
}

func (m memoryKeys) Set(name, value string) error {
	m.keys[name] = value
	return nil
}

func (m memoryKeys) Delete(name string) error {
	delete(m.keys, name)
	return nil
}

func TestACoreWithoutAKeyVaultHasNoKeyStore(t *testing.T) {
	if NewCore().getKeyStore() != nil {
		t.Error("a core with no key vault has a key store")
	}
}

func TestTheKeyStoreOfACoreKeepsKeysInTheVaultThePlatformGave(t *testing.T) {
	core := NewCore()
	keys := memoryKeys{keys: map[string]string{}}
	core.SetKeyVault(keys)

	store := core.getKeyStore()
	if store == nil {
		t.Fatal("a core with a key vault has no key store")
	}
	if err := store.SetKey(t.Context(), "identity", "secret"); err != nil {
		t.Fatal(err)
	}
	if keys.keys["identity"] != "secret" {
		t.Errorf("the platform vault holds %v, want the key the store was given", keys.keys)
	}
}
