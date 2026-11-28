package vault

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
)

func (e *lookupsCalls) ReadFrontmatter(
	ctx context.Context, req *connect.Request[v1.ReadFrontmatterRequest],
) (*connect.Response[v1.ReadFrontmatterResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil || vault.Frontmatter == nil {
		return connect.NewResponse(&v1.ReadFrontmatterResponse{Error: noVault}), nil
	}
	page, err := vault.Frontmatter.ReadFrontmatter(ctx, frontmatter.Query{
		Paths: req.Msg.GetPaths(), Keys: req.Msg.GetKeys(), After: req.Msg.GetPageToken(), Limit: int(req.Msg.GetPageSize()),
	})
	if errors.Is(err, frontmatter.ErrInvalidQuery) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ReadFrontmatterResponse{Error: code}), err
	}
	notes := make([]*v1.NoteFrontmatter, 0, len(page.Notes))
	for _, one := range page.Notes {
		values, err := json.Marshal(one.Values)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		notes = append(notes, &v1.NoteFrontmatter{Path: one.Path, Values: values, SourceId: int64(one.SourceID)})
	}
	return connect.NewResponse(&v1.ReadFrontmatterResponse{
		Notes: notes, NextPageToken: page.After, Cursor: newCursorMessage(page.Cursor),
	}), nil
}
