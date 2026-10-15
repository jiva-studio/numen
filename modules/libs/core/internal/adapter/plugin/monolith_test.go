//go:build plugins_monolith

package plugin_test

import (
	"context"
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

func TestMonolithInProcessDispatch(t *testing.T) {
	plugin.ClearMonolithHandlers()
	defer plugin.ClearMonolithHandlers()

	plugin.RegisterDocumentServiceHandler(&mockDocumentServer{})
	plugin.RegisterRecognitionServiceHandler(&mockRecognitionServer{})
	plugin.RegisterBookServiceHandler(&mockBookServer{})
	plugin.RegisterTranscriptionServiceHandler(&mockTranscriptionServer{})
	plugin.RegisterEmbeddingServiceHandler(&mockEmbeddingServer{})

	host := plugin.NewHost("desktop.editor")
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

	docClient := numenv1connect.NewDocumentServiceClient(endpointDoc.HTTPClient(), endpointDoc.BaseURL())
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
	recogClient := numenv1connect.NewRecognitionServiceClient(endpointOcr.HTTPClient(), endpointOcr.BaseURL())
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
	bookClient := numenv1connect.NewBookServiceClient(endpointBook.HTTPClient(), endpointBook.BaseURL())
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
	transClient := numenv1connect.NewTranscriptionServiceClient(endpointTrans.HTTPClient(), endpointTrans.BaseURL())
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
	embedClient := numenv1connect.NewEmbeddingServiceClient(endpointEmbed.HTTPClient(), endpointEmbed.BaseURL())
	embedResp, err := embedClient.GetEmbeddingModel(ctx, connect.NewRequest(&v1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("embedClient.GetEmbeddingModel failed: %v", err)
	}
	if embedResp.Msg.GetName() != "bge-m3-inprocess" || embedResp.Msg.GetDimensions() != 1024 {
		t.Fatalf("unexpected embedding model: %+v", embedResp.Msg)
	}

	// 6. Teardown
	if err := host.TeardownAll(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}
