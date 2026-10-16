package app

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// RegisterDocumentHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentHandler(h numenv1connect.DocumentServiceHandler) {
	platform.RegisterDocumentHandler(h)
}

// RegisterBookHandler sets the in-process BookServiceHandler.
func RegisterBookHandler(h numenv1connect.BookServiceHandler) {
	platform.RegisterBookHandler(h)
}

// RegisterRecognitionHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionHandler(h numenv1connect.RecognitionServiceHandler) {
	platform.RegisterRecognitionHandler(h)
}

// RegisterTranscriptionHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionHandler(h numenv1connect.TranscriptionServiceHandler) {
	platform.RegisterTranscriptionHandler(h)
}

// RegisterEmbeddingHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingHandler(h numenv1connect.EmbeddingServiceHandler) {
	platform.RegisterEmbeddingHandler(h)
}
