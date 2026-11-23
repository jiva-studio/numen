package tagcache_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tagfile"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

// recordedReads is the frontmatter and the feed of a vault, recording each query asked of them. A read
// of frontmatter waits on hold when hold is set, and fails while failing is set.
type recordedReads struct {
	*hosttest.Vault
	mu      sync.Mutex
	asked   []host.FrontmatterQuery
	feeds   int
	hold    chan struct{}
	started chan struct{}
	failing bool
}

func (r *recordedReads) ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error) {
	r.mu.Lock()
	r.asked = append(r.asked, q)
	hold, started, failing := r.hold, r.started, r.failing
	r.mu.Unlock()
	if hold != nil {
		started <- struct{}{}
		<-hold
	}
	if failing {
		return host.FrontmatterPage{}, errors.New("the index is busy")
	}
	return r.Vault.ReadFrontmatter(ctx, q)
}

func (r *recordedReads) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	r.mu.Lock()
	r.feeds++
	r.mu.Unlock()
	return r.Vault.ListFileChanges(ctx, q)
}

func (r *recordedReads) take() []host.FrontmatterQuery {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.asked
	r.asked, r.feeds = nil, 0
	return out
}

func (r *recordedReads) getFeedReads() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.feeds
}

// cursorStore is the plugin's store folder of a vault whose writes of the cursor fail while
// isRefused is set.
type cursorStore struct {
	*hosttest.Store
	isRefused bool
}

func (c *cursorStore) Write(ctx context.Context, name string, content []byte) error {
	if c.isRefused && name == tagcache.CursorFile {
		return errors.New("disk full")
	}
	return c.Store.Write(ctx, name, content)
}

type scene struct {
	vault *hosttest.Vault
	reads *recordedReads
	store *cursorStore
	cache *tagcache.Caches
}

func newScene(files map[string]string) *scene {
	v := hosttest.New("garden")
	for path, content := range files {
		v.WriteFile(path, content)
	}
	s := &scene{vault: v, reads: &recordedReads{Vault: v}, store: &cursorStore{Store: v.Store()}}
	s.restart()
	return s
}

func (s *scene) restart() {
	s.cache = tagcache.New(func(domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: s.reads, Feed: s.reads, Store: s.store}
	})
}

func (s *scene) readFile(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := s.vault.Store().Read(t.Context(), tagcache.FileName)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Version int                     `json:"version"`
		Notes   map[string]tagfile.Note `json:"notes"`
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Version != 2 {
		t.Errorf("tags.json is version %d", stored.Version)
	}
	out := map[string][]string{}
	for _, note := range stored.Notes {
		out[note.Path] = note.Tags
	}
	return out
}

func (s *scene) getCounts(t *testing.T) map[string]int {
	t.Helper()
	tree, err := s.cache.GetTree(t.Context(), "garden", "")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	var walk func([]tags.Node)
	walk = func(nodes []tags.Node) {
		for _, node := range nodes {
			out[node.Tag] = node.Count
			walk(node.Children)
		}
	}
	walk(tree)
	return out
}

func (s *scene) update(t *testing.T) {
	t.Helper()
	if err := s.cache.UpdateVault(t.Context(), "garden"); err != nil {
		t.Fatal(err)
	}
}

func TestTheFirstUseReadsEveryNoteAndWritesTheTags(t *testing.T) {
	s := newScene(map[string]string{
		"a.md": "---\ntags: [river, delta]\n---\n", "b.md": "---\ntags: river\n---\n", "c.md": "# no tags\n",
	})

	counts := s.getCounts(t)

	if counts["river"] != 2 || counts["delta"] != 1 || len(counts) != 2 {
		t.Errorf("counts = %v", counts)
	}
	asked := s.reads.take()
	if len(asked) != 1 || len(asked[0].Paths) != 0 || !slices.Equal(asked[0].Keys, []string{"tags"}) {
		t.Errorf("asked %+v, want one read of the key tags of every note", asked)
	}
	if got := s.readFile(t); !slices.Equal(got["a.md"], []string{"river", "delta"}) || len(got) != 2 {
		t.Errorf("tags.json holds %v", got)
	}
	s.getCounts(t)
	if asked := s.reads.take(); len(asked) != 0 {
		t.Errorf("a second use read %+v again", asked)
	}
}

func TestAnUpdateReadsTheNoteWhoseFrontmatterChangedAlone(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.WriteFile("a.md", "---\ntags: [lake]\n---\n")
	s.update(t)

	if asked := s.reads.take(); len(asked) != 1 || !slices.Equal(asked[0].Paths, []string{"a.md"}) {
		t.Errorf("asked %+v, want the note alone", asked)
	}
	if counts := s.getCounts(t); counts["river"] != 1 || counts["lake"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if got := s.readFile(t); !slices.Equal(got["a.md"], []string{"lake"}) {
		t.Errorf("tags.json holds %v", got)
	}
}

func TestARemovedNoteIsDroppedWithoutARead(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.Remove("a.md")
	s.update(t)

	if asked := s.reads.take(); len(asked) != 0 {
		t.Errorf("a removal read %+v", asked)
	}
	if counts := s.getCounts(t); len(counts) != 1 || counts["lake"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if got := s.readFile(t); len(got) != 1 {
		t.Errorf("tags.json holds %v", got)
	}
}

func TestAMovedNoteKeepsItsTagsWithoutARead(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	if _, err := s.vault.MoveNotes(t.Context(), []string{"a.md"}, "rivers"); err != nil {
		t.Fatal(err)
	}
	s.update(t)

	if asked := s.reads.take(); len(asked) != 0 {
		t.Errorf("a move read %+v", asked)
	}
	if got := s.readFile(t); len(got) != 1 || !slices.Equal(got["rivers/a.md"], []string{"river"}) {
		t.Errorf("tags.json holds %v, want the tags under the new path", got)
	}
}

func TestAListAskedDuringTheFirstReadWaitsForIt(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.reads.hold, s.reads.started = make(chan struct{}), make(chan struct{}, 2)
	ctx := t.Context()
	first := make(chan error, 1)
	go func() {
		_, err := s.cache.GetTree(ctx, "garden", "")
		first <- err
	}()
	<-s.reads.started
	second := make(chan []tags.Node, 1)
	go func() {
		tree, err := s.cache.GetTree(ctx, "garden", "")
		if err != nil {
			tree = nil
		}
		second <- tree
	}()

	select {
	case <-second:
		t.Fatal("a list was answered while the first read was running")
	case <-time.After(50 * time.Millisecond):
	}
	close(s.reads.hold)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if tree := <-second; len(tree) != 1 || tree[0].Count != 1 {
		t.Errorf("the waiting list got %+v", tree)
	}
}

func TestAFailedUpdateReadsTheSameChangesAtTheNextUse(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.vault.WriteFile("a.md", "---\ntags: lake\n---\n")
	s.reads.failing = true
	if err := s.cache.UpdateVault(t.Context(), "garden"); err == nil {
		t.Fatal("a failed read was not reported")
	}
	s.reads.failing = false
	s.reads.take()

	counts := s.getCounts(t)

	if asked := s.reads.take(); len(asked) != 1 || !slices.Equal(asked[0].Paths, []string{"a.md"}) {
		t.Errorf("asked %+v, want the changed note read again", asked)
	}
	if counts["lake"] != 1 || len(counts) != 1 {
		t.Errorf("counts = %v", counts)
	}
}

func TestApplyingABatchTwiceMatchesApplyingItOnce(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getCounts(t)
	s.vault.WriteFile("a.md", "---\ntags: delta\n---\n")
	s.vault.Remove("b.md")
	s.store.isRefused = true
	if err := s.cache.UpdateVault(t.Context(), "garden"); err == nil {
		t.Fatal("a cursor that was not written was not reported")
	}
	once := s.readFile(t)

	s.store.isRefused = false
	s.restart()
	s.update(t)

	if twice := s.readFile(t); len(twice) != 1 || !slices.Equal(twice["a.md"], once["a.md"]) || len(once) != 1 {
		t.Errorf("tags.json holds %v after the batch once and %v after it twice", once, twice)
	}
}

func TestAnUpdateOfSeveralNotesReadsThemInOneRead(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n", "c.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.WriteFile("a.md", "---\ntags: lake\n---\n")
	s.vault.WriteFile("b.md", "---\ntags: delta\n---\n")
	s.update(t)

	if asked := s.reads.take(); len(asked) != 1 || !slices.Equal(asked[0].Paths, []string{"a.md", "b.md"}) {
		t.Errorf("asked %+v, want one read of both notes", asked)
	}
	if counts := s.getCounts(t); counts["river"] != 1 || counts["lake"] != 1 || counts["delta"] != 1 {
		t.Errorf("counts = %v", counts)
	}
}

func TestARestartReadsOnlyTheChangesMadeWhileThePluginWasNotRunning(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	s.getCounts(t)

	s.vault.WriteFile("a.md", "---\ntags: lake\n---\n")
	s.vault.WriteFile("c.md", "---\ntags: delta\n---\n")
	s.vault.Remove("b.md")
	s.restart()
	s.reads.take()

	if counts := s.getCounts(t); len(counts) != 2 || counts["lake"] != 1 || counts["delta"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	feeds := s.reads.getFeedReads()
	asked := s.reads.take()
	if len(asked) != 1 || !slices.Equal(asked[0].Paths, []string{"a.md", "c.md"}) || feeds != 1 {
		t.Errorf("the restart asked %+v and read the feed %d times, want one read of the changed notes", asked, feeds)
	}
	if got := s.readFile(t); len(got) != 2 || !slices.Equal(got["c.md"], []string{"delta"}) {
		t.Errorf("tags.json holds %v", got)
	}
}

func TestARebuiltIndexReadsEveryNoteAgain(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.ResetFeed()
	s.update(t)

	if asked := s.reads.take(); len(asked) != 1 || len(asked[0].Paths) != 0 {
		t.Errorf("asked %+v, want every note read again", asked)
	}
}
