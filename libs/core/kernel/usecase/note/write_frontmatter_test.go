package note_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
	"github.com/jiva-studio/numen/libs/shared/note/markdown"
)

const riverNote = "---\n" +
	"# kept as written\n" +
	"title: River\n" +
	"mine: keep me verbatim\n" +
	"tags:\n" +
	"  - lake\n" +
	"---\n" +
	"\n" +
	"The river runs.\n"

func (c changeFixture) writeFrontmatter() note.WriteFrontmatter {
	return note.NewWriteFrontmatter(
		vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, c.index, testsupport.NewClock())
}

func (c changeFixture) statOf(t *testing.T, path string) domain.FileStat {
	t.Helper()
	reader, err := vaultreader.Readers{}.Open(c.vault)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := reader.StatFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	stat.Hash = domain.HashContent([]byte(c.read(t, path)))
	return stat
}

func readKeys(t *testing.T, raw string) *markdown.Document {
	t.Helper()
	doc, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestWriteFrontmatterSetsAListAndKeepsTheOtherKeysAndTheBodyAsTheyWere(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": []string{"lake", "river"}}},
	})

	if err != nil || len(written) != 1 || written[0].Err != nil {
		t.Fatalf("Execute returned %+v (%v), want one written note", written, err)
	}
	want := strings.Replace(riverNote, "  - lake\n", "  - lake\n  - river\n", 1)
	if body := withoutStampedID(c.read(t, "River.md")); body != want {
		t.Errorf("the note without its stamped id is\n%q\nwant\n%q", body, want)
	}
	if now := c.statOf(t, "River.md"); written[0].Stat.Size != now.Size || !written[0].Stat.ModTime.Equal(now.ModTime) {
		t.Errorf("the result has the stat %+v, want the stat of the note now, %+v", written[0].Stat, now)
	}
}

func TestWriteFrontmatterWritesATextANumberAndABooleanAsTheirOwnTypes(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"status": "open", "rating": float64(3), "starred": true}},
	})

	if err != nil || len(written) != 1 || written[0].Err != nil {
		t.Fatalf("Execute returned %+v (%v), want one written note", written, err)
	}
	body := c.read(t, "River.md")
	for _, line := range []string{"status: open\n", "rating: 3\n", "starred: true\n"} {
		if !strings.Contains(body, line) {
			t.Errorf("the note has no line %q:\n%s", line, body)
		}
	}
}

func TestWriteFrontmatterRemovesKeysWithNilValues(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": nil, "mine": nil, "absent": nil}},
	})

	if err != nil || len(written) != 1 || written[0].Err != nil {
		t.Fatalf("Execute returned %+v (%v), want one written note", written, err)
	}
	want := "---\n# kept as written\ntitle: River\n---\n\nThe river runs.\n"
	if body := withoutStampedID(c.read(t, "River.md")); body != want {
		t.Errorf("the note without its stamped id is\n%q\nwant\n%q", body, want)
	}
}

func TestWriteFrontmatterAddsABlockToANoteWithoutFrontmatter(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Plain.md": "# Plain\n\nJust prose.\n"})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "Plain.md", Values: map[string]any{"tags": []string{"river"}}},
	})

	if err != nil || len(written) != 1 || written[0].Err != nil {
		t.Fatalf("Execute returned %+v (%v), want one written note", written, err)
	}
	body := c.read(t, "Plain.md")
	if names, _ := readKeys(t, body).List("tags"); !slices.Equal(names, []string{"river"}) || !strings.HasSuffix(body, "# Plain\n\nJust prose.\n") {
		t.Errorf("the note is %q, want the tag in a block and the prose kept", body)
	}
}

func TestWriteFrontmatterIndexesTheNotesItWrote(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote, "Lake.md": "# Lake\n"})
	var mu sync.Mutex
	var indexed []string
	spied := c
	spied.index = func(ctx context.Context, v domain.Vault, paths []string) error {
		mu.Lock()
		indexed = append(indexed, paths...)
		mu.Unlock()
		return c.index(ctx, v, paths)
	}

	if _, err := spied.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": "river"}},
	}); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(indexed, []string{"River.md"}) {
		t.Errorf("the index received %v, want River.md alone", indexed)
	}
}

func TestWriteFrontmatterKeepsAStaleNoteUnchanged(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", FileStat: domain.FileStat{Size: 1}, Values: map[string]any{"tags": "river"}},
	})

	if err != nil || len(written) != 1 || !errors.Is(written[0].Err, edit.ErrStale) {
		t.Fatalf("Execute returned %+v (%v), want one stale result", written, err)
	}
	if body := c.read(t, "River.md"); body != riverNote {
		t.Errorf("the stale note was written:\n%s", body)
	}
}

func TestWriteFrontmatterWritesANoteWhoseBaselineIsTheStateOnDisk(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", FileStat: c.statOf(t, "River.md"), Values: map[string]any{"title": "Rivers"}},
	})

	if err != nil || len(written) != 1 || written[0].Err != nil {
		t.Fatalf("Execute returned %+v (%v), want one written note", written, err)
	}
	if title, _ := readKeys(t, c.read(t, "River.md")).GetTitle(); title != "Rivers" {
		t.Errorf("the title is %q, want Rivers", title)
	}
}

func TestAFailedWriteFrontmatterCostsTheOtherNotesNothing(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote, "Lake.md": "# Lake\n"})

	written, err := c.writeFrontmatter().Execute(t.Context(), c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": []string{"river"}}},
		{Path: "Gone.md", Values: map[string]any{"tags": "gone"}},
		{Path: "Lake.md", Values: map[string]any{"tags": []string{"lake"}}},
	})

	if err != nil || len(written) != 3 {
		t.Fatalf("Execute returned %+v (%v), want one result for each note", written, err)
	}
	if written[0].Err != nil || written[2].Err != nil {
		t.Errorf("the notes that exist failed: %v, %v", written[0].Err, written[2].Err)
	}
	if written[1].Err == nil {
		t.Error("a note that is not there was written")
	}
	for _, path := range []string{"River.md", "Lake.md"} {
		if names, _ := readKeys(t, c.read(t, path)).List("tags"); len(names) == 0 {
			t.Errorf("%s has no tag after the call", path)
		}
	}
}

func TestAWriteFrontmatterCanceledBeforeTheFirstNoteWritesNothingAndIsNotAttempted(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	written, err := c.writeFrontmatter().Execute(ctx, c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": "river"}},
	})

	if err != nil {
		t.Fatalf("Execute returned %v, want the results and no error", err)
	}
	if len(written) != 1 || !errors.Is(written[0].Err, plugin.ErrNotAttempted) {
		t.Fatalf("Execute returned %+v, want one result with ErrNotAttempted", written)
	}
	if body := c.read(t, "River.md"); body != riverNote {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestAWriteFrontmatterCanceledAfterTheFirstNoteKeepsItsResultAndIsNotAttemptedForTheRest(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"River.md": riverNote, "Lake.md": "# Lake\n", "Sea.md": "# Sea\n"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	canceling := c
	canceling.index = func(ctx context.Context, v domain.Vault, paths []string) error {
		err := c.index(ctx, v, paths)
		cancel()
		return err
	}

	written, err := canceling.writeFrontmatter().Execute(ctx, c.vault, []note.FrontmatterWrite{
		{Path: "River.md", Values: map[string]any{"tags": "river"}},
		{Path: "Lake.md", Values: map[string]any{"tags": "lake"}},
		{Path: "Sea.md", Values: map[string]any{"tags": "sea"}},
	})

	if err != nil || len(written) != 3 {
		t.Fatalf("Execute returned %+v (%v), want one result for each note", written, err)
	}
	if written[0].Err != nil {
		t.Errorf("the first note failed with %v", written[0].Err)
	}
	for _, i := range []int{1, 2} {
		if !errors.Is(written[i].Err, plugin.ErrNotAttempted) {
			t.Errorf("note %d returned %v, want ErrNotAttempted", i, written[i].Err)
		}
	}
	if body := c.read(t, "Sea.md"); body != "# Sea\n" {
		t.Errorf("a note not attempted was written:\n%s", body)
	}
}

func withoutStampedID(body string) string {
	lines := strings.SplitAfter(body, "\n")
	return strings.Join(slices.DeleteFunc(lines, func(line string) bool { return strings.HasPrefix(line, "id: ") }), "")
}
