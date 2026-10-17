package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestABrokenManifestIsSkippedAndShownAsFailed(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
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

	if _, ok := host.GetPlugin("md.numen.ocr"); !ok {
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
	if len(failed) != 1 || failed[0].ID != broken || failed[0].FailureReason == "" {
		t.Fatalf("expected the broken plugin to be shown as failed, got %+v", failed)
	}
}

func TestAPluginRemovedFromDiskIsStoppedAndForgotten(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	host.SetLauncher(startSleepLauncher(t, &atomic.Int32{}, nil))
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("md.numen.ocr")
	exited := getExitSignal(host.supervisors["md.numen.ocr"])

	if err := os.RemoveAll(filepath.Join(tmp, "ocr")); err != nil {
		t.Fatal(err)
	}
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.GetPlugin("md.numen.ocr"); ok {
		t.Fatal("a plugin removed from disk is still registered")
	}
	waitForProcessExit(t, exited)
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); !errors.Is(err, errPluginNotFound) {
		t.Fatalf("expected errPluginNotFound, got %v", err)
	}
}

func TestAChangedManifestReplacesTheRegisteredOneAndRestartsThePlugin(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("md.numen.ocr")
	exited := getExitSignal(host.supervisors["md.numen.ocr"])

	manifestPath := filepath.Join(tmp, "ocr", "manifest.json")
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

	p, ok := host.GetPlugin("md.numen.ocr")
	if !ok || p.Manifest.Description != "changed" {
		t.Fatalf("expected the changed manifest to be registered, got %+v", p.Manifest)
	}
	waitForProcessExit(t, exited)
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive after the manifest change failed: %v", err)
	}
	host.Release("md.numen.ocr")
	if started.Load() != 2 {
		t.Fatalf("expected the plugin to be launched again, got %d launches", started.Load())
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAPluginWhoseDirectoryCannotBeReadStaysRegistered(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	manifestPath := filepath.Join(tmp, "ocr", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if _, ok := host.GetPlugin("md.numen.ocr"); !ok {
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
	createTestPluginTree(t, tmp)
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
