package query

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
)

// FileChangePageSize is the most changes one read of the feed returns.
const FileChangePageSize = 1000

// ListFileChanges reads the file change feed of a vault, a page at a time.
type ListFileChanges struct {
	feed FileChangeFeed
}

func NewListFileChanges(feed FileChangeFeed) ListFileChanges {
	return ListFileChanges{feed: feed}
}

// Execute returns the changes of the events after the cursor of the query. A reader whose cursor the
// feed cannot continue gets no changes, IsReset and the cursor of the feed now. A Limit of 0 reads a
// page of FileChangePageSize; a larger page, no events or an event the feed does not keep is
// filechange.ErrInvalidQuery.
func (u ListFileChanges) Execute(ctx context.Context, v domain.Vault, q filechange.Query) (filechange.Page, error) {
	if err := checkFileChangeQuery(q); err != nil {
		return filechange.Page{}, err
	}
	limit := q.Limit
	if limit == 0 {
		limit = FileChangePageSize
	}
	head, err := u.feed.GetHead(ctx, v.ID)
	if err != nil {
		return filechange.Page{}, err
	}
	if head.IsReset(q.Since) {
		return filechange.Page{Next: head.Cursor, IsReset: true}, nil
	}
	changes, err := u.feed.ListFileChanges(ctx, v.ID, q.Events, q.Since.Seq, head.Cursor.Seq, limit+1)
	if err != nil {
		return filechange.Page{}, err
	}
	if len(changes) <= limit {
		return filechange.Page{Changes: changes, Next: head.Cursor}, nil
	}
	changes = changes[:limit]
	next := filechange.Cursor{FeedID: head.Cursor.FeedID, Seq: changes[limit-1].Seq}
	return filechange.Page{Changes: changes, Next: next, HasMore: true}, nil
}

func checkFileChangeQuery(q filechange.Query) error {
	if q.Limit < 0 || q.Limit > FileChangePageSize {
		return fmt.Errorf("%w: a page of %d changes, at most %d", filechange.ErrInvalidQuery, q.Limit, FileChangePageSize)
	}
	if len(q.Events) == 0 {
		return fmt.Errorf("%w: no events", filechange.ErrInvalidQuery)
	}
	for _, event := range q.Events {
		if !filechange.IsEvent(event) {
			return fmt.Errorf("%w: the feed keeps no event %q", filechange.ErrInvalidQuery, event)
		}
	}
	return nil
}
