package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

type mockProofreadingServer struct{}

func (s *mockProofreadingServer) GetProofreaderInfo(
	_ context.Context,
	_ *connect.Request[v1.GetProofreaderInfoRequest],
) (*connect.Response[v1.GetProofreaderInfoResponse], error) {
	return connect.NewResponse(&v1.GetProofreaderInfoResponse{
		Name: "test-proofreader-model",
	}), nil
}

func (s *mockProofreadingServer) Proofread(
	_ context.Context,
	req *connect.Request[v1.ProofreadRequest],
) (*connect.Response[v1.ProofreadResponse], error) {
	replies := make(map[int32]string)
	for _, batch := range req.Msg.GetBatches() {
		replies[batch.GetNumber()] = req.Msg.GetInstruction() + " corrected: " + batch.GetPrompt()
	}
	return connect.NewResponse(&v1.ProofreadResponse{
		Replies: replies,
	}), nil
}

func createTestProofreadingPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "proofreading")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.proofreading",
		"name": "Proofreading Plugin",
		"version": "1.0.0",
		"description": "Proofreading",
		"capabilities": ["proofreading"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onCommand:proofread"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/proofreading",
			"serviceName": "numen.v1.ProofreadingService"
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

func TestProofreaderBridge(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(&mockProofreadingServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestProofreadingPluginTree(t, tmp)

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

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	bridge, err := OpenProofreaderBridge(host, "proofread the speech")
	if err != nil {
		t.Fatalf("OpenProofreaderBridge failed: %v", err)
	}
	defer bridge.Close()

	if name := bridge.GetName(); name != "Proofreading Plugin" {
		t.Fatalf("unexpected name: %s", name)
	}

	batches := []domain.ProofreadBatch{
		{Number: 1, Prompt: "line 1 text"},
		{Number: 2, Prompt: "line 2 text"},
	}
	replies, err := bridge.Proofread(t.Context(), batches)
	if err != nil {
		t.Fatalf("Proofread failed: %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(replies))
	}
	if replies[1] != "proofread the speech corrected: line 1 text" {
		t.Fatalf("unexpected reply: %s", replies[1])
	}

	host.mu.RLock()
	sup, ok := host.supervisors["md.numen.proofreading"]
	host.mu.RUnlock()
	if !ok {
		t.Fatalf("supervisor not found")
	}
	if !sup.IsIdle(0, time.Now()) {
		t.Fatalf("expected supervisor to be idle after Proofread returned")
	}
}

func TestProofreaderBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := OpenProofreaderBridge(host, "proofread the speech")
	if err == nil {
		t.Fatal("expected error when no plugin registered, got nil")
	}
}

func (s *mockProofreadingServer) ProofreadText(
	_ context.Context,
	_ *connect.Request[v1.ProofreadTextRequest],
) (*connect.Response[v1.ProofreadTextResponse], error) {
	return connect.NewResponse(&v1.ProofreadTextResponse{IsDone: true}), nil
}
