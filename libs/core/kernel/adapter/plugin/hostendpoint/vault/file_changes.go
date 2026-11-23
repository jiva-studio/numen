package vault

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
)

func (e *lookupsCalls) ListFileChanges(
	ctx context.Context, req *connect.Request[v1.ListFileChangesRequest],
) (*connect.Response[v1.ListFileChangesResponse], error) {
	pluginID, vault, err := e.beginAs(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil || vault.FileChanges == nil {
		return connect.NewResponse(&v1.ListFileChangesResponse{Error: noVault}), nil
	}
	since := req.Msg.GetSince()
	page, err := vault.FileChanges.ListFileChanges(ctx, filechange.Query{
		Since:  filechange.Cursor{FeedID: since.GetFeedId(), Seq: since.GetSeq()},
		Events: req.Msg.GetEvents(),
		Limit:  int(req.Msg.GetLimit()),
	})
	if errors.Is(err, filechange.ErrInvalidQuery) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ListFileChangesResponse{Error: code}), err
	}
	changes := make([]*v1.FileChange, 0, len(page.Changes))
	for _, one := range page.Changes {
		if !e.auth.IsTakingFile(pluginID, one.Path, one.NoteTypes) {
			continue
		}
		changes = append(changes, &v1.FileChange{SourceId: int64(one.SourceID), Path: one.Path, Event: one.Event, Seq: one.Seq,
			Hash: one.Hash, Size: one.Size, Mtime: one.ModifiedAt,
		})
	}
	return connect.NewResponse(&v1.ListFileChangesResponse{
		Changes: changes, Next: newCursorMessage(page.Next), IsReset: page.IsReset, HasMore: page.HasMore,
	}), nil
}

func newCursorMessage(c filechange.Cursor) *v1.FileChangeCursor {
	return &v1.FileChangeCursor{FeedId: c.FeedID, Seq: c.Seq}
}
