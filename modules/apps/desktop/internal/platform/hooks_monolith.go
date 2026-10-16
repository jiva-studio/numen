//go:build plugins_monolith

package platform

import (
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// RegisterDocumentHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentHandler(h numenv1connect.DocumentServiceHandler) {
	SetMonolith(true)
	container.RegisterDocumentServiceHandler(h)
}

// RegisterBookHandler sets the in-process BookServiceHandler.
func RegisterBookHandler(h numenv1connect.BookServiceHandler) {
	SetMonolith(true)
	container.RegisterBookServiceHandler(h)
}

// RegisterRecognitionHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionHandler(h numenv1connect.RecognitionServiceHandler) {
	SetMonolith(true)
	container.RegisterRecognitionServiceHandler(h)
}

// RegisterTranscriptionHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionHandler(h numenv1connect.TranscriptionServiceHandler) {
	SetMonolith(true)
	container.RegisterTranscriptionServiceHandler(h)
}

// RegisterEmbeddingHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingHandler(h numenv1connect.EmbeddingServiceHandler) {
	SetMonolith(true)
	container.RegisterEmbeddingServiceHandler(h)
}
