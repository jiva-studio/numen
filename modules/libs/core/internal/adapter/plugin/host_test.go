package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "ocr")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.ocr",
		"name": "OCR Plugin",
		"version": "1.0.0",
		"description": "OCR",
		"capabilities": ["recognition"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.pdf"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/ocr",
			"serviceName": "numen.v1.OcrService"
		},
		"frontend": {
			"entrypoint": "dist/index.js",
			"styles": "dist/styles.css"
		},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestHostDiscoveryAndLookup(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p, ok := host.GetPlugin("md.numen.ocr")
	if !ok || p.Manifest.ID != "md.numen.ocr" {
		t.Fatal("expected to find md.numen.ocr")
	}

	capPlugin, ok := host.FindPluginByCapability("recognition")
	if !ok || capPlugin.Manifest.ID != "md.numen.ocr" {
		t.Fatal("expected to find plugin by capability recogniser")
	}
}

func TestHostLifecycleWithMockLauncher(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port == 0 || endpoint.Token == "" {
		t.Fatalf("expected non-zero port and token, got %+v", endpoint)
	}

	// Calling EnsureActive again should return the existing active endpoint
	endpoint2, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("second EnsureActive failed: %v", err)
	}
	if endpoint.Port != endpoint2.Port || endpoint.Token != endpoint2.Token {
		t.Fatalf("expected identical endpoints, got %+v and %+v", endpoint, endpoint2)
	}

	// Release twice to enter idle state
	host.Release("md.numen.ocr")
	host.Release("md.numen.ocr")

	// TriggerEvent for matching event
	activated, err := host.TriggerEvent(ctx, "onFile:test.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if len(activated) != 1 || activated[0] != "md.numen.ocr" {
		t.Fatalf("expected md.numen.ocr to be activated, got %v", activated)
	}

	// EvictIdle test
	evicted := host.EvictIdle(ctx, 10*time.Millisecond, 50*time.Millisecond)
	// Because idleTimeoutSeconds is 1s, right now it is not evicted yet
	if len(evicted) != 0 {
		t.Fatalf("expected 0 evicted before threshold, got %v", evicted)
	}

	// Cleanup
	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEvictLoopTeardown(t *testing.T) {
	host := NewHost("desktop.editor")
	if host.isEvictStopped {
		t.Fatal("expected evict loop to be running")
	}

	ctx := t.Context()
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if !host.isEvictStopped {
		t.Fatal("expected evict loop to be stopped after TeardownAll")
	}

	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("second TeardownAll failed: %v", err)
	}
}

func TestIsWhitelistedEnv(t *testing.T) {
	allowed := []string{
		"PATH", "path", "Path",
		"HOME", "home",
		"USER", "user",
		"TMP", "tmp",
		"TEMP", "temp",
		"SYSTEMROOT", "SystemRoot",
		"SYSTEMDRIVE", "SystemDrive",
		"APPDATA", "AppData",
		"LOCALAPPDATA", "LocalAppData",
		"LANG", "lang",
		"LC_ALL", "lc_ctype", "LC_MESSAGES",
	}
	for _, k := range allowed {
		if !isWhitelistedEnv(k) {
			t.Errorf("expected %s to be allowed in environment whitelist", k)
		}
	}

	disallowed := []string{
		"AWS_SECRET_ACCESS_KEY",
		"GITHUB_TOKEN",
		"PRIVATE_KEY",
		"DATABASE_URL",
		"SECRET",
		"MY_VAR",
		"FOO",
	}
	for _, k := range disallowed {
		if isWhitelistedEnv(k) {
			t.Errorf("expected %s to be blocked from environment whitelist", k)
		}
	}
}

func TestHostEmitEventActivation(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()

	// Initial state is dormant
	sup := host.supervisors["md.numen.ocr"]
	if sup == nil || sup.GetState() != StateDormant {
		t.Fatalf("expected initial state dormant, got %v", sup.GetState())
	}

	// Emit onFile event via debounced event bus
	host.EmitEvent("onFile:document.pdf")
	host.GetEventBus().Flush(ctx)

	// Plugin should now be activated
	if sup.GetState() != StateActive && sup.GetState() != StateIdle {
		t.Fatalf("expected state active or idle after EmitEvent, got %v", sup.GetState())
	}

	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostDrainsStdoutAfterHandshake(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe failed: %v", err)
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, pr, nil
	})

	writeDone := make(chan error, 1)
	go func() {
		defer pw.Close()
		if _, err := pw.Write([]byte("NUMEN_READY port=54321\n")); err != nil {
			writeDone <- err
			return
		}
		// Write >64KB to overflow the standard OS pipe buffer.
		largePayload := bytes.Repeat([]byte("A"), 128*1024)
		if _, err := pw.Write(largePayload); err != nil {
			writeDone <- err
			return
		}
		writeDone <- nil
	}()

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port != 54321 {
		t.Fatalf("expected port 54321, got %d", endpoint.Port)
	}

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write to pipe failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for large stdout write to complete")
	}

	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}
