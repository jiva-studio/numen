package proofreaderbridge

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugintest"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// mockProofreadingServer answers a request with what its fields hold and keeps
// the requests it was given.
type mockProofreadingServer struct {
	mu       sync.Mutex
	requests []*v1.ProofreadTextRequest
	response *v1.ProofreadTextResponse
	failure  error
}

func (s *mockProofreadingServer) ProofreadText(
	_ context.Context,
	req *connect.Request[v1.ProofreadTextRequest],
) (*connect.Response[v1.ProofreadTextResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req.Msg)
	if s.failure != nil {
		return nil, s.failure
	}
	return connect.NewResponse(s.response), nil
}

func (s *mockProofreadingServer) lastRequest(t *testing.T) *v1.ProofreadTextRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		t.Fatal("the plugin was asked nothing")
	}
	return s.requests[len(s.requests)-1]
}

func createTestProofreadingPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "proofreading")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.tools.proofreading",
		"name": "Proofreading Plugin",
		"version": "1.0.0",
		"description": "Proofreading",
		"capabilities": ["proofreading"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onCommand:proofread"],
		"permissions": ["ai:complete"],
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

// openTestProofreader serves the mock as the proofreading plugin and returns the
// bridge to it with the host that runs it.
func openTestProofreader(t *testing.T, mock *mockProofreadingServer) (*Proofreader, *plugin.Host) {
	t.Helper()
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(mock))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestProofreadingPluginTree(t, tmp)

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	bridge, err := Open(host)
	if err != nil {
		t.Fatalf("OpenProofreaderBridge failed: %v", err)
	}
	t.Cleanup(func() { _ = bridge.Close() })
	return bridge, host
}

func TestProofreaderBridgeSendsAScanAndReadsTheCorrections(t *testing.T) {
	mock := &mockProofreadingServer{response: &v1.ProofreadTextResponse{
		Corrections: []*v1.ProofreadCorrection{{Number: 10, Last: 10, Text: "the cat sat"}},
		Uncorrected: 1,
	}}
	bridge, host := openTestProofreader(t, mock)

	if name := bridge.GetName(); name != "Proofreading Plugin" {
		t.Fatalf("unexpected name: %s", name)
	}

	pages := []domain.ProofreadPage{
		{Number: 3, Lines: []domain.ProofreadLine{{Number: 10, Text: "the cta sat"}, {Number: 11, Text: "on the mat"}}},
		{Number: 4, Lines: []domain.ProofreadLine{{Number: 12, Text: "the dgo ran"}}},
	}
	got, err := bridge.ProofreadScan(t.Context(), pages)
	if err != nil {
		t.Fatalf("ProofreadScan failed: %v", err)
	}

	want := domain.ScanProofreading{
		Corrections: []domain.ProofreadCorrection{{Number: 10, Last: 10, Text: "the cat sat"}},
		Uncorrected: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	req := mock.lastRequest(t)
	if req.GetKind() != v1.ProofreadKind_PROOFREAD_KIND_SCAN {
		t.Errorf("kind %v", req.GetKind())
	}
	if req.GetScan().GetMaxEditDistance() != 0 || req.GetSpeech() != nil {
		t.Errorf("options %v", req.GetOptions())
	}
	if len(req.GetGroups()) != 2 || req.GetGroups()[0].GetNumber() != 3 ||
		req.GetGroups()[0].GetLines()[1].GetNumber() != 11 || req.GetGroups()[0].GetLines()[1].GetText() != "on the mat" ||
		req.GetGroups()[1].GetLines()[0].GetText() != "the dgo ran" {
		t.Errorf("groups %v", req.GetGroups())
	}

	sup, ok := host.GetSupervisor("md.numen.tools.proofreading")
	if !ok {
		t.Fatalf("supervisor not found")
	}
	if !sup.IsIdle(0, time.Now()) {
		t.Fatalf("expected supervisor to be idle after the proofreading returned")
	}
}

func TestProofreaderBridgeSendsSpeechAsOneGroupWithItsWindow(t *testing.T) {
	mock := &mockProofreadingServer{response: &v1.ProofreadTextResponse{
		Corrections: []*v1.ProofreadCorrection{{Number: 0, Last: 1, Text: "We should go."}},
		Uncorrected: 2,
		NextLine:    7,
		IsDone:      true,
	}}
	bridge, _ := openTestProofreader(t, mock)

	lines := []domain.ProofreadLine{{Number: 0, Text: "we should"}, {Number: 1, Text: "go"}}
	got, err := bridge.ProofreadSpeech(t.Context(), lines, 3)
	if err != nil {
		t.Fatalf("ProofreadSpeech failed: %v", err)
	}

	want := domain.SpeechProofreading{
		Corrections: []domain.ProofreadCorrection{{Number: 0, Last: 1, Text: "We should go."}},
		Uncorrected: 2,
		Next:        7,
		IsDone:      true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	req := mock.lastRequest(t)
	if req.GetKind() != v1.ProofreadKind_PROOFREAD_KIND_SPEECH || req.GetScan() != nil {
		t.Errorf("kind %v, options %v", req.GetKind(), req.GetOptions())
	}
	speech := req.GetSpeech()
	if speech.GetFromLine() != 3 || speech.GetSize() != 0 || speech.GetOverlap() != 0 || speech.GetLineLimit() != 0 {
		t.Errorf("options %v", speech)
	}
	if len(req.GetGroups()) != 1 || len(req.GetGroups()[0].GetLines()) != 2 ||
		req.GetGroups()[0].GetLines()[1].GetText() != "go" {
		t.Errorf("groups %v", req.GetGroups())
	}
}

func TestProofreaderBridgeReadsAPluginWithNothingToAskAsUnavailable(t *testing.T) {
	mock := &mockProofreadingServer{failure: connect.NewError(connect.CodeUnavailable, errors.New("no route"))}
	bridge, _ := openTestProofreader(t, mock)

	_, err := bridge.ProofreadScan(t.Context(), []domain.ProofreadPage{{Number: 1}})
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("scan: got %v, want %v", err, completionport.ErrUnavailable)
	}
	_, err = bridge.ProofreadSpeech(t.Context(), nil, 0)
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("speech: got %v, want %v", err, completionport.ErrUnavailable)
	}
}

func TestProofreaderBridgeKeepsAnyOtherFailureAsItIs(t *testing.T) {
	mock := &mockProofreadingServer{failure: connect.NewError(connect.CodeInternal, errors.New("boom"))}
	bridge, _ := openTestProofreader(t, mock)

	_, err := bridge.ProofreadScan(t.Context(), []domain.ProofreadPage{{Number: 1}})

	if err == nil || errors.Is(err, completionport.ErrUnavailable) || connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("got %v", err)
	}
}

func TestReadProofreadErrorReadsTheEndOfTheContextAsItsOwn(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := readProofreadError(ctx, connect.NewError(connect.CodeCanceled, errors.New("canceled")))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestReadProofreadErrorReadsAFailureThatIsNotAPluginAnswerAsUnavailable(t *testing.T) {
	err := readProofreadError(t.Context(), errors.New("connection failed"))

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("got %v", err)
	}
}

func TestProofreaderBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := Open(host)
	if err == nil {
		t.Fatal("expected error when no plugin registered, got nil")
	}
}

// inactiveHost resolves the proofreading plugin and fails to start it with the
// error it holds.
type inactiveHost struct {
	plugintest.LeaseHost
	failure error
}

func (h *inactiveHost) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	return pluginport.Endpoint{}, h.failure
}

func TestProofreaderBridgeReadsAPluginThatCannotStartAsUnavailable(t *testing.T) {
	bridge, err := Open(&inactiveHost{failure: errors.New("the process exited")})
	if err != nil {
		t.Fatal(err)
	}

	_, err = bridge.ProofreadScan(t.Context(), []domain.ProofreadPage{{Number: 1}})
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("scan: got %v, want %v", err, completionport.ErrUnavailable)
	}
	_, err = bridge.ProofreadSpeech(t.Context(), nil, 0)
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("speech: got %v, want %v", err, completionport.ErrUnavailable)
	}
}

func TestProofreaderBridgeKeepsTheEndOfAContextWhileStartingAsItIs(t *testing.T) {
	for _, end := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(end.Error(), func(t *testing.T) {
			bridge, err := Open(&inactiveHost{failure: fmt.Errorf("wait for ready: %w", end)})
			if err != nil {
				t.Fatal(err)
			}

			_, err = bridge.ProofreadSpeech(t.Context(), nil, 0)
			if !errors.Is(err, end) || errors.Is(err, completionport.ErrUnavailable) {
				t.Errorf("got %v, want %v alone", err, end)
			}
		})
	}
}

func TestProofreaderBridgeReadsAModelThePersonHasNotAllowedAsNoConsent(t *testing.T) {
	mock := &mockProofreadingServer{failure: connect.NewError(connect.CodePermissionDenied, errors.New("not allowed"))}
	bridge, _ := openTestProofreader(t, mock)

	_, err := bridge.ProofreadScan(t.Context(), []domain.ProofreadPage{{Number: 1}})
	if !errors.Is(err, completionport.ErrNoConsent) {
		t.Errorf("got %v, want %v", err, completionport.ErrNoConsent)
	}
}

// An unavailable plugin is stated once, with the message the plugin gave.
func TestProofreaderBridgeStatesAnUnavailablePluginOnce(t *testing.T) {
	mock := &mockProofreadingServer{failure: connect.NewError(connect.CodeUnavailable, errors.New("ai: routed agent did not answer: boom"))}
	bridge, _ := openTestProofreader(t, mock)

	_, err := bridge.ProofreadScan(t.Context(), []domain.ProofreadPage{{Number: 1}})
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Fatalf("got %v, want %v", err, completionport.ErrUnavailable)
	}
	if want := "unavailable: ai: routed agent did not answer: boom"; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
