package plugin

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// bundledPluginsDir is the directory of the plugins this repository ships, from this package.
const bundledPluginsDir = "../../../../../plugins"

// searchRelevanceID is the id of the bundled search relevance plugin.
const searchRelevanceID = "md.numen.tools.search.relevance"

// newEventManifest is a plugin with a backend that declares the activation events.
func newEventManifest(events ...string) pluginport.PluginManifest {
	return pluginport.PluginManifest{ActivationEvents: events}
}

func TestTriggerEventStartsOnlyPluginsThatDeclareTheEvent(t *testing.T) {
	startup, vectors, files := &eventRecorder{}, &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{
			"md.test.startup": newEventManifest("onStartup"),
			"md.test.vectors": newEventManifest("onVectorsReady"),
			"md.test.files":   newEventManifest("onFile:md"),
		},
		map[string]int{
			"md.test.startup": servePlugin(t, startup),
			"md.test.vectors": servePlugin(t, vectors),
			"md.test.files":   servePlugin(t, files),
		},
	)

	for _, step := range []struct {
		event string
		want  []string
	}{
		{"onStartup", []string{"md.test.startup"}},
		{"onVectorsReady:01VAULT", []string{"md.test.vectors"}},
		{"onFile:md", []string{"md.test.files"}},
		{"onFile:pdf", []string{}},
	} {
		activated, err := host.TriggerEvent(t.Context(), step.event)
		if err != nil {
			t.Fatalf("TriggerEvent(%s) failed: %v", step.event, err)
		}
		if !slices.Equal(activated, step.want) {
			t.Errorf("TriggerEvent(%s) started %v, want %v", step.event, activated, step.want)
		}
	}
	for name, got := range map[string][]string{
		"onStartup":      startup.getEvents(),
		"onVectorsReady": vectors.getEvents(),
		"onFile:md":      files.getEvents(),
	} {
		if len(got) != 1 || !strings.HasPrefix(got[0], name) {
			t.Errorf("the plugin declaring %s received %v, want that event once", name, got)
		}
	}
}

func TestTriggerEventStartsNoPluginOnAnEventNoneDeclares(t *testing.T) {
	declaring, silent := &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]pluginport.PluginManifest{
			"md.test.declaring": newEventManifest("onStartup", "onVectorsReady", "onFile:md"),
			"md.test.silent":    newEventManifest(),
		},
		map[string]int{"md.test.declaring": servePlugin(t, declaring), "md.test.silent": servePlugin(t, silent)},
	)

	for _, event := range []string{"onOpen", "onFile:pdf", "onVectorsReady", "onStartup:01VAULT", "*"} {
		activated, err := host.TriggerEvent(t.Context(), event)
		if err != nil {
			t.Fatalf("TriggerEvent(%s) failed: %v", event, err)
		}
		if len(activated) != 0 {
			t.Errorf("TriggerEvent(%s) started %v, want none", event, activated)
		}
	}
	for _, id := range []string{"md.test.declaring", "md.test.silent"} {
		if isPluginRunning(t, host, id) {
			t.Errorf("%s runs after events it does not declare", id)
		}
	}
}

func TestSearchRelevanceStartsWhenVectorsAreReadyAndIsEvictedAfterIdle(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(bundledPluginsDir, "tools-search-relevance", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "tools-search-relevance")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	clock := testsupport.NewClock()
	host := NewHost(clock, "desktop.editor", root)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	if _, ok := host.GetPlugin(searchRelevanceID); !ok {
		t.Fatalf("the host did not discover %s", searchRelevanceID)
	}
	recorder := &eventRecorder{}
	launchIdleProcess(t, host, servePlugin(t, recorder))

	for _, event := range []string{"onStartup", "onFile:md"} {
		if _, err := host.TriggerEvent(t.Context(), event); err != nil {
			t.Fatalf("TriggerEvent(%s) failed: %v", event, err)
		}
		if isPluginRunning(t, host, searchRelevanceID) {
			t.Fatalf("the plugin runs after %s", event)
		}
	}

	activated, err := host.TriggerEvent(t.Context(), "onVectorsReady:01VAULT")
	if err != nil {
		t.Fatalf("TriggerEvent(onVectorsReady) failed: %v", err)
	}
	if !slices.Equal(activated, []string{searchRelevanceID}) || !isPluginRunning(t, host, searchRelevanceID) {
		t.Fatalf("the vectors-ready event started %v, want %s running", activated, searchRelevanceID)
	}
	if got := recorder.getEvents(); !slices.Equal(got, []string{"onVectorsReady:01VAULT"}) {
		t.Fatalf("the plugin received %v, want only the vectors-ready event", got)
	}

	clock.Advance(idleThreshold - time.Second)
	if evicted := host.EvictIdle(t.Context(), idleThreshold, 50*time.Millisecond); len(evicted) != 0 {
		t.Fatalf("evicted %v before the idle time", evicted)
	}
	clock.Advance(time.Second)
	if evicted := host.EvictIdle(t.Context(), idleThreshold, 50*time.Millisecond); !slices.Equal(evicted, []string{searchRelevanceID}) {
		t.Fatalf("evicted %v after the idle time, want %s", evicted, searchRelevanceID)
	}
	if isPluginRunning(t, host, searchRelevanceID) {
		t.Fatal("the plugin runs after it was evicted")
	}
}

// isPluginRunning reports whether the host runs the plugin.
func isPluginRunning(t *testing.T, host *Host, pluginID string) bool {
	t.Helper()
	status, ok := host.GetPluginStatus(pluginID)
	if !ok {
		t.Fatalf("the host has no plugin %s", pluginID)
	}
	return status.IsActive
}

// launchIdleProcess makes the host start every plugin as an idle child process that reports the port.
// The host is torn down before the processes are killed.
func launchIdleProcess(t *testing.T, host *Host, port int) {
	t.Helper()
	var mu sync.Mutex
	var procs []*os.Process
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, proc := range procs {
			plugintest.KillProcess(t, proc)
		}
	})
	host.SetLauncher(func(ctx context.Context, _ pluginport.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		mu.Lock()
		procs = append(procs, cmd.Process)
		mu.Unlock()
		return cmd.Process, plugintest.GetReadyOutput(port), nil
	})
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond); err != nil {
			t.Errorf("TeardownAll failed: %v", err)
		}
	})
}
