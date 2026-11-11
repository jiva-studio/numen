package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func writeSecondPlugin(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "second")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	body := strings.ReplaceAll(`{
		"id": "org.example.second",
		"name": "Second Plugin",
		"version": "1.0.0",
		"description": "Second",
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"backend": {"entrypoint": "bin/second"},
		"settings": []
	}`, "\t", "")
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

// A plugin whose process never says it is ready holds back no other plugin: the others start, or
// fail, while it waits, and so does everything that reads what the host knows.
func TestAnotherPluginStartsWhileAPluginIsNotReady(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	writeSecondPlugin(t, tmp)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the silent process: %v", err)
	}
	t.Cleanup(func() { plugintest.KillProcess(t, cmd.Process) })
	silent, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close the pipe of the silent plugin: %v", err)
		}
	})
	stuck := make(chan struct{})
	host.Processes.SetLauncher(func(_ context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		if p.Manifest.ID == "org.example.sample" {
			close(stuck)
			return cmd.Process, silent, nil
		}
		return nil, nil, errors.New("the second plugin does not run in this test")
	})

	stuckDone := make(chan struct{})
	var stuckErr error
	go func() {
		defer close(stuckDone)
		_, stuckErr = host.Activation.EnsureActive(t.Context(), "org.example.sample")
	}()
	select {
	case <-stuck:
	case <-time.After(5 * time.Second):
		t.Fatal("the first plugin never started")
	}

	reply := make(chan error, 1)
	go func() {
		_, err := host.Activation.EnsureActive(t.Context(), "org.example.second")
		host.Registry.GetPluginInfos()
		host.Registry.GetPluginStatuses()
		reply <- err
	}()
	select {
	case err := <-reply:
		if err == nil {
			t.Error("the second plugin was expected to fail in its launcher")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the second plugin waited for the plugin that does not become ready")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill the silent process: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close the pipe of the silent plugin: %v", err)
	}
	<-stuckDone
	if stuckErr == nil {
		t.Error("the plugin that never became ready was expected to fail")
	}
}
