package container_test

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
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

type mockProofreadService struct {
	numenv1connect.UnimplementedProofreadingServiceHandler
}

func (s *mockProofreadService) ProofreadText(
	_ context.Context,
	req *connect.Request[v1.ProofreadTextRequest],
) (*connect.Response[v1.ProofreadTextResponse], error) {
	var corrections []*v1.ProofreadCorrection
	for _, group := range req.Msg.GetGroups() {
		for _, line := range group.GetLines() {
			corrections = append(corrections, &v1.ProofreadCorrection{
				Number: line.GetNumber(),
				Last:   line.GetNumber(),
				Text:   "fixed: " + line.GetText(),
			})
		}
	}
	return connect.NewResponse(&v1.ProofreadTextResponse{Corrections: corrections, IsDone: true}), nil
}

// A container nobody configured proofreads with nothing, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsProofreadsWithNothing(t *testing.T) {
	proofreader, why := container.Config{}.Proofreader("")
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

// A profile without a plugin host or when plugin is absent degrades gracefully.
func TestAProfileDegradesWhenPluginAbsent(t *testing.T) {
	proofreader, why := container.Config{}.Proofreader("careful")
	if why == nil {
		t.Fatal("want error when plugin absent, got nil")
	}
	if !errors.Is(why, port.ErrUnavailable) {
		t.Errorf("expected unavailable error, got: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

func TestTheProofreadingPluginWiredViaHostAnswersProofreader(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(&mockProofreadService{}))
	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "proofreading")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.tools.proofreading",
		"name": "test-model",
		"version": "1.0.0",
		"capabilities": ["proofreading"],
		"targets": ["desktop.editor"],
		"backend": {"type": "process", "entrypoint": "bin/proofreading"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ plugin.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := container.Config{PluginHost: host}

	proofreader, err := cfg.Proofreader("careful")
	if err != nil {
		t.Fatalf("Proofreader failed: %v", err)
	}
	if proofreader == nil {
		t.Fatal("no proofreader returned")
	}
	if got := proofreader.GetName(); got != "test-model" {
		t.Errorf("got name %q, want %q", got, "test-model")
	}

	res, err := proofreader.ProofreadScan(t.Context(), []domain.ProofreadPage{
		{Number: 1, Lines: []domain.ProofreadLine{{Number: 4, Text: "wrods"}}},
	})
	if err != nil {
		t.Fatalf("ProofreadScan failed: %v", err)
	}
	if len(res.Corrections) != 1 || res.Corrections[0].Number != 4 || res.Corrections[0].Text != "fixed: wrods" {
		t.Errorf("corrections %+v", res.Corrections)
	}
}

// The settings of a kind of text reach the run: whether it was named, whether
// it is automatic, and what opens the proofreader.
func TestTheProofreadingOfAKindOfTextCarriesTheSettingsOfItsProfile(t *testing.T) {
	cfg := container.Config{
		ScanProofreading:       container.Proofread{Profile: "careful", IsAutomatic: true},
		TranscriptProofreading: container.Proofread{},
	}

	scan := cfg.GetScanProofreading()
	if !scan.IsNamed || !scan.IsAutomatic {
		t.Errorf("scan proofreading %+v", scan)
	}
	if scan.By == nil {
		t.Fatal("a named profile opens nothing")
	}
	if _, why := scan.By(); !errors.Is(why, port.ErrUnavailable) {
		t.Errorf("a profile with no plugin opened with %v", why)
	}

	speech := cfg.GetSpeechProofreading()
	if speech.IsNamed || speech.IsAutomatic {
		t.Errorf("speech proofreading %+v", speech)
	}
	if by, why := speech.By(); by != nil || why != nil {
		t.Errorf("an unnamed profile opened %v, %v", by, why)
	}
}
