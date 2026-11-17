// Package changes follows the file change feed of a vault for a plugin: it reads the changes after
// the cursor the plugin keeps in its store folder of the vault, passes them to the plugin in batches,
// and moves the cursor on once a batch is applied.
//
// A change is delivered at least once: a batch whose apply failed, or whose cursor was not written,
// is read again. An apply sets the state a change describes, so applying a batch twice leaves what
// applying it once does.
package changes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// Store is the plugin's folder of the store of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
}

// Source is the feed of one vault and the file of the plugin's store folder of that vault the cursor
// is kept in.
type Source struct {
	Feed       host.FileChangeLister
	Store      Store
	CursorFile string
}

// Apply brings the plugin's state up to one batch of changes, in the order of their numbers.
type Apply func(ctx context.Context, batch []host.FileChange) error

// Review rebuilds the plugin's state from the vault as it is: the feed cannot continue from the
// cursor, because the plugin has none yet, the index was built anew or the cursor is older than the
// removals the feed keeps. The changes made during a review come in the next batches.
type Review func(ctx context.Context) error

// Follow reads the changes of the events after the cursor of the source up to the end of the feed.
// The cursor is written after each batch is applied and after a review; an error of apply or review
// leaves the cursor where it was and is returned.
func Follow(ctx context.Context, from Source, events []string, apply Apply, review Review) error {
	cursor, err := ReadCursor(ctx, from)
	if err != nil {
		return err
	}
	f := &follower{from: from, events: events, apply: apply, review: review, isReviewed: false}
	for {
		page, err := f.readBatch(ctx, cursor)
		if err != nil {
			return err
		}
		if page.Next != cursor {
			if err := writeCursor(ctx, from, page.Next); err != nil {
				return err
			}
			cursor = page.Next
		}
		if !page.IsReset && !page.HasMore {
			return nil
		}
	}
}

// follower is one run of Follow. isReviewed is true once the run reviewed.
type follower struct {
	from       Source
	events     []string
	apply      Apply
	review     Review
	isReviewed bool
}

func (f *follower) readBatch(ctx context.Context, cursor host.FileChangeCursor) (host.FileChangePage, error) {
	page, err := f.from.Feed.ListFileChanges(ctx, host.FileChangeQuery{Since: cursor, Events: f.events})
	if err != nil {
		return host.FileChangePage{}, fmt.Errorf("read the file changes: %w", err)
	}
	switch {
	case page.IsReset && f.isReviewed:
		return host.FileChangePage{}, errors.New("the file change feed reset again after a review")
	case page.IsReset:
		f.isReviewed = true
		return page, f.review(ctx)
	case len(page.Changes) > 0:
		return page, f.apply(ctx, page.Changes)
	}
	return page, nil
}

// ReadCursor returns the cursor kept in the store, the zero cursor when there is none or the file
// holds no cursor.
func ReadCursor(ctx context.Context, from Source) (host.FileChangeCursor, error) {
	content, err := from.Store.Read(ctx, from.CursorFile)
	if errors.Is(err, host.ErrNotExist) {
		return host.FileChangeCursor{}, nil
	}
	if err != nil {
		return host.FileChangeCursor{}, fmt.Errorf("read %s: %w", from.CursorFile, err)
	}
	return parseCursor(content), nil
}

func parseCursor(content []byte) host.FileChangeCursor {
	var cursor host.FileChangeCursor
	if json.Unmarshal(content, &cursor) != nil {
		return host.FileChangeCursor{}
	}
	return cursor
}

// Forget drops the cursor kept in the store, so the next Follow reviews.
func Forget(ctx context.Context, from Source) error {
	return writeCursor(ctx, from, host.FileChangeCursor{})
}

func writeCursor(ctx context.Context, from Source, cursor host.FileChangeCursor) error {
	content, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	if err := from.Store.Write(ctx, from.CursorFile, content); err != nil {
		return fmt.Errorf("write %s: %w", from.CursorFile, err)
	}
	return nil
}
