package flashcards_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

// readTally hands out the vault's readers and says how many times each file was
// read.
type readTally struct {
	inner ports.VaultReaders
	mu    sync.Mutex
	reads map[string]int
}

func (c *readTally) Open(v domain.Vault) (ports.VaultReader, error) {
	one, err := c.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return deckReadCounter{VaultReader: one, of: c}, nil
}

func (c *readTally) take() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.reads
	c.reads = nil
	return out
}

type deckReadCounter struct {
	ports.VaultReader
	of *readTally
}

func (r deckReadCounter) Read(ctx context.Context, path string) ([]byte, error) {
	r.of.mu.Lock()
	if r.of.reads == nil {
		r.of.reads = make(map[string]int)
	}
	r.of.reads[path]++
	r.of.mu.Unlock()
	return r.VaultReader.Read(ctx, path)
}

func openTallied(t *testing.T, notes map[string]string) (vaulted, flashcards.ListCardFaces, *readTally) {
	t.Helper()
	s := openVault(t, notes)
	reads := &readTally{inner: s.env.Ports.Readers}
	faces := flashcards.NewListCardFaces(reads, s.notes, s.links)
	faces.Cores = 4
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	reads.take()
	return s, faces, reads
}

func ids(faces []flashcards.CardFace) []string {
	out := make([]string, 0, len(faces))
	for _, f := range faces {
		out = append(out, f.Deck+"|"+f.ID.Card+"|"+f.ID.Face)
	}
	return out
}

func TestASecondRequestDoesNotReadADeckThatHasNotChanged(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	again, err := faces.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); len(got) != 0 {
		t.Errorf("an unchanged vault was read again: %v", got)
	}
	if len(again) != 5 {
		t.Errorf("the second request stands for %d card faces, want 5", len(again))
	}
	for _, one := range again {
		front, _ := one.Lay()
		if front == "" {
			t.Errorf("a cached face lays out nothing: %+v", one.ID)
		}
	}
}

func TestADeckWhoseSizeChangedIsReadAgain(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	write(t, s, "decks/Words.md", vault["decks/Words.md"]+"\n")
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 {
		t.Errorf("the changed deck was read %d times: %v", got["decks/Words.md"], got)
	}
	if got["decks/Mammals.md"] != 0 {
		t.Errorf("the unchanged deck was read: %v", got)
	}
}

func TestADeckWhoseModificationTimeChangedIsReadAgain(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	at := filepath.Join(s.vault.Path, "decks", "Words.md")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(at, later, later); err != nil {
		t.Fatal(err)
	}
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); got["decks/Words.md"] != 1 {
		t.Errorf("a deck touched in place was read %d times: %v", got["decks/Words.md"], got)
	}
}

func TestADeckThatIsDeletedLeavesNoEntryBehind(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)
	at := filepath.Join(s.vault.Path, "decks", "Words.md")
	info, err := os.Stat(at)
	if err != nil {
		t.Fatal(err)
	}

	remove(t, s, "decks/Words.md")
	left, err := faces.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range left {
		if one.Deck == "decks/Words.md" {
			t.Fatalf("a deleted deck stands: %+v", one.ID)
		}
	}
	reads.take()

	// The same bytes under the same stamp are the file the entry was made from.
	write(t, s, "decks/Words.md", vault["decks/Words.md"])
	if err := os.Chtimes(at, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	back, err := faces.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := reads.take(); got["decks/Words.md"] != 1 {
		t.Errorf("a deck that came back was read %d times: %v", got["decks/Words.md"], got)
	}
	if len(back) != 5 {
		t.Errorf("the vault stands for %d card faces, want 5", len(back))
	}
}

func TestAChangedStencilIsReadAgainWithEveryDeckCutByIt(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	write(t, s, "Animal.md", vault["Animal.md"]+"\n")
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 1 {
		t.Errorf("the deck cut by the changed stencil was read %d times: %v", got["decks/Mammals.md"], got)
	}
	if got["decks/Words.md"] != 0 {
		t.Errorf("a deck cut by another stencil was read: %v", got)
	}
}

func TestInvalidatingAStencilForgetsEveryDeckCutByIt(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	faces.Invalidate(s.vault.ID, "Animal.md")
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 1 || got["decks/Words.md"] != 0 {
		t.Errorf("reads after forgetting a stencil: %v", got)
	}
}

func TestInvalidatingADeckForgetsIt(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	faces.Invalidate(s.vault.ID, "decks/Words.md")
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 || got["decks/Mammals.md"] != 0 {
		t.Errorf("reads after forgetting a deck: %v", got)
	}
}

func TestInvalidatingAllForgetsEveryDeck(t *testing.T) {
	t.Parallel()
	s, faces, reads := openTallied(t, vault)

	faces.InvalidateAll()
	if _, err := faces.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	got := reads.take()
	if got["decks/Words.md"] != 1 || got["decks/Mammals.md"] != 1 {
		t.Errorf("reads after forgetting everything: %v", got)
	}
}

func TestAMarkWrittenThroughTheAppIsStandingInTheNextRequest(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Term.md": vault["Term.md"],
		"decks/Words.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nCompost\n",
	}
	s := openVault(t, notes)
	faces := flashcards.NewListCardFaces(s.env.Ports.Readers, s.notes, s.links)
	marking := s.marking.WithIndex(func(ctx context.Context, v domain.Vault, paths []string) error {
		faces.Invalidate(v.ID, paths...)
		return s.scan(ctx, v, paths)
	})

	before, err := faces.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("a card with no mark stands: %v", ids(before))
	}
	if _, err := marking.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	after, err := faces.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Errorf("the marked card stands as %d card faces, want 1", len(after))
	}
}

func TestTheFacesComeBackInTheOrderOfThePathsWhateverParsesFirst(t *testing.T) {
	t.Parallel()
	notes := map[string]string{"Term.md": vault["Term.md"]}
	var paths []string
	for i := range 24 {
		path := fmt.Sprintf("decks/Deck%02d.md", i)
		paths = append(paths, path)
		notes[path] = fmt.Sprintf("---\ntype: deck\n---\n\n## Card ^q%02dwxyzpqr\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n", i, i, i)
	}
	s := openVault(t, notes)

	one := flashcards.NewListCardFaces(s.env.Ports.Readers, s.notes, s.links)
	want := ids(one.GetFaces(t.Context(), s.vault, paths))
	if len(want) != len(paths) {
		t.Fatalf("%d decks stand as %d card faces", len(paths), len(want))
	}

	for range 10 {
		many := flashcards.NewListCardFaces(s.env.Ports.Readers, s.notes, s.links)
		many.Cores = 64
		if got := ids(many.GetFaces(t.Context(), s.vault, paths)); !reflect.DeepEqual(got, want) {
			t.Fatalf("the order moved:\n got %v\nwant %v", got, want)
		}
	}
}

func TestRequestsAndInvalidationsMakeNoRaceBetweenThem(t *testing.T) {
	t.Parallel()
	s, faces, _ := openTallied(t, vault)

	var running sync.WaitGroup
	for range 8 {
		running.Add(2)
		go func() {
			defer running.Done()
			for range 20 {
				got, err := faces.Execute(t.Context(), s.vault)
				if err != nil || len(got) != 5 {
					t.Errorf("a request came back with %d card faces and %v", len(got), err)
					return
				}
			}
		}()
		go func() {
			defer running.Done()
			for range 20 {
				faces.Invalidate(s.vault.ID, "Term.md")
				faces.InvalidateAll()
			}
		}()
	}
	running.Wait()
}

func TestAPoolOfOneAndAPoolOfManyGiveTheSameFacesEveryTime(t *testing.T) {
	t.Parallel()
	notes := map[string]string{"Term.md": vault["Term.md"]}
	var paths []string
	for i := range 16 {
		path := fmt.Sprintf("decks/Deck%02d.md", i)
		paths = append(paths, path)
		notes[path] = fmt.Sprintf("---\ntype: deck\n---\n\n## Card ^q%02dwxyzpqr\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n", i, i, i)
	}
	s := openVault(t, notes)
	one := flashcards.NewListCardFaces(s.env.Ports.Readers, s.notes, s.links)
	one.Cores = 1
	want := ids(one.GetFaces(t.Context(), s.vault, paths))

	for _, cores := range []int{2, 3, 16, 64} {
		many := flashcards.NewListCardFaces(s.env.Ports.Readers, s.notes, s.links)
		many.Cores = cores
		for range 3 {
			if got := ids(many.GetFaces(t.Context(), s.vault, paths)); !reflect.DeepEqual(got, want) {
				t.Fatalf("a pool of %d gave another order:\n got %v\nwant %v", cores, got, want)
			}
		}
	}
}

func TestADeckPastTheFaceBoundIsParsedAgainWhileTheOthersStay(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	reads := &readTally{inner: s.env.Ports.Readers}
	faces := flashcards.NewListCardFaces(reads, s.notes, s.links)
	// Mammals stands as four faces and Words as one.
	faces.MaxFaces = 3
	for range 2 {
		if _, err := faces.Execute(t.Context(), s.vault); err != nil {
			t.Fatal(err)
		}
	}
	got := reads.take()
	if got["decks/Mammals.md"] != 2 || got["decks/Words.md"] != 1 {
		t.Errorf("reads under a bound of three faces: %v", got)
	}
}
