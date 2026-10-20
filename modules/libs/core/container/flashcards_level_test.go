package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const (
	stencilFile = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"
	presetFile = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\n---\n\n# Sanskrit\n"
	unmarkedDeck = "---\ntype: deck\n---\n" +
		"\n## Leaf mould\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n"
)

// walkedVault is a vault of the notes given, read into an index of its own.
func walkedVault(t *testing.T, notes map[string]string) (Config, *Index, domain.Vault) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := Config{IndexPath: indexfile.Path(t), SchedulesPath: t.TempDir()}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	v := testsupport.NewVault(t, notes)
	if _, err := cfg.VaultOpener(db).Begin(t.Context(), v).Read(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	return cfg, db, v
}

func readFile(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A deck marked is brought level in the index, the decks parsed before it are
// forgotten, and the cards stand afterwards.
func TestADeckMarkedIsBroughtLevelAndItsCardsStand(t *testing.T) {
	cfg, db, v := walkedVault(t, map[string]string{"Term.md": stencilFile, "decks/Own.md": unmarkedDeck})
	var levelled [][]string
	held := cfg.Flashcards(db.Queries(), db.Links(), db.Problems(),
		func(_ context.Context, _ domain.Vault, paths []string) error {
			levelled = append(levelled, paths)
			return nil
		})

	before, err := held.CardFaces.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("a card with no mark stands %d times", len(before))
	}
	if _, err := held.Marks.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	if len(levelled) != 1 || !slices.Equal(levelled[0], []string{"decks/Own.md"}) {
		t.Errorf("the index was brought level with %v", levelled)
	}
	if !strings.Contains(readFile(t, v, "decks/Own.md"), "## Leaf mould ^") {
		t.Error("the deck carries no mark")
	}
	after, err := held.CardFaces.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Errorf("the marked card stands %d times, want once", len(after))
	}
}

// A window with no index to bring level still writes its decks.
func TestADeckIsMarkedWhereNoIndexIsGivenToLevel(t *testing.T) {
	cfg, db, v := walkedVault(t, map[string]string{"Term.md": stencilFile, "decks/Own.md": unmarkedDeck})
	held := cfg.Flashcards(db.Queries(), db.Links(), db.Problems(), nil)

	marked, err := held.Marks.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if len(marked.Unwritten) != 0 {
		t.Errorf("named %v as unwritten", marked.Unwritten)
	}
	if !strings.Contains(readFile(t, v, "decks/Own.md"), "## Leaf mould ^") {
		t.Error("the deck carries no mark")
	}
}

// A preset saved is written all the same when the index cannot be brought
// level, and the failure comes back; with an index that can, none does.
func TestAPresetSavedSaysWhetherTheIndexWasBroughtLevel(t *testing.T) {
	for _, one := range []struct {
		what     string
		index    func(context.Context, domain.Vault, []string) error
		isFailed bool
	}{
		{"no index given", nil, false},
		{"an index that levels", func(context.Context, domain.Vault, []string) error { return nil }, false},
		{"an index that cannot", func(context.Context, domain.Vault, []string) error { return errors.New("busy") }, true},
	} {
		cfg, db, v := walkedVault(t, map[string]string{"Sanskrit.md": presetFile})
		held := cfg.Flashcards(db.Queries(), db.Links(), db.Problems(), one.index)

		read, err := held.Presets.Read(t.Context(), v, "Sanskrit.md")
		if err != nil {
			t.Fatal(err)
		}
		moved := read.Settings
		moved.MinutesADay = 35
		_, err = held.Presets.Save(t.Context(), v, "Sanskrit.md", moved, read.Fingerprint)
		if got := errors.Is(err, note.ErrUnlevelled); got != one.isFailed {
			t.Errorf("%s: saving said %v", one.what, err)
		}
		if !strings.Contains(readFile(t, v, "Sanskrit.md"), "35") {
			t.Errorf("%s: the preset was not written", one.what)
		}
	}
}

// What a deck links to is read through the installation's own readers and index.
func TestTheNeighbourhoodOfADeckIsReadWithTheInstallationsOwnParts(t *testing.T) {
	cfg, db, v := walkedVault(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	})
	shown := cfg.ShowNeighbourhood(note.NewShowLinks(db.Links()), db.Queries(), note.NewRead(cfg.VaultReaders()))

	got, err := shown.Execute(t.Context(), v, "decks/Birds.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Notes) != 1 || got.Notes[0].Path != "Migration.md" ||
		!strings.Contains(got.Notes[0].Body, "Birds go south") {
		t.Errorf("joined to %+v", got.Notes)
	}
}
