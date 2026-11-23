package extract

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// CursorFile is the file of the plugin's store folder of a vault the cursor of its file change feed
// is kept in.
const CursorFile = "feed.cursor.json"

// Feed is the file change feed of a vault and the plugin's store folder of that vault.
type Feed struct {
	Changes host.FileChangeLister
	Store   changes.Store
}

func getFollowedEvents() []string {
	return []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}
}

// Follower writes the text layers of the PDF files the file change feed of a vault names as added or
// changed. A moved file keeps its text layer, which the host moves with it, and a removed one takes
// it away. A file that has a text layer made from its content as it is gets none again.
type Follower struct {
	writer *Writer
	getFeed  func(domain.VaultID) Feed
	report func(what string, err error)

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	runs     map[domain.VaultID]*run
}

// run is the following of the feed of one vault. isAgain is true when a wake came while it ran.
type run struct {
	cancel  context.CancelFunc
	isAgain bool
}

// NewFollower returns the follower of the feeds of the vaults that writes through writer. report
// receives the failures of the background work.
func NewFollower(writer *Writer, feeds func(domain.VaultID) Feed, report func(what string, err error)) *Follower {
	return &Follower{
		writer: writer, getFeed: feeds, report: report,
		wg: sync.WaitGroup{}, mu: sync.Mutex{}, isClosed: false, runs: map[domain.VaultID]*run{},
	}
}

// Wake starts following the feed of the vault in the background, and returns. A wake while the feed
// of the vault is followed follows it again once that ends.
func (f *Follower) Wake(ctx context.Context, vault domain.VaultID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.isClosed {
		return
	}
	if one, isRunning := f.runs[vault]; isRunning {
		one.isAgain = true
		return
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	one := &run{cancel: cancel, isAgain: false}
	f.runs[vault] = one
	f.wg.Go(func() {
		defer cancel()
		for isAgain := true; isAgain; {
			isAgain = f.followOnce(runCtx, vault)
		}
	})
}

func (f *Follower) followOnce(ctx context.Context, vault domain.VaultID) bool {
	if err := f.Follow(ctx, vault); err != nil && ctx.Err() == nil {
		f.report("follow the file changes of "+string(vault), err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	one := f.runs[vault]
	if one.isAgain && !f.isClosed {
		one.isAgain = false
		return true
	}
	delete(f.runs, vault)
	return false
}

// Wait waits for the following it started to end.
func (f *Follower) Wait() { f.wg.Wait() }

// Close stops the following it started and waits for it to end.
func (f *Follower) Close() {
	f.mu.Lock()
	f.isClosed = true
	for _, one := range f.runs {
		one.cancel()
	}
	f.mu.Unlock()
	f.Wait()
}

// Follow writes the text layers of the changes of the feed of the vault after its cursor, and returns
// when they are written. On a reset of the feed it reads every file the vault holds.
func (f *Follower) Follow(ctx context.Context, vault domain.VaultID) error {
	feed := f.getFeed(vault)
	from := changes.Source{Feed: feed.Changes, Store: feed.Store, CursorFile: CursorFile}
	return changes.Follow(ctx, from, getFollowedEvents(),
		func(ctx context.Context, batch []host.FileChange) error { return f.apply(ctx, vault, batch) },
		func(ctx context.Context) error { return f.review(ctx, vault, feed.Changes) })
}

func (f *Follower) apply(ctx context.Context, vault domain.VaultID, batch []host.FileChange) error {
	var failures []error
	isDone := map[int64]bool{}
	for _, change := range batch {
		if change.Event != host.EventFileAdded && change.Event != host.EventFileChanged {
			continue
		}
		if isDone[change.SourceID] || !isPDF(change.Path) {
			continue
		}
		isDone[change.SourceID] = true
		failures = append(failures, f.writer.WriteMissingText(ctx, Request{Vault: vault, Path: change.Path}))
	}
	return errors.Join(failures...)
}

func (f *Follower) review(ctx context.Context, vault domain.VaultID, feed host.FileChangeLister) error {
	added := []string{host.EventFileAdded}
	head, err := feed.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{}, Events: added, Limit: 0})
	if err != nil {
		return fmt.Errorf("read the file changes: %w", err)
	}
	q := host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}, Events: added, Limit: 0}
	for {
		page, err := feed.ListFileChanges(ctx, q)
		if err != nil {
			return fmt.Errorf("read the file changes: %w", err)
		}
		if page.IsReset {
			return errors.New("the file change feed reset while its files were read")
		}
		if err := f.apply(ctx, vault, page.Changes); err != nil {
			return err
		}
		if !page.HasMore {
			return nil
		}
		q.Since = page.Next
	}
}

func isPDF(filePath string) bool { return strings.EqualFold(path.Ext(filePath), ".pdf") }
