package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func (s *mockProofreadingServer) ProofreadText(
	_ context.Context,
	req *connect.Request[v1.ProofreadTextRequest],
) (*connect.Response[v1.ProofreadTextResponse], error) {
	var corrections []*v1.ProofreadCorrection
	for _, group := range req.Msg.GetGroups() {
		for _, line := range group.GetLines() {
			corrections = append(corrections, &v1.ProofreadCorrection{
				Number: line.GetNumber(),
				Last:   line.GetNumber() + 1,
				Text:   req.Msg.GetKind().String() + ": " + line.GetText(),
			})
		}
	}
	return connect.NewResponse(&v1.ProofreadTextResponse{
		Corrections: corrections,
		Uncorrected: req.Msg.GetSize(),
		NextLine:    req.Msg.GetFromLine(),
		IsDone:      req.Msg.GetLineLimit() == 0,
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

	bridge, err := OpenProofreaderBridge(host)
	if err != nil {
		t.Fatalf("OpenProofreaderBridge failed: %v", err)
	}
	defer bridge.Close()

	if name := bridge.GetName(); name != "Proofreading Plugin" {
		t.Fatalf("unexpected name: %s", name)
	}

	answer, err := bridge.Proofread(t.Context(), domain.ProofreadRequest{
		Kind: domain.ProofreadSpeech,
		Groups: []domain.ProofreadGroup{{Number: 1, Lines: []domain.ProofreadLine{
			{Number: 4, Text: "line 4 text"},
			{Number: 5, Text: "line 5 text"},
		}}},
		Size:      7,
		FromLine:  2,
		LineLimit: 0,
	})
	if err != nil {
		t.Fatalf("Proofread failed: %v", err)
	}
	want := domain.ProofreadAnswer{
		Corrections: []domain.ProofreadCorrection{
			{Number: 4, Last: 5, Text: "PROOFREAD_KIND_SPEECH: line 4 text"},
			{Number: 5, Last: 6, Text: "PROOFREAD_KIND_SPEECH: line 5 text"},
		},
		Uncorrected: 7,
		NextLine:    2,
		IsDone:      true,
	}
	if !reflect.DeepEqual(answer, want) {
		t.Fatalf("answer %+v, want %+v", answer, want)
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

	_, err := OpenProofreaderBridge(host)
	if err == nil {
		t.Fatal("expected error when no plugin registered, got nil")
	}
}

func TestProofreaderBridgeRefusesAnUnknownKind(t *testing.T) {
	bridge := &ProofreaderBridge{}
	_, err := bridge.Proofread(t.Context(), domain.ProofreadRequest{
		Groups: []domain.ProofreadGroup{{Number: 1}},
	})
	if !errors.Is(err, errUnknownProofreadKind) {
		t.Fatalf("got %v", err)
	}
}

func TestProofreaderBridgeAsksNothingWithoutLines(t *testing.T) {
	bridge := &ProofreaderBridge{}
	answer, err := bridge.Proofread(t.Context(), domain.ProofreadRequest{Kind: domain.ProofreadScan})
	if err != nil || !answer.IsDone || len(answer.Corrections) != 0 {
		t.Fatalf("answer %+v, %v", answer, err)
	}
}
