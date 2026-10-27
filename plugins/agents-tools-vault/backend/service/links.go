package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type LinkListInput struct {
	Path string `json:"path" jsonschema:"path of the note to inspect for links and backlinks"`
}

type ResolvedLinkItem struct {
	From        string `json:"from" jsonschema:"source note path"`
	To          string `json:"to,omitempty" jsonschema:"resolved target note path"`
	Address     string `json:"address" jsonschema:"target link address as written"`
	Role        string `json:"role,omitempty" jsonschema:"link role (e.g. parent, child, jump, ref)"`
	Label       string `json:"label,omitempty" jsonschema:"link label"`
	IsAmbiguous bool   `json:"is_ambiguous,omitempty" jsonschema:"true if link target matches multiple notes"`
}

type LinkListOutput struct {
	Path      string             `json:"path"`
	Links     []ResolvedLinkItem `json:"links" jsonschema:"outgoing links from this note"`
	Backlinks []ResolvedLinkItem `json:"backlinks" jsonschema:"incoming backlinks pointing to this note"`
}

func (t *Tools) addNoteNeighbourhood(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "note_neighbourhood",
		Title:       "Note Neighbourhood",
		Description: "List outgoing wikilinks and incoming backlinks connected to a note.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listLinks))
}

func (t *Tools) addLinkList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "link_list",
		Title:       "List Links",
		Description: "List outgoing wikilinks and incoming backlinks connected to a note.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listLinks))
}

func (t *Tools) listLinks(ctx context.Context, in LinkListInput) (LinkListOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return LinkListOutput{}, errors.New("path required")
	}
	path := t.resolveNotePath(ctx, in.Path)

	outgoing, err := t.vault.GetLinks(ctx, path)
	if err != nil {
		return LinkListOutput{}, err
	}

	incoming, err := t.vault.GetBacklinks(ctx, path)
	if err != nil {
		return LinkListOutput{}, err
	}

	return LinkListOutput{
		Path:      path,
		Links:     convertResolvedLinks(outgoing),
		Backlinks: convertResolvedLinks(incoming),
	}, nil
}

func convertResolvedLinks(links []domain.ResolvedLink) []ResolvedLinkItem {
	items := make([]ResolvedLinkItem, 0, len(links))
	for _, l := range links {
		addr := l.Target.GetWritten()
		if addr == "" {
			addr = l.Target.String()
		}
		items = append(items, ResolvedLinkItem{
			From:        l.From,
			To:          l.To,
			Address:     addr,
			Role:        string(l.Role),
			Label:       l.Label,
			IsAmbiguous: l.IsAmbiguous,
		})
	}
	return items
}
