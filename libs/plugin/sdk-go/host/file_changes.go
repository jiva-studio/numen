package host

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/protocol"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// ErrInvalidFileChangeQuery is a read of the file change feed over its limits, without events or with
// an event the feed does not keep.
var ErrInvalidFileChangeQuery = errors.New("host: the file change query is not one the feed answers")

// FileChangeCursor is a place in the file change feed of a vault: the feed and the number of the last
// change read. The zero cursor is before the start of any feed.
type FileChangeCursor struct {
	FeedID string `json:"feedId"`
	Seq    int64  `json:"seq"`
}

// FileChange is the latest event of one kind of one file: the number the index keeps the file under,
// which stays the same when the file moves, its path now, or where it was for file.removed, and the
// number of the change. Hash, Size and ModifiedAt (nanoseconds since the Unix epoch) are those of the
// file now, empty and 0 for file.removed.
type FileChange struct {
	SourceID   int64
	Path       string
	Event      string
	Hash       string
	Seq        int64
	Size       int64
	ModifiedAt int64
}

// FileChangeQuery names the changes a read returns: those after Since of the Events, at most Limit,
// 0 for the most (1000).
type FileChangeQuery struct {
	Since  FileChangeCursor
	Events []string
	Limit  int
}

// FileChangePage is one page of a read of the feed. Next is the cursor the next read starts from.
// IsReset is true when the feed cannot continue from Since; Changes is empty then, and the reader
// reads the vault again from the start. HasMore is true when more changes follow Next.
type FileChangePage struct {
	Changes []FileChange
	Next    FileChangeCursor
	IsReset bool
	HasMore bool
}

// FileChangeLister reads the file change feed of a vault. It needs vault:read.
type FileChangeLister interface {
	ListFileChanges(ctx context.Context, q FileChangeQuery) (FileChangePage, error)
}

func (c *Client) ListFileChanges(ctx context.Context, q FileChangeQuery) (FileChangePage, error) {
	res, err := c.files.ListFileChanges(ctx, request(c, &v1.ListFileChangesRequest{
		Since: newCursorMessage(q.Since), Events: q.Events, Limit: int32(min(q.Limit, 1<<30)),
	}))
	if connect.CodeOf(err) == connect.CodeInvalidArgument {
		return FileChangePage{}, fmt.Errorf("%w: %w", ErrInvalidFileChangeQuery, err)
	}
	if err != nil {
		return FileChangePage{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return FileChangePage{}, err
	}
	page := FileChangePage{Next: readCursor(res.Msg.GetNext()), IsReset: res.Msg.GetIsReset(), HasMore: res.Msg.GetHasMore()}
	for _, one := range res.Msg.GetChanges() {
		page.Changes = append(page.Changes, FileChange{
			SourceID: one.GetSourceId(), Path: one.GetPath(), Event: one.GetEvent(), Seq: one.GetSeq(),
			Hash: one.GetHash(), Size: one.GetSize(), ModifiedAt: one.GetMtime(),
		})
	}
	return page, nil
}

func newCursorMessage(c FileChangeCursor) *v1.FileChangeCursor {
	return &v1.FileChangeCursor{FeedId: c.FeedID, Seq: c.Seq}
}

func readCursor(c *v1.FileChangeCursor) FileChangeCursor {
	return FileChangeCursor{FeedID: c.GetFeedId(), Seq: c.GetSeq()}
}

// ReadFilesChangedEvent returns the identity of the vault that an event of the host names as having new
// changes in its file change feed, and false for any other event. The host emits the event after a write of
// the index stored changes and the write is committed: a handler reads the feed from its cursor, with
// changes.Follow, and handles every change after it.
func ReadFilesChangedEvent(event string) (domain.VaultID, bool) {
	vaultID, found := protocol.ReadFilesChangedEvent(event)
	return domain.VaultID(vaultID), found
}
