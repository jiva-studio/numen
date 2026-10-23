package recogniserbridge

import (
	"context"
	"fmt"
	"image"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

type mockRecognitionServer struct{}

func (s *mockRecognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[v1.GetRecognitionModelRequest],
) (*connect.Response[v1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&v1.GetRecognitionModelResponse{
		Layout:     "pp-layout-v2",
		Recogniser: "pp-ocrv6-test",
		Dpi:        300,
		From:       "test-fixture",
	}), nil
}

func (s *mockRecognitionServer) RecognisePage(
	_ context.Context,
	req *connect.Request[v1.RecognisePageRequest],
) (*connect.Response[v1.RecognisePageResponse], error) {
	if len(req.Msg.GetImageData()) == 0 {
		return connect.NewResponse(&v1.RecognisePageResponse{
			Blocks: nil,
		}), nil
	}

	return connect.NewResponse(&v1.RecognisePageResponse{
		Blocks: []*v1.OcrBlock{
			{
				Label:     "heading",
				Text:      "Test OCR Heading",
				IsHeading: true,
				Depth:     1,
				Boxes: []*v1.OcrBox{
					{
						MinX: 10,
						MinY: 20,
						MaxX: 100,
						MaxY: 40,
						Span: &v1.Span{
							From: 0,
							To:   16,
						},
					},
				},
			},
		},
	}), nil
}

func TestRecogniserBridge(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewRecognitionServiceHandler(&mockRecognitionServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	host := plugin.NewHost("desktop.editor", tmp)
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

	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	bridge, err := Open(t.Context(), host)
	if err != nil {
		t.Fatalf("OpenRecogniserBridge failed: %v", err)
	}
	defer bridge.Close()

	model := bridge.GetRecognitionModel()
	if model.Layout != "pp-layout-v2" || model.Recogniser != "pp-ocrv6-test" {
		t.Fatalf("unexpected model: %+v", model)
	}

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blocks, err := bridge.Recognise(t.Context(), img)
	if err != nil {
		t.Fatalf("Recognise failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Text != "Test OCR Heading" || !blocks[0].IsHeading {
		t.Fatalf("unexpected block content: %+v", blocks[0])
	}
	if len(blocks[0].Boxes) != 1 {
		t.Fatalf("expected 1 box, got %d", len(blocks[0].Boxes))
	}
}

func TestRecogniserBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := Open(t.Context(), host)
	if err == nil {
		t.Fatal("expected error when no plugin registered, got nil")
	}
}

type streamAuthServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	gotAuth string
}

func (s *streamAuthServer) AskAgent(
	_ context.Context,
	req *connect.Request[v1.AskAgentRequest],
	_ *connect.ServerStream[v1.AskAgentResponse],
) error {
	s.gotAuth = req.Header().Get("Authorization")
	return nil
}

func TestAuthInterceptorAttachesTokenToStreamingRequests(t *testing.T) {
	serverImpl := &streamAuthServer{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(serverImpl))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(
		server.Client(),
		server.URL,
		connect.WithInterceptors(transport.AuthInterceptor("stream-secret-token")),
	)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{}))
	if err != nil {
		t.Fatalf("AskAgent: %v", err)
	}
	defer stream.Close()

	for stream.Receive() {
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if serverImpl.gotAuth != "Bearer stream-secret-token" {
		t.Fatalf("received Authorization = %q, want Bearer stream-secret-token", serverImpl.gotAuth)
	}
}
