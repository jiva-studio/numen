// Command main is the monolithic entry point: desktop application with plugins wired in-process.
package main

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/apps/desktop/app"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	epubservice "github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

func main() {
	wireMonolithHandlers()
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

func wireMonolithHandlers() {
	app.RegisterDocumentHandler(pdfservice.NewDocumentServer())
	app.RegisterBookHandler(epubservice.NewBookServer())
	app.RegisterRecognitionHandler(&monolithRecognitionServer{})
	app.RegisterTranscriptionHandler(&monolithTranscriptionServer{})
	app.RegisterEmbeddingHandler(&monolithEmbeddingServer{})
}

type monolithRecognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
}

func (s *monolithRecognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[v1.GetRecognitionModelRequest],
) (*connect.Response[v1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&v1.GetRecognitionModelResponse{
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
	_ *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{
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
	_ *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
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
	_ *connect.Request[v1.EmbedTextRequest],
) (*connect.Response[v1.EmbedTextResponse], error) {
	return connect.NewResponse(&v1.EmbedTextResponse{
		Vector: make([]float32, 1024),
	}), nil
}

func (s *monolithEmbeddingServer) EmbedBatch(
	_ context.Context,
	req *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
	vectors := make([]*v1.FloatVector, len(texts))
	for i := range vectors {
		vectors[i] = &v1.FloatVector{
			Values: make([]float32, 1024),
		}
	}
	return connect.NewResponse(&v1.EmbedBatchResponse{
		Vectors: vectors,
	}), nil
}
