package plugin_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	requiringPlugin  = "org.example.requiring"
	requiringService = "org.example.RequiringService"
	requiringEvent   = "onCommand:run"
)

// fakeIdentity is the identity of the device at the outer edge of the host: the one thing a test flips.
type fakeIdentity struct{ isPresent atomic.Bool }

func (f *fakeIdentity) IsRequirementMet(_ context.Context, requirement string) bool {
	return requirement == catalogmanifest.RequirementIdentity && f.isPresent.Load()
}

type requiringHost struct {
	keptHost
	identity *fakeIdentity
}

func newRequiringHost(t *testing.T) requiringHost {
	t.Helper()
	clock := testsupport.NewClock()
	host := newEventHostWithClock(t, clock, map[string]catalogmanifest.Manifest{
		requiringPlugin: {
			Backend:          catalogmanifest.Backend{Lifecycle: catalogmanifest.LifecyclePersistent, Requires: []string{catalogmanifest.RequirementIdentity}},
			Services:         []string{requiringService},
			ActivationEvents: []string{requiringEvent},
		},
	}, map[string]int{requiringPlugin: servePlugin(t, nil)})
	launcher := &keptLauncher{t: t, port: servePlugin(t, nil)}
	host.Supervision.Processes.SetLauncher(func(ctx context.Context, p catalogmanifest.Discovered, port int, token string) (*os.Process, io.ReadCloser, error) {
		return launcher.launch(ctx, p, port, token)
	})
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = keptStartLimit
	host.Supervision.Processes.SetActivationPolicy(policy)
	identity := &fakeIdentity{}
	host.Management.Requirements.SetSource(identity)
	return requiringHost{keptHost: keptHost{Host: host, clock: clock, launcher: launcher}, identity: identity}
}

func (h requiringHost) setPresent(isPresent bool) {
	h.identity.isPresent.Store(isPresent)
	h.Management.Keeper.Refresh(context.Background())
}

func TestABackendThatRequiresTheIdentityIsNotStartedWhileTheDeviceHasNone(t *testing.T) {
	host := newRequiringHost(t)

	host.Management.Keeper.KeepAll(t.Context())
	host.Management.Keeper.Refresh(t.Context())

	time.Sleep(quietPeriod)
	if host.isRunning(requiringPlugin) || host.launcher.launches.Load() != 0 {
		t.Errorf("running = %v, launches = %d, want the backend not started", host.isRunning(requiringPlugin), host.launcher.launches.Load())
	}
}

func TestABackendThatRequiresTheIdentityStartsOnceTheDeviceHasOne(t *testing.T) {
	host := newRequiringHost(t)
	host.Management.Keeper.KeepAll(t.Context())

	host.setPresent(true)

	host.waitForRunning(t, requiringPlugin)
	if got := host.launcher.launches.Load(); got != 1 {
		t.Errorf("launches = %d, want 1", got)
	}
}

func TestABackendThatRequiresTheIdentityStopsWhenTheIdentityIsGoneAndIsNotStartedAgain(t *testing.T) {
	host := newRequiringHost(t)
	host.setPresent(true)
	host.waitForRunning(t, requiringPlugin)

	host.setPresent(false)

	wiretest.WaitUntil(t, "the backend to stop", func() bool { return !host.isRunning(requiringPlugin) })
	time.Sleep(quietPeriod)
	if host.isRunning(requiringPlugin) || host.launcher.launches.Load() != 1 {
		t.Errorf("running = %v, launches = %d, want stopped after the one start", host.isRunning(requiringPlugin), host.launcher.launches.Load())
	}
}

func TestACallForABackendThatRequiresTheIdentityIsRefusedAndStartsNoProcess(t *testing.T) {
	host := newRequiringHost(t)

	_, err := host.Management.Activation.EnsureActive(t.Context(), requiringPlugin, getTestHolder())

	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.RequirementUnmet {
		t.Fatalf("error = %v, want an unavailable plugin with the requirement unmet", err)
	}
	if host.launcher.launches.Load() != 0 {
		t.Errorf("launches = %d, want 0", host.launcher.launches.Load())
	}
}

func TestAWindowCallForABackendThatRequiresTheIdentityAnswersTheCodeOfTheRequirement(t *testing.T) {
	host := newRequiringHost(t)
	proxy := servicebridge.Open(host.Management.Activation, host.Management.Registry, host.Management.Registry)
	window := httptest.NewServer(proxy)
	t.Cleanup(window.Close)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, window.URL+"/"+requiringService+"/GetAccountState", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}

	if res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(answer), `"code":"`+servicebridge.RequirementUnmetCode+`"`) {
		t.Errorf("the window was answered %d %s, want 503 with the code of the requirement", res.StatusCode, answer)
	}
	if host.launcher.launches.Load() != 0 {
		t.Errorf("launches = %d, want 0", host.launcher.launches.Load())
	}
}

func TestAnEventDoesNotStartABackendThatRequiresTheIdentityWhileTheDeviceHasNone(t *testing.T) {
	host := newRequiringHost(t)

	activated, err := host.Management.Events.TriggerEvent(t.Context(), requiringEvent)

	if err != nil || len(activated) != 0 || host.launcher.launches.Load() != 0 {
		t.Errorf("activated = %v, error = %v, launches = %d, want nothing started and no error", activated, err, host.launcher.launches.Load())
	}
}
