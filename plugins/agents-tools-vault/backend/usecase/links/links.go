// Package links lists the links of notes and writes links into their frontmatter.
package links

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notepath"
)

// UseCase works the links of the notes of one vault.
type UseCase struct {
	lookup   ports.LinkLookup
	editor   ports.LinkEditor
	resolver *notepath.Resolver
}

// New creates a UseCase over the links of one vault; resolver finds the note an agent names.
func New(lookup ports.LinkLookup, editor ports.LinkEditor, resolver *notepath.Resolver) *UseCase {
	return &UseCase{lookup: lookup, editor: editor, resolver: resolver}
}

// List returns the outgoing links and the backlinks of a note.
func (u *UseCase) List(ctx context.Context, in api.LinkListInput) (api.LinkListOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.LinkListOutput{}, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, in.Path)
	outgoing, err := u.lookup.GetLinks(ctx, path)
	if err != nil {
		return api.LinkListOutput{}, err
	}
	incoming, err := u.lookup.GetBacklinks(ctx, path)
	if err != nil {
		return api.LinkListOutput{}, err
	}
	return api.LinkListOutput{Path: path, Links: newItems(outgoing), Backlinks: newItems(incoming)}, nil
}

func newItems(links []domain.ResolvedLink) []api.ResolvedLinkItem {
	items := make([]api.ResolvedLinkItem, 0, len(links))
	for _, link := range links {
		address := link.Target.GetWritten()
		if address == "" {
			address = link.Target.String()
		}
		items = append(items, api.ResolvedLinkItem{
			From:        link.From,
			To:          link.To,
			Address:     address,
			Role:        string(link.Role),
			Label:       link.Label,
			IsAmbiguous: link.IsAmbiguous,
		})
	}
	return items
}
