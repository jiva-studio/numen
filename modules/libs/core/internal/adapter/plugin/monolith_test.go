//go:build plugins_monolith

package plugin_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
)

type mockDocumentServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler
}

func (s *mockDocumentServer) ExtractText(
	_ context.Context,
	_ *connect.Request[v1.ExtractTextRequest],
) (*connect.Response[v1.ExtractTextResponse], error) {
	return connect.NewResponse(&v1.ExtractTextResponse{
		Text: "in-process document text",
	}), nil
}

type mockRecognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
}

func (s *mockRecognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[v1.GetRecognitionModelRequest],
) (*connect.Response[v1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&v1.GetRecognitionModelResponse{
		Layout:     "monolith-layout",
		Recogniser: "monolith-recog",
	}), nil
}

type mockBookServer struct {
	numenv1connect.UnimplementedBookServiceHandler
}

func (s *mockBookServer) GetBook(
	_ context.Context,
	_ *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	return connect.NewResponse(&v1.GetBookResponse{
		Title: "In-Process Book",
	}), nil
}

type mockTranscriptionServer struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
}

func (s *mockTranscriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{
		Model: "monolith-parakeet",
	}), nil
}

type mockEmbeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
}

func (s *mockEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Name:       "bge-m3-inprocess",
		Dimensions: 1024,
	}), nil
}

type mockDownloadServer struct {
	numenv1connect.UnimplementedDownloadServiceHandler
}

func (s *mockDownloadServer) GetDownloadModel(
	_ context.Context,
	_ *connect.Request[v1.GetDownloadModelRequest],
) (*connect.Response[v1.GetDownloadModelResponse], error) {
	return connect.NewResponse(&v1.GetDownloadModelResponse{
		Tool: "yt-dlp",
	}), nil
}

type mockAgentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
}

func (s *mockAgentServer) GetAgentTools(
	_ context.Context,
	_ *connect.Request[v1.GetAgentToolsRequest],
) (*connect.Response[v1.GetAgentToolsResponse], error) {
	return connect.NewResponse(&v1.GetAgentToolsResponse{
		SystemPromptExtension: "monolith agent prompt",
	}), nil
}

type mockProofreadingServer struct {
	numenv1connect.UnimplementedProofreadingServiceHandler
}

func (s *mockProofreadingServer) GetProofreaderInfo(
	_ context.Context,
	_ *connect.Request[v1.GetProofreaderInfoRequest],
) (*connect.Response[v1.GetProofreaderInfoResponse], error) {
	return connect.NewResponse(&v1.GetProofreaderInfoResponse{
		Name: "monolith-proofreader",
	}), nil
}

func TestMonolithInProcessDispatch(t *testing.T) {
	plugin.ClearMonolithHandlers()
	defer plugin.ClearMonolithHandlers()
	t.Setenv(plugin.EnvONNXRuntimeLib, "")
	t.Setenv(plugin.EnvONNXProvider, "")

	plugin.RegisterDocumentServiceHandler(&mockDocumentServer{})
	plugin.RegisterRecognitionServiceHandler(&mockRecognitionServer{})
	plugin.RegisterBookServiceHandler(&mockBookServer{})
	plugin.RegisterTranscriptionServiceHandler(&mockTranscriptionServer{})
	plugin.RegisterEmbeddingServiceHandler(&mockEmbeddingServer{})
	plugin.RegisterDownloadServiceHandler(&mockDownloadServer{})
	plugin.RegisterAgentServiceHandler(&mockAgentServer{})
	plugin.RegisterProofreadingServiceHandler(&mockProofreadingServer{})

	host := plugin.NewHost("desktop.editor")
	host.SetRuntimeProvisioner(fixedProvisioner{library: "/models/libonnxruntime.so", provider: "cpu"})
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := t.Context()

	// 1. DocumentService dispatch
	endpointDoc, err := host.EnsureActive(ctx, "md.numen.pdf")
	if err != nil {
		t.Fatalf("EnsureActive(pdf) failed: %v", err)
	}
	if endpointDoc.Port != 0 {
		t.Fatalf("expected port 0 for in-process monolith, got %d", endpointDoc.Port)
	}
	if endpointDoc.Transport == nil {
		t.Fatal("expected non-nil in-process transport")
	}

	docClient := numenv1connect.NewDocumentServiceClient(plugin.EndpointClient(endpointDoc), endpointDoc.BaseURL())
	docResp, err := docClient.ExtractText(ctx, connect.NewRequest(&v1.ExtractTextRequest{}))
	if err != nil {
		t.Fatalf("docClient.ExtractText failed: %v", err)
	}
	if docResp.Msg.GetText() != "in-process document text" {
		t.Fatalf("unexpected document text: %q", docResp.Msg.GetText())
	}

	// 2. RecognitionService dispatch
	endpointOcr, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive(ocr) failed: %v", err)
	}
	recogClient := numenv1connect.NewRecognitionServiceClient(plugin.EndpointClient(endpointOcr), endpointOcr.BaseURL())
	recogResp, err := recogClient.GetRecognitionModel(ctx, connect.NewRequest(&v1.GetRecognitionModelRequest{}))
	if err != nil {
		t.Fatalf("recogClient.GetRecognitionModel failed: %v", err)
	}
	if recogResp.Msg.GetLayout() != "monolith-layout" {
		t.Fatalf("unexpected layout: %q", recogResp.Msg.GetLayout())
	}

	// 3. BookService dispatch
	endpointBook, err := host.EnsureActive(ctx, "md.numen.epub")
	if err != nil {
		t.Fatalf("EnsureActive(epub) failed: %v", err)
	}
	bookClient := numenv1connect.NewBookServiceClient(plugin.EndpointClient(endpointBook), endpointBook.BaseURL())
	bookResp, err := bookClient.GetBook(ctx, connect.NewRequest(&v1.GetBookRequest{}))
	if err != nil {
		t.Fatalf("bookClient.GetBook failed: %v", err)
	}
	if bookResp.Msg.GetTitle() != "In-Process Book" {
		t.Fatalf("unexpected book title: %q", bookResp.Msg.GetTitle())
	}

	// 4. TranscriptionService dispatch
	endpointTrans, err := host.EnsureActive(ctx, "md.numen.transcription")
	if err != nil {
		t.Fatalf("EnsureActive(transcription) failed: %v", err)
	}
	transClient := numenv1connect.NewTranscriptionServiceClient(plugin.EndpointClient(endpointTrans), endpointTrans.BaseURL())
	transResp, err := transClient.GetTranscriptionModel(ctx, connect.NewRequest(&v1.GetTranscriptionModelRequest{}))
	if err != nil {
		t.Fatalf("transClient.GetTranscriptionModel failed: %v", err)
	}
	if transResp.Msg.GetModel() != "monolith-parakeet" {
		t.Fatalf("unexpected transcription model: %q", transResp.Msg.GetModel())
	}

	// 5. EmbeddingService dispatch
	endpointEmbed, err := host.EnsureActive(ctx, "md.numen.embeddings")
	if err != nil {
		t.Fatalf("EnsureActive(embeddings) failed: %v", err)
	}
	embedClient := numenv1connect.NewEmbeddingServiceClient(plugin.EndpointClient(endpointEmbed), endpointEmbed.BaseURL())
	embedResp, err := embedClient.GetEmbeddingModel(ctx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("embedClient.GetEmbeddingModel failed: %v", err)
	}
	if embedResp.Msg.GetName() != "bge-m3-inprocess" || embedResp.Msg.GetDimensions() != 1024 {
		t.Fatalf("unexpected embedding model: %+v", embedResp.Msg)
	}

	// 6. DownloadService dispatch
	endpointDownloader, err := host.EnsureActive(ctx, "md.numen.downloader")
	if err != nil {
		t.Fatalf("EnsureActive(downloader) failed: %v", err)
	}
	downloadClient := numenv1connect.NewDownloadServiceClient(plugin.EndpointClient(endpointDownloader), endpointDownloader.BaseURL())
	downloadResp, err := downloadClient.GetDownloadModel(ctx, connect.NewRequest(&v1.GetDownloadModelRequest{}))
	if err != nil {
		t.Fatalf("downloadClient.GetDownloadModel failed: %v", err)
	}
	if downloadResp.Msg.GetTool() != "yt-dlp" {
		t.Fatalf("unexpected download model tool: %q", downloadResp.Msg.GetTool())
	}

	// 7. AgentService dispatch
	endpointAgent, err := host.EnsureActive(ctx, "md.numen.agent")
	if err != nil {
		t.Fatalf("EnsureActive(agent) failed: %v", err)
	}
	agentClient := numenv1connect.NewAgentServiceClient(plugin.EndpointClient(endpointAgent), endpointAgent.BaseURL())
	agentResp, err := agentClient.GetAgentTools(ctx, connect.NewRequest(&v1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("agentClient.GetAgentTools failed: %v", err)
	}
	if agentResp.Msg.GetSystemPromptExtension() != "monolith agent prompt" {
		t.Fatalf("unexpected system prompt extension: %q", agentResp.Msg.GetSystemPromptExtension())
	}

	// 8. ProofreadingService dispatch
	endpointProof, err := host.EnsureActive(ctx, "md.numen.proofreading")
	if err != nil {
		t.Fatalf("EnsureActive(proofreading) failed: %v", err)
	}
	proofClient := numenv1connect.NewProofreadingServiceClient(plugin.EndpointClient(endpointProof), endpointProof.BaseURL())
	proofResp, err := proofClient.GetProofreaderInfo(ctx, connect.NewRequest(&v1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatalf("proofClient.GetProofreaderInfo failed: %v", err)
	}
	if proofResp.Msg.GetName() != "monolith-proofreader" {
		t.Fatalf("unexpected proofreader name: %q", proofResp.Msg.GetName())
	}

	// 9. Teardown
	if err := host.TeardownAll(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

type fixedProvisioner struct {
	library, provider string
	err               error
}

func (f fixedProvisioner) Provision(context.Context) (string, string, error) {
	return f.library, f.provider, f.err
}

// An in-process plugin that declares the runtime finds the same variables a
// launched process is given, in the process that is its own.
func TestMonolithInProcessPluginIsGivenTheRuntime(t *testing.T) {
	plugin.ClearMonolithHandlers()
	defer plugin.ClearMonolithHandlers()
	t.Setenv(plugin.EnvONNXRuntimeLib, "")
	t.Setenv(plugin.EnvONNXProvider, "")

	host := plugin.NewHost("desktop.editor")
	host.SetRuntimeProvisioner(fixedProvisioner{library: "/models/libonnxruntime.so", provider: "cuda"})
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive(ocr) failed: %v", err)
	}
	if got := os.Getenv(plugin.EnvONNXRuntimeLib); got != "/models/libonnxruntime.so" {
		t.Errorf("%s = %q", plugin.EnvONNXRuntimeLib, got)
	}
	if got := os.Getenv(plugin.EnvONNXProvider); got != "cuda" {
		t.Errorf("%s = %q", plugin.EnvONNXProvider, got)
	}
}

// An in-process plugin whose runtime cannot be provisioned is not activated.
func TestMonolithInProcessPluginIsNotActivatedWithoutItsRuntime(t *testing.T) {
	plugin.ClearMonolithHandlers()
	defer plugin.ClearMonolithHandlers()
	t.Setenv(plugin.EnvONNXRuntimeLib, "")

	failure := errors.New("the archive is not the pinned one")
	host := plugin.NewHost("desktop.editor")
	host.SetRuntimeProvisioner(fixedProvisioner{err: failure})
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	_, err := host.EnsureActive(t.Context(), "md.numen.transcription")
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "md.numen.transcription") {
		t.Fatalf("EnsureActive() = %v, want the failure with the plugin named", err)
	}
	if got := os.Getenv(plugin.EnvONNXRuntimeLib); got != "" {
		t.Errorf("%s = %q after a failed provisioning", plugin.EnvONNXRuntimeLib, got)
	}
}
