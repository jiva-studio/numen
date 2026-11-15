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

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/tagcache"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/tags"
)

// recordedReads is the frontmatter of a vault, recording each query asked of it. A read waits on hold
// when hold is set, and fails while failing is set.
type recordedReads struct {
	*hosttest.Vault
	mu      sync.Mutex
	asked   []host.FrontmatterQuery
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

func (r *recordedReads) take() []host.FrontmatterQuery {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.asked
	r.asked = nil
	return out
}

type scene struct {
	vault *hosttest.Vault
	reads *recordedReads
	cache *tagcache.Caches
}

func newScene(files map[string]string) *scene {
	v := hosttest.New("garden")
	for path, content := range files {
		v.File(path, content)
	}
	s := &scene{vault: v, reads: &recordedReads{Vault: v}}
	s.cache = tagcache.New(func(domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: s.reads, Store: v.Store()}
	})
	return s
}

func (s *scene) readFile(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := s.vault.Store().Read(t.Context(), tagcache.FileName)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Version int                 `json:"version"`
		Notes   map[string][]string `json:"notes"`
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Version != 1 {
		t.Errorf("tags.json is version %d", stored.Version)
	}
	return stored.Notes
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

func TestAnUpdateOfANoteReadsThatNoteAlone(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.File("a.md", "---\ntags: [lake]\n---\n")
	if err := s.cache.UpdateNotes(t.Context(), "garden", []string{"a.md"}); err != nil {
		t.Fatal(err)
	}

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

func TestARemovedNoteIsDroppedFromTheTags(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getCounts(t)

	s.vault.Remove("a.md")
	if err := s.cache.UpdateNotes(t.Context(), "garden", []string{"a.md"}); err != nil {
		t.Fatal(err)
	}

	if counts := s.getCounts(t); len(counts) != 1 || counts["lake"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if got := s.readFile(t); len(got) != 1 {
		t.Errorf("tags.json holds %v", got)
	}
}

func TestAnUpdateBeforeTheFirstUseReadsEveryNote(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})

	if err := s.cache.UpdateNotes(t.Context(), "garden", []string{"a.md"}); err != nil {
		t.Fatal(err)
	}

	if asked := s.reads.take(); len(asked) != 1 || len(asked[0].Paths) != 0 {
		t.Errorf("asked %+v, want one read of every note", asked)
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

func TestAFailedUpdateReadsEveryNoteAtTheNextUse(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.failing = true
	if err := s.cache.UpdateNotes(t.Context(), "garden", []string{"a.md"}); err == nil {
		t.Fatal("a failed read was not reported")
	}
	s.reads.failing = false
	s.reads.take()

	s.vault.File("b.md", "---\ntags: lake\n---\n")
	counts := s.getCounts(t)

	if asked := s.reads.take(); len(asked) != 1 || len(asked[0].Paths) != 0 {
		t.Errorf("asked %+v, want every note read again", asked)
	}
	if counts["lake"] != 1 || counts["river"] != 1 {
		t.Errorf("counts = %v", counts)
	}
}

func TestAnUpdateOfSeveralNotesReadsThemInOneRead(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n", "c.md": "---\ntags: river\n---\n"})
	s.getCounts(t)
	s.reads.take()

	s.vault.File("a.md", "---\ntags: lake\n---\n")
	s.vault.Remove("b.md")
	if err := s.cache.UpdateNotes(t.Context(), "garden", []string{"a.md", "b.md"}); err != nil {
		t.Fatal(err)
	}

	if asked := s.reads.take(); len(asked) != 1 || !slices.Equal(asked[0].Paths, []string{"a.md", "b.md"}) {
		t.Errorf("asked %+v, want one read of both notes", asked)
	}
	if counts := s.getCounts(t); counts["river"] != 1 || counts["lake"] != 1 || len(counts) != 2 {
		t.Errorf("counts = %v", counts)
	}
	if got := s.readFile(t); len(got) != 2 || !slices.Equal(got["a.md"], []string{"lake"}) {
		t.Errorf("tags.json holds %v", got)
	}
}

func TestANewCacheReadsTheTagsChangedWhileThePluginWasNotRunning(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	s.getCounts(t)

	s.vault.File("a.md", "---\ntags: lake\n---\n")
	s.vault.File("c.md", "---\ntags: delta\n---\n")
	s.vault.Remove("b.md")
	s.cache = tagcache.New(func(domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: s.reads, Store: s.vault.Store()}
	})

	if counts := s.getCounts(t); len(counts) != 2 || counts["lake"] != 1 || counts["delta"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if got := s.readFile(t); len(got) != 2 || !slices.Equal(got["c.md"], []string{"delta"}) {
		t.Errorf("tags.json holds %v", got)
	}
}
