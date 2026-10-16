//go:build plugins_monolith

package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// RegisterDocumentServiceHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentServiceHandler(h numenv1connect.DocumentServiceHandler) {
	plugin.RegisterDocumentServiceHandler(h)
}

// RegisterRecognitionServiceHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionServiceHandler(h numenv1connect.RecognitionServiceHandler) {
	plugin.RegisterRecognitionServiceHandler(h)
}

// RegisterBookServiceHandler sets the in-process BookServiceHandler.
func RegisterBookServiceHandler(h numenv1connect.BookServiceHandler) {
	plugin.RegisterBookServiceHandler(h)
}

// RegisterTranscriptionServiceHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionServiceHandler(h numenv1connect.TranscriptionServiceHandler) {
	plugin.RegisterTranscriptionServiceHandler(h)
}

// RegisterEmbeddingServiceHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingServiceHandler(h numenv1connect.EmbeddingServiceHandler) {
	plugin.RegisterEmbeddingServiceHandler(h)
}
