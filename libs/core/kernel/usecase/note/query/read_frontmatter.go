package query

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// FrontmatterLimits bound one read of frontmatter: the notes of a page, the paths and the keys a
// request names.
type FrontmatterLimits struct {
	PageSize int
	MaxPaths int
	MaxKeys  int
}

// GetFrontmatterLimits returns the limits a host call reads frontmatter under.
func GetFrontmatterLimits() FrontmatterLimits {
	return FrontmatterLimits{PageSize: 1000, MaxPaths: 1000, MaxKeys: 64}
}

// ReadFrontmatter reads the stored frontmatter of the notes of a vault, a page at a time.
type ReadFrontmatter struct {
	queries noteport.FrontmatterQueries
	feed    sourceport.FileChangeFeed
	limits  FrontmatterLimits
}

func NewReadFrontmatter(queries noteport.FrontmatterQueries, feed sourceport.FileChangeFeed, limits FrontmatterLimits) ReadFrontmatter {
	return ReadFrontmatter{queries: queries, feed: feed, limits: limits}
}

// Execute returns one page of the notes the query names, and the cursor of the file change feed taken
// before the page was read. A Limit of 0 reads a page of the page size;
// a larger page, or more paths or keys than the limits allow, is frontmatter.ErrInvalidQuery.
func (u ReadFrontmatter) Execute(ctx context.Context, v domain.Vault, q frontmatter.Query) (frontmatter.Page, error) {
	switch {
	case len(q.Paths) > u.limits.MaxPaths:
		return frontmatter.Page{}, fmt.Errorf("%w: %d paths, at most %d", frontmatter.ErrInvalidQuery, len(q.Paths), u.limits.MaxPaths)
	case len(q.Keys) > u.limits.MaxKeys:
		return frontmatter.Page{}, fmt.Errorf("%w: %d keys, at most %d", frontmatter.ErrInvalidQuery, len(q.Keys), u.limits.MaxKeys)
	case q.Limit < 0 || q.Limit > u.limits.PageSize:
		return frontmatter.Page{}, fmt.Errorf("%w: a page of %d notes, at most %d", frontmatter.ErrInvalidQuery, q.Limit, u.limits.PageSize)
	}
	if q.Limit == 0 {
		q.Limit = u.limits.PageSize
	}
	head, err := u.feed.GetHead(ctx, v.ID)
	if err != nil {
		return frontmatter.Page{}, err
	}
	notes, err := u.queries.ListFrontmatter(ctx, v.ID, q)
	if err != nil {
		return frontmatter.Page{}, err
	}
	page := frontmatter.Page{Notes: notes, Cursor: head.Cursor}
	if len(notes) == q.Limit {
		page.After = notes[len(notes)-1].Path
	}
	return page, nil
}
