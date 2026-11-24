package hostendpoint

import (
	"context"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// memoryFeed is the file change feed of one vault, held as a list of changes in the order of their
// numbers. onRead runs while a read of frontmatter lists its notes.
type memoryFeed struct {
	feedID  string
	trimmed int64
	changes []filechange.Change
	onRead  func()
}

func (f *memoryFeed) GetHead(context.Context, domain.VaultID) (filechange.Head, error) {
	head := filechange.Head{Cursor: filechange.Cursor{FeedID: f.feedID}, TrimmedSeq: f.trimmed}
	if len(f.changes) > 0 {
		head.Cursor.Seq = f.changes[len(f.changes)-1].Seq
	}
	return head, nil
}

func (f *memoryFeed) ListFileChanges(
	_ context.Context, _ domain.VaultID, events []string, after, upTo int64, limit int,
) ([]filechange.Change, error) {
	var out []filechange.Change
	for _, one := range f.changes {
		if one.Seq > after && one.Seq <= upTo && slices.Contains(events, one.Event) && len(out) < limit {
			out = append(out, one)
		}
	}
	return out, nil
}

func (f *memoryFeed) add(event, path string) {
	seq := int64(len(f.changes) + 1)
	f.changes = append(f.changes, filechange.Change{SourceID: filechange.SourceID(seq), Path: path, Event: event, Seq: seq})
}

// feedVault is a vault whose feed and frontmatter the use cases read.
type feedVault struct {
	list query.ListFileChanges
	read query.ReadFrontmatter
}

func (r feedVault) ListFileChanges(ctx context.Context, q filechange.Query) (filechange.Page, error) {
	return r.list.Execute(ctx, domain.Vault{ID: "v1"}, q)
}

func (r feedVault) ReadFrontmatter(ctx context.Context, q frontmatter.Query) (frontmatter.Page, error) {
	return r.read.Execute(ctx, domain.Vault{ID: "v1"}, q)
}

// readingStored is stored frontmatter that runs a function while it is listed.
type readingStored struct {
	storedFrontmatter
	during func()
}

func (s readingStored) ListFrontmatter(ctx context.Context, id domain.VaultID, q frontmatter.Query) ([]frontmatter.Note, error) {
	s.during()
	return s.storedFrontmatter.ListFrontmatter(ctx, id, q)
}

func newFeedClient(t *testing.T, token string, feed *memoryFeed, stored storedFrontmatter) HostVaultClient {
	t.Helper()
	hosted := newCallers()
	during := func() {
		if feed.onRead != nil {
			feed.onRead()
		}
	}
	hosted.vault = vaultOf(feedVault{
		list: query.NewListFileChanges(feed),
		read: query.NewReadFrontmatter(readingStored{storedFrontmatter: stored, during: during}, feed, query.GetFrontmatterLimits()),
	})
	return newVaultClient(t, hosted, token)
}

func listFileChanges(t *testing.T, client HostVaultClient, request *v1.ListFileChangesRequest) *v1.ListFileChangesResponse {
	t.Helper()
	res, err := client.ListFileChanges(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func getChangePaths(changes []*v1.FileChange) []string {
	out := make([]string, 0, len(changes))
	for _, one := range changes {
		out = append(out, one.GetPath())
	}
	return out
}

func TestListFileChangesPagesThroughTheEventsAsked(t *testing.T) {
	feed := &memoryFeed{feedID: "feed-1"}
	feed.add(manifest.EventFileAdded, "a.md")
	feed.add(manifest.EventNoteFrontmatterChanged, "a.md")
	feed.add(manifest.EventFileChanged, "b.pdf")
	feed.add(manifest.EventNoteFrontmatterChanged, "c.md")
	client := newFeedClient(t, "token-a", feed, storedFrontmatter{})
	events := []string{manifest.EventNoteFrontmatterChanged, manifest.EventFileRemoved}

	first := listFileChanges(t, client, &v1.ListFileChangesRequest{
		Since: &v1.FileChangeCursor{FeedId: "feed-1"}, Events: events, Limit: 1,
	})
	if got := getChangePaths(first.GetChanges()); !slices.Equal(got, []string{"a.md"}) || !first.GetHasMore() || first.GetIsReset() {
		t.Fatalf("the first page is %v, has more %v, reset %v", got, first.GetHasMore(), first.GetIsReset())
	}
	second := listFileChanges(t, client, &v1.ListFileChangesRequest{Since: first.GetNext(), Events: events, Limit: 1})
	if got := getChangePaths(second.GetChanges()); !slices.Equal(got, []string{"c.md"}) || second.GetHasMore() {
		t.Fatalf("the second page is %v, has more %v", got, second.GetHasMore())
	}
	if second.GetNext().GetSeq() != 4 {
		t.Errorf("the last page ends at %d, want the head 4", second.GetNext().GetSeq())
	}
}

func TestListFileChangesResetsACursorTheFeedCannotContinue(t *testing.T) {
	feed := &memoryFeed{feedID: "feed-2", trimmed: 2}
	feed.add(manifest.EventFileAdded, "a.md")
	feed.add(manifest.EventFileRemoved, "")
	feed.add(manifest.EventFileAdded, "b.md")
	client := newFeedClient(t, "token-a", feed, storedFrontmatter{})
	events := []string{manifest.EventFileAdded}

	for name, since := range map[string]*v1.FileChangeCursor{
		"no cursor":           nil,
		"another feed":        {FeedId: "feed-1", Seq: 3},
		"before the trimming": {FeedId: "feed-2", Seq: 1},
	} {
		res := listFileChanges(t, client, &v1.ListFileChangesRequest{Since: since, Events: events})
		if !res.GetIsReset() || len(res.GetChanges()) != 0 || res.GetNext().GetFeedId() != "feed-2" || res.GetNext().GetSeq() != 3 {
			t.Errorf("%s answered %v", name, res)
		}
	}
	res := listFileChanges(t, client, &v1.ListFileChangesRequest{Since: &v1.FileChangeCursor{FeedId: "feed-2", Seq: 2}, Events: events})
	if res.GetIsReset() || !slices.Equal(getChangePaths(res.GetChanges()), []string{"b.md"}) {
		t.Errorf("a cursor at the trimming answered %v", res)
	}
}

func TestListFileChangesOfAnUnknownEventIsAnInvalidArgument(t *testing.T) {
	client := newFeedClient(t, "token-a", &memoryFeed{feedID: "feed-1"}, storedFrontmatter{})

	for name, request := range map[string]*v1.ListFileChangesRequest{
		"no events":      {},
		"unknown event":  {Events: []string{"file.renamed"}},
		"too large page": {Events: []string{manifest.EventFileAdded}, Limit: 1001},
	} {
		_, err := client.ListFileChanges(t.Context(), connect.NewRequest(request))
		if got := readCode(t, err); got != connect.CodeInvalidArgument {
			t.Errorf("%s answered %v, want an invalid argument", name, got)
		}
	}
}

func TestListFileChangesIsRefusedWithoutVaultRead(t *testing.T) {
	client := newFeedClient(t, "token-b", &memoryFeed{feedID: "feed-1"}, storedFrontmatter{})

	_, err := client.ListFileChanges(t.Context(), connect.NewRequest(&v1.ListFileChangesRequest{Events: []string{manifest.EventFileAdded}}))

	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("a read without vault:read is %v", got)
	}
}

func TestReadFrontmatterReturnsTheCursorTakenBeforeTheRead(t *testing.T) {
	feed := &memoryFeed{feedID: "feed-1"}
	feed.add(manifest.EventNoteFrontmatterChanged, "a.md")
	feed.onRead = func() {
		feed.add(manifest.EventNoteFrontmatterChanged, "b.md")
		feed.onRead = nil
	}
	client := newFeedClient(t, "token-a", feed, storedFrontmatter{"a.md": {"tags": "lake"}})

	page, err := client.ReadFrontmatter(t.Context(), connect.NewRequest(&v1.ReadFrontmatterRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if page.Msg.GetCursor().GetSeq() != 1 {
		t.Fatalf("the read returned the cursor %v, want the one before the read", page.Msg.GetCursor())
	}
	res := listFileChanges(t, client, &v1.ListFileChangesRequest{
		Since: page.Msg.GetCursor(), Events: []string{manifest.EventNoteFrontmatterChanged},
	})
	if got := getChangePaths(res.GetChanges()); !slices.Equal(got, []string{"b.md"}) {
		t.Errorf("the feed after the cursor holds %v, want the change made during the read", got)
	}
}

func TestListFileChangesReturnsTheHashSizeAndModificationTimeOfEachChange(t *testing.T) {
	feed := &memoryFeed{feedID: "feed-1"}
	feed.add(manifest.EventFileChanged, "a.md")
	feed.changes[0].Hash = domain.HashContent([]byte("A"))
	feed.changes[0].Size = 1
	feed.changes[0].ModifiedAt = 1_700_000_000_000_000_123
	feed.add(manifest.EventFileRemoved, "b.md")
	client := newFeedClient(t, "token-a", feed, storedFrontmatter{})

	res := listFileChanges(t, client, &v1.ListFileChangesRequest{
		Since: &v1.FileChangeCursor{FeedId: "feed-1"}, Events: []string{manifest.EventFileChanged, manifest.EventFileRemoved},
	})

	if len(res.GetChanges()) != 2 {
		t.Fatalf("the read returned %v, want two changes", res.GetChanges())
	}
	changed, removed := res.GetChanges()[0], res.GetChanges()[1]
	if changed.GetHash() != domain.HashContent([]byte("A")) || changed.GetSize() != 1 || changed.GetMtime() != 1_700_000_000_000_000_123 {
		t.Errorf("the change of a.md has hash %q, size %d, mtime %d", changed.GetHash(), changed.GetSize(), changed.GetMtime())
	}
	if removed.GetHash() != "" || removed.GetSize() != 0 || removed.GetMtime() != 0 {
		t.Errorf("the removal of b.md has hash %q, size %d, mtime %d, want empty values", removed.GetHash(), removed.GetSize(), removed.GetMtime())
	}
}
