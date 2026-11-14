// Package linkswire calls the links use case with the messages of the link tools.
package linkswire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/links"
)

// ListLinks answers link_list.
func ListLinks(ctx context.Context, u *links.UseCase, in api.LinkListInput) (api.LinkListOutput, error) {
	list, err := u.List(ctx, in.Path)
	if err != nil {
		return api.LinkListOutput{}, err
	}
	return api.LinkListOutput{Path: list.Path, Links: newItems(list.Links), Backlinks: newItems(list.Backlinks)}, nil
}

// AddLinks answers link_add.
func AddLinks(ctx context.Context, u *links.UseCase, in api.LinkAddInput) (api.LinkAddOutput, error) {
	changes := make([]links.LinkChange, len(in.Links))
	for i, link := range in.Links {
		changes[i] = links.LinkChange{
			From: link.From, Fingerprint: link.Fingerprint, Address: link.Address,
			Role: link.Role, Type: link.Type, Label: link.Label, Note: link.Note,
		}
	}
	results, err := u.Add(ctx, changes)
	if err != nil {
		return api.LinkAddOutput{}, err
	}
	out := api.LinkAddOutput{Results: make([]api.LinkAddItem, len(results))}
	for i, result := range results {
		out.Results[i] = api.LinkAddItem{From: in.Links[i].From, Address: in.Links[i].Address, Fingerprint: result.Fingerprint, Error: ""}
		if result.Err != nil {
			out.Results[i].Error = result.Err.Error()
		}
	}
	return out, nil
}

// UpdateLinks answers link_update.
func UpdateLinks(ctx context.Context, u *links.UseCase, in api.LinkUpdateInput) (api.LinkEditOutput, error) {
	edit, err := u.Update(ctx, links.LinkChange{
		From: in.From, Fingerprint: in.Fingerprint, Address: in.Address,
		Role: in.Role, Type: in.Type, Label: in.Label, Note: in.Note,
	})
	if err != nil {
		return api.LinkEditOutput{}, err
	}
	return api.LinkEditOutput{Path: edit.Path, Fingerprint: edit.Fingerprint}, nil
}

// RemoveLinks answers link_remove.
func RemoveLinks(ctx context.Context, u *links.UseCase, in api.LinkRemoveInput) (api.LinkEditOutput, error) {
	edit, err := u.Remove(ctx, links.LinkChange{
		From: in.From, Fingerprint: in.Fingerprint, Address: in.Address,
		Role: in.Role, Type: "", Label: "", Note: "",
	})
	if err != nil {
		return api.LinkEditOutput{}, err
	}
	return api.LinkEditOutput{Path: edit.Path, Fingerprint: edit.Fingerprint}, nil
}

func newItems(found []domain.ResolvedLink) []api.ResolvedLinkItem {
	items := make([]api.ResolvedLinkItem, 0, len(found))
	for _, link := range found {
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
