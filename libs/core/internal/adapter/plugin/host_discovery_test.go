package plugin

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestABrokenManifestIsSkippedAndShownAsFailed(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	broken := filepath.Join(tmp, "broken")
	if err := os.Mkdir(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "manifest.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewHost("desktop.editor", tmp)
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.GetPlugin("org.example.sample"); !ok {
		t.Fatal("the valid plugin was not registered")
	}
	if len(reported) != 1 {
		t.Fatalf("expected one reported problem, got %v", reported)
	}
	var failed []PluginStatus
	for _, status := range host.GetPluginStatuses() {
		if status.Failed {
			failed = append(failed, status)
		}
	}
	if len(failed) != 1 || failed[0].Dir != broken || failed[0].ID != "" || failed[0].FailureReason == "" {
		t.Fatalf("expected the broken plugin to be shown as failed, got %+v", failed)
	}
}

func TestAPluginRemovedFromDiskIsStoppedAndForgotten(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	host.SetLauncher(startSleepLauncher(t, &atomic.Int32{}, nil))
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("org.example.sample")
	exited := host.supervisors["org.example.sample"].GetExitSignal()

	if err := os.RemoveAll(filepath.Join(tmp, "sample")); err != nil {
		t.Fatal(err)
	}
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.GetPlugin("org.example.sample"); ok {
		t.Fatal("a plugin removed from disk is still registered")
	}
	waitForProcessExit(t, exited)
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, errPluginNotFound) {
		t.Fatalf("expected errPluginNotFound, got %v", err)
	}
}

func TestDiscover_ChangedManifestRestartsPlugin(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("org.example.sample")
	exited := host.supervisors["org.example.sample"].GetExitSignal()

	manifestPath := filepath.Join(tmp, "sample", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := []byte(string(data[:len(data)-1]) + ` , "description": "changed"}`)
	if err := os.WriteFile(manifestPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	p, ok := host.GetPlugin("org.example.sample")
	if !ok || p.Manifest.Description != "changed" {
		t.Fatalf("expected the changed manifest to be registered, got %+v", p.Manifest)
	}
	waitForProcessExit(t, exited)
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive after the manifest change failed: %v", err)
	}
	host.Release("org.example.sample")
	if started.Load() != 2 {
		t.Fatalf("expected the plugin to be launched again, got %d launches", started.Load())
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAPluginWhoseDirectoryCannotBeReadStaysRegistered(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	manifestPath := filepath.Join(tmp, "sample", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.GetPlugin("org.example.sample"); !ok {
		t.Fatal("a plugin with a failed scan was unregistered")
	}
	failed := 0
	for _, status := range host.GetPluginStatuses() {
		if status.Failed {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("expected the failed scan to be reported once, got %d", failed)
	}
}

func TestConcurrentDiscoveriesLeaveTheRegistryConsistent(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Discover(); err != nil {
				t.Errorf("Discover failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if plugins := host.GetPlugins(); len(plugins) != 1 {
		t.Fatalf("expected one plugin, got %d", len(plugins))
	}
}

func TestEnsureActiveWaitsForADiscoveryInProgress(t *testing.T) {
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	host.discoverMu.Lock()

	reply := make(chan error, 1)
	calling := make(chan struct{})
	go func() {
		close(calling)
		_, err := host.EnsureActive(t.Context(), "org.example.sample")
		reply <- err
	}()
	<-calling
	runtime.Gosched()

	if host.discoverMu.TryRLock() {
		t.Fatal("the discovery lock is not held")
	}
	select {
	case err := <-reply:
		t.Fatalf("EnsureActive answered while a discovery was in progress: %v", err)
	default:
	}
	host.discoverMu.Unlock()
	if err := <-reply; err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAPluginStoppedForMemoryIsReported(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	sup, _ := host.GetSupervisor("org.example.sample")
	onExceeded := sup.GetOnMemoryExceeded()
	if onExceeded == nil {
		t.Fatal("the supervisor of the plugin reports no memory stop")
	}
	onExceeded()

	if len(reported) != 1 || !strings.Contains(reported[0].Error(), "org.example.sample was stopped") {
		t.Fatalf("expected the stop to be reported for the plugin, got %v", reported)
	}
}
