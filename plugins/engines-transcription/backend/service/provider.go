package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// provider is the plugin's texts as the host reads them: what the model heard in a recording, and a
// person's edits of a recording's text, each one page of timed parts.
type provider struct {
	stores      host.Store
	transcriber *Transcriber
}

var _ host.TextProvider = provider{}

func (p provider) GetText(ctx context.Context, req host.TextRequest) (host.TextPage, error) {
	if req.Cursor != "" {
		return host.TextPage{}, fmt.Errorf("cursor %q: %w", req.Cursor, host.ErrNotExist)
	}
	cues, err := transcripts{store: p.stores.With(req.Vault)}.readCues(ctx, req.Key)
	if err != nil {
		return host.TextPage{}, err
	}
	return host.TextPage{Parts: writeParts(cues)}, nil
}

// CreateText transcribes the recording. The plugin reads recordings and no version, so a request
// naming an input is not one it takes.
func (p provider) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Input != "" {
		return fmt.Errorf("%s: %w", req.Input, host.ErrInvalidVersion)
	}
	return p.transcriber.Start(ctx, req.Vault, req.Path)
}

func (p provider) DeleteText(ctx context.Context, vault domain.VaultID, _, key string) error {
	return transcripts{store: p.stores.With(vault)}.remove(ctx, key)
}
