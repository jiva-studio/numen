package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// provider creates the plugin's texts on the host's request: the model's transcription of a recording.
type provider struct {
	transcriber *Transcriber
}

var _ host.TextProvider = provider{transcriber: nil}

// CreateText transcribes the recording. The plugin reads recordings and no version, so a request
// naming a source is not one it takes.
func (p provider) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("%s: %w", req.Source, host.ErrInvalidVersion)
	}
	return p.transcriber.Start(ctx, req.Vault, req.Path)
}
