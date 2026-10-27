package link

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Provider makes the plugin's texts on the host's request: the text published at a link's address.
type Provider struct {
	links *Links
}

// NewProvider is the texts of the links.
func NewProvider(links *Links) Provider { return Provider{links: links} }

var _ host.TextProvider = Provider{}

// CreateText fetches the text of the link in the background. The plugin reads links and no
// version, so a request naming a source is not one it takes.
func (p Provider) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("%s: %w", req.Source, host.ErrInvalidVersion)
	}
	link, err := readLink(ctx, p.links.host.Vault.With(req.Vault), req.Path)
	if err != nil {
		return err
	}
	p.links.StartFetchText(link, req.IsUserStarted)
	return nil
}
