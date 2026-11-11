package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestABrokenManifestIsSkippedAndShownAsFailed(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	broken := filepath.Join(tmp, "broken")
	if err := os.Mkdir(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "manifest.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	var reported []error
	host.Failures.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.Registry.GetPlugin("org.example.sample"); !ok {
		t.Fatal("the valid plugin was not registered")
	}
	if len(reported) != 1 {
		t.Fatalf("expected one reported problem, got %v", reported)
	}
	var failed []PluginStatus
	for _, status := range host.Registry.GetPluginStatuses() {
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
	wiretest.CreateTestPluginTree(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	host.Processes.SetLauncher(startSleepLauncher(t, &atomic.Int32{}, nil))
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Activation.Release("org.example.sample")
	exited := supervisorOf(host, "org.example.sample").GetExitSignal()

	if err := os.RemoveAll(filepath.Join(tmp, "sample")); err != nil {
		t.Fatal(err)
	}
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.Registry.GetPlugin("org.example.sample"); ok {
		t.Fatal("a plugin removed from disk is still registered")
	}
	waitForProcessExit(t, exited)
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, errPluginNotFound) {
		t.Fatalf("expected errPluginNotFound, got %v", err)
	}
}

func TestDiscover_ChangedManifestRestartsPlugin(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	var started atomic.Int32
	host.Processes.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Activation.Release("org.example.sample")
	exited := supervisorOf(host, "org.example.sample").GetExitSignal()

	manifestPath := filepath.Join(tmp, "sample", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := []byte(string(data[:len(data)-1]) + ` , "description": "changed"}`)
	if err := os.WriteFile(manifestPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	p, ok := host.Registry.GetPlugin("org.example.sample")
	if !ok || p.Manifest.Description != "changed" {
		t.Fatalf("expected the changed manifest to be registered, got %+v", p.Manifest)
	}
	waitForProcessExit(t, exited)
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive after the manifest change failed: %v", err)
	}
	host.Activation.Release("org.example.sample")
	if started.Load() != 2 {
		t.Fatalf("expected the plugin to be launched again, got %d launches", started.Load())
	}
	if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAPluginWhoseDirectoryCannotBeReadStaysRegistered(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	manifestPath := filepath.Join(tmp, "sample", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.Registry.GetPlugin("org.example.sample"); !ok {
		t.Fatal("a plugin with a failed scan was unregistered")
	}
	failed := 0
	for _, status := range host.Registry.GetPluginStatuses() {
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
	wiretest.CreateTestPluginTree(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := host.Discovery.Discover(); err != nil {
				t.Errorf("Discover failed: %v", err)
			}
		})
	}
	wg.Wait()

	if plugins := host.Registry.GetPlugins(); len(plugins) != 1 {
		t.Fatalf("expected one plugin, got %d", len(plugins))
	}
}

func TestEnsureActiveWaitsForADiscoveryInProgress(t *testing.T) {
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.Processes.SetLauncher(startSleepLauncher(t, &started, nil))
	host.Registry.discovering.Lock()

	reply := make(chan error, 1)
	calling := make(chan struct{})
	go func() {
		close(calling)
		_, err := host.Activation.EnsureActive(t.Context(), "org.example.sample")
		reply <- err
	}()
	<-calling
	runtime.Gosched()

	if host.Registry.discovering.TryRLock() {
		t.Fatal("the discovery lock is not held")
	}
	select {
	case err := <-reply:
		t.Fatalf("EnsureActive answered while a discovery was in progress: %v", err)
	default:
	}
	host.Registry.discovering.Unlock()
	if err := <-reply; err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAPluginThatDeclaresATabKindOfAnotherPluginIsRefused(t *testing.T) {
	tmp := t.TempDir()
	write := func(dir, id string) {
		path := filepath.Join(tmp, dir)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		text := `{"id":"` + id + `","name":"A","version":"1.0.0","protocol":1,"targets":["desktop.editor"],"frontend":{},` +
			`"contributes":{"tabKinds":[{"kind":"deck","title":"Deck"}]}}`
		if err := os.WriteFile(filepath.Join(path, "manifest.json"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("first", "org.example.first")
	write("second", "org.example.second")
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	var reported []error
	host.Failures.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.Registry.GetPlugin("org.example.first"); !ok {
		t.Error("the plugin first in the order of ids was refused")
	}
	if _, ok := host.Registry.GetPlugin("org.example.second"); ok {
		t.Error("the plugin that declares the same tab kind was registered")
	}
	if len(reported) == 0 || !errors.Is(reported[0], manifest.ErrInvalidContribution) {
		t.Errorf("reported = %v, want an invalid contribution", reported)
	}
}
