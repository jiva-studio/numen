//go:build !plugins_monolith

package platform

import (
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// RegisterDocumentHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentHandler(_ numenv1connect.DocumentServiceHandler) {}

// RegisterBookHandler sets the in-process BookServiceHandler.
func RegisterBookHandler(_ numenv1connect.BookServiceHandler) {}

// RegisterRecognitionHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionHandler(_ numenv1connect.RecognitionServiceHandler) {}

// RegisterTranscriptionHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionHandler(_ numenv1connect.TranscriptionServiceHandler) {}

// RegisterEmbeddingHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingHandler(_ numenv1connect.EmbeddingServiceHandler) {}
