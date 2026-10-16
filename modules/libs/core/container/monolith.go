//go:build plugins_monolith

package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
)

// RegisterDocumentServiceHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentServiceHandler(h any) {
	plugin.RegisterDocumentServiceHandler(h)
}

// RegisterRecognitionServiceHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionServiceHandler(h any) {
	plugin.RegisterRecognitionServiceHandler(h)
}

// RegisterBookServiceHandler sets the in-process BookServiceHandler.
func RegisterBookServiceHandler(h any) {
	plugin.RegisterBookServiceHandler(h)
}

// RegisterTranscriptionServiceHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionServiceHandler(h any) {
	plugin.RegisterTranscriptionServiceHandler(h)
}

// RegisterEmbeddingServiceHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingServiceHandler(h any) {
	plugin.RegisterEmbeddingServiceHandler(h)
}

// RegisterDownloadServiceHandler sets the in-process DownloadServiceHandler.
func RegisterDownloadServiceHandler(h any) {
	plugin.RegisterDownloadServiceHandler(h)
}

// RegisterAgentServiceHandler sets the in-process AgentServiceHandler.
func RegisterAgentServiceHandler(h any) {
	plugin.RegisterAgentServiceHandler(h)
}

// RegisterProofreadingServiceHandler sets the in-process ProofreadingServiceHandler.
func RegisterProofreadingServiceHandler(h any) {
	plugin.RegisterProofreadingServiceHandler(h)
}
