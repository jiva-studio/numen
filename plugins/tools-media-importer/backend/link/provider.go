package link

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Provider is the plugin's texts as the host reads them: the text published at a link's address,
// one page of it.
type Provider struct {
	links *Links
}

// NewProvider is the texts of the links.
func NewProvider(links *Links) Provider { return Provider{links: links} }

var _ host.TextProvider = Provider{}

func (p Provider) GetText(ctx context.Context, req host.TextRequest) (host.TextPage, error) {
	if req.Cursor != "" {
		return host.TextPage{}, fmt.Errorf("cursor %q: %w", req.Cursor, host.ErrNotExist)
	}
	parts, _, err := texts{store: p.links.host.Store.With(req.Vault)}.readParts(ctx, req.Key)
	if err != nil {
		return host.TextPage{}, err
	}
	return host.TextPage{Parts: parts}, nil
}

// CreateText fetches the text of the link in the background. The plugin reads links and no
// version, so a request naming an input is not one it takes.
func (p Provider) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Input != "" {
		return fmt.Errorf("%s: %w", req.Input, host.ErrInvalidVersion)
	}
	link, err := readLink(ctx, p.links.host.Vault.With(req.Vault), req.Path)
	if err != nil {
		return err
	}
	p.links.StartFetchText(link, req.IsUserStarted)
	return nil
}

func (p Provider) DeleteText(ctx context.Context, vault domain.VaultID, _, key string) error {
	return texts{store: p.links.host.Store.With(vault)}.deleteText(ctx, key)
}
