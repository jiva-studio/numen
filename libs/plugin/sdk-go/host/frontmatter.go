package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// ErrInvalidFrontmatterQuery is a read of frontmatter over its limits: more than 1000 paths, more than
// 64 keys or a page of more than 1000 notes.
var ErrInvalidFrontmatterQuery = errors.New("host: the frontmatter query exceeds the limits of one read")

// FrontmatterQuery names the notes a read of frontmatter returns: the notes at Paths, or every note of
// the vault when Paths is empty, with the values of Keys, or of every key when Keys is empty. PageSize
// is the number of notes of a page, 0 for the most; PageToken is the NextPageToken of the page before.
type FrontmatterQuery struct {
	Paths     []string
	Keys      []string
	PageSize  int
	PageToken string
}

// NoteFrontmatter is the frontmatter of one note, as JSON decodes it, and the number the index keeps
// the note under, which stays the same when the note moves.
type NoteFrontmatter struct {
	SourceID int64
	Path     string
	Values   map[string]any
}

// FrontmatterPage is one page of a read, in path order. NextPageToken reads the next page, and is
// empty after the last. Cursor is the place in the file change feed taken before the page was read:
// a reader that reads the feed from the cursor of its first page loses no change made during the read.
type FrontmatterPage struct {
	Notes         []NoteFrontmatter
	NextPageToken string
	Cursor        FileChangeCursor
}

func (c *Client) ReadFrontmatter(ctx context.Context, q FrontmatterQuery) (FrontmatterPage, error) {
	res, err := c.notes.ReadFrontmatter(ctx, request(c, &v1.ReadFrontmatterRequest{
		Paths: q.Paths, Keys: q.Keys, PageSize: int32(min(q.PageSize, 1<<30)), PageToken: q.PageToken,
	}))
	if connect.CodeOf(err) == connect.CodeInvalidArgument {
		return FrontmatterPage{}, fmt.Errorf("%w: %w", ErrInvalidFrontmatterQuery, err)
	}
	if err != nil {
		return FrontmatterPage{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return FrontmatterPage{}, err
	}
	page := FrontmatterPage{NextPageToken: res.Msg.GetNextPageToken(), Cursor: readCursor(res.Msg.GetCursor())}
	for _, one := range res.Msg.GetNotes() {
		values := map[string]any{}
		if err := json.Unmarshal(one.GetValues(), &values); err != nil {
			return FrontmatterPage{}, fmt.Errorf("read the frontmatter of %s: %w", one.GetPath(), err)
		}
		page.Notes = append(page.Notes, NoteFrontmatter{SourceID: one.GetSourceId(), Path: one.GetPath(), Values: values})
	}
	return page, nil
}

// FrontmatterWrite sets the keys of Values on the note at Path; a nil value removes the key.
// Baseline is the file state the caller read, the zero value writes over any state.
type FrontmatterWrite struct {
	Path     string
	Values   map[string]any
	Baseline domain.FileStat
}

// FrontmatterResult is one outcome of a write: Stat is the note after it.
type FrontmatterResult struct {
	Stat domain.FileStat
	Err  error
}

// WriteFrontmatter sets, replaces and removes top-level frontmatter keys of notes in one call, and
// returns one result per write, in order. A stale or failed note costs the others nothing. It needs
// vault:write, accepts at most 1000 notes, and returns ErrNoVault while the host shows no vault.
func (c *Client) WriteFrontmatter(ctx context.Context, writes []FrontmatterWrite) ([]FrontmatterResult, error) {
	notes := make([]*v1.NoteFrontmatterWrite, len(writes))
	for i, one := range writes {
		values := one.Values
		if values == nil {
			values = map[string]any{}
		}
		raw, err := json.Marshal(values)
		if err != nil {
			return nil, fmt.Errorf("write the frontmatter of %s: %w", one.Path, err)
		}
		notes[i] = &v1.NoteFrontmatterWrite{Path: one.Path, Values: raw, Baseline: writeFileStat(one.Baseline)}
	}
	res, err := c.notes.WriteFrontmatter(ctx, request(c, &v1.WriteFrontmatterRequest{Notes: notes}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodePermissionDenied {
			return nil, fmt.Errorf("%w: %w", ErrPermissionDenied, err)
		}
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]FrontmatterResult, 0, len(res.Msg.GetResults()))
	for _, one := range res.Msg.GetResults() {
		written := FrontmatterResult{Err: getNoteError(one.GetError(), one.GetDetail())}
		if written.Err == nil {
			written.Stat = readFileStat(one.GetStat())
		}
		out = append(out, written)
	}
	return out, nil
}
