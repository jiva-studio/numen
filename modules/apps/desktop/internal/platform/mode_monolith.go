//go:build plugins_monolith

package platform

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	epubservice "github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

// BuildMode is the desktop build variant.
const BuildMode = "monolith"

// IsMonolith reports whether plugins are compiled directly into the binary.
func IsMonolith() bool {
	return true
}

func init() {
	wireMonolithHandlers()
}

func wireMonolithHandlers() {
	container.RegisterDocumentServiceHandler(pdfservice.NewDocumentServer())
	container.RegisterBookServiceHandler(epubservice.NewBookServer())
	container.RegisterRecognitionServiceHandler(&monolithRecognitionServer{})
	container.RegisterTranscriptionServiceHandler(&monolithTranscriptionServer{})
	container.RegisterEmbeddingServiceHandler(&monolithEmbeddingServer{})
}

type monolithRecognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
}

func (s *monolithRecognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetRecognitionModelRequest],
) (*connect.Response[numenv1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
		Layout:     "monolith-layout",
		Recogniser: "monolith-recog",
		Dpi:        150,
		From:       "builtin",
	}), nil
}

type monolithTranscriptionServer struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
}

func (s *monolithTranscriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetTranscriptionModelRequest],
) (*connect.Response[numenv1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetTranscriptionModelResponse{
		Model:     "monolith-transcription",
		Segmenter: "builtin",
		Cutting:   "",
		From:      "builtin",
	}), nil
}

type monolithEmbeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
}

func (s *monolithEmbeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetEmbeddingModelRequest],
) (*connect.Response[numenv1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        "bge-m3",
		Dimensions:  1024,
		MaxTokens:   512,
		Pooling:     "mean",
		From:        "builtin",
		Concurrency: 4,
	}), nil
}

func (s *monolithEmbeddingServer) EmbedText(
	_ context.Context,
	_ *connect.Request[numenv1.EmbedTextRequest],
) (*connect.Response[numenv1.EmbedTextResponse], error) {
	return connect.NewResponse(&numenv1.EmbedTextResponse{
		Vector: make([]float32, 1024),
	}), nil
}

func (s *monolithEmbeddingServer) EmbedBatch(
	_ context.Context,
	req *connect.Request[numenv1.EmbedBatchRequest],
) (*connect.Response[numenv1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
	vectors := make([]*numenv1.FloatVector, len(texts))
	for i := range vectors {
		vectors[i] = &numenv1.FloatVector{
			Values: make([]float32, 1024),
		}
	}
	return connect.NewResponse(&numenv1.EmbedBatchResponse{
		Vectors: vectors,
	}), nil
}
