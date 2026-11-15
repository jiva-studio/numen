package index

import (
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// frontmatterReports is the reports a watcher of the frontmatter of a vault received, in order.
type frontmatterReports struct {
	mu      sync.Mutex
	reports [][]string
}

func (r *frontmatterReports) report(paths []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = append(r.reports, paths)
}

func (r *frontmatterReports) take() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.reports
	r.reports = nil
	return out
}

func watchFrontmatter(t *testing.T, db *DB, v domain.Vault) *frontmatterReports {
	t.Helper()
	reports := &frontmatterReports{}
	t.Cleanup(db.FrontmatterChanges().Watch(v.ID, reports.report))
	return reports
}

func saveNoteWithFrontmatter(t *testing.T, db *DB, v domain.Vault, path string, values map[string]any, body string) {
	t.Helper()
	n := domain.Note{
		FileStat:    domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: getWalkedTime()},
		Title:       path,
		Frontmatter: values,
		Body:        body,
	}
	if err := saveCut(t.Context(), db, v.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

func TestASaveThatChangesTheFrontmatterOfANoteReportsItsPath(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water"}}, "River")
	reports := watchFrontmatter(t, db, getFirstVault())

	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water", "delta"}}, "River")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/lake.md", map[string]any{"tags": "still"}, "Lake")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/plain.md", nil, "Plain")

	want := [][]string{{"notes/river.md"}, {"notes/lake.md"}}
	if got := reports.take(); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("reported %v, want %v", got, want)
	}
}

func TestASaveThatKeepsTheFrontmatterReportsNothing(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water"}}, "River")
	reports := watchFrontmatter(t, db, getFirstVault())

	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water"}}, "River flows into the sea")

	if got := reports.take(); len(got) != 0 {
		t.Errorf("reported %v for a note whose frontmatter stayed", got)
	}
}

func TestRemovingANoteReportsItOnlyWhenItHadFrontmatter(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water"}}, "River")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/plain.md", nil, "Plain")
	reports := watchFrontmatter(t, db, getFirstVault())

	if err := db.Notes().Remove(t.Context(), getFirstVault().ID, []string{"notes/river.md", "notes/plain.md", "notes/never.md"}); err != nil {
		t.Fatal(err)
	}

	want := [][]string{{"notes/river.md"}}
	if got := reports.take(); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("reported %v, want %v", got, want)
	}
}

func TestAChangeOfTheFrontmatterOfOneVaultIsNotReportedToAnother(t *testing.T) {
	db := openDB(t)
	reports := watchFrontmatter(t, db, getSecond())

	saveNoteWithFrontmatter(t, db, getFirstVault(), "notes/river.md", map[string]any{"tags": []any{"water"}}, "River")

	if got := reports.take(); len(got) != 0 {
		t.Errorf("the second vault received %v", got)
	}
}

func TestTheFrontmatterOfEveryNoteIsReadInPagesInPathOrder(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "c.md", map[string]any{"tags": []any{"c"}, "status": "done"}, "C")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "a.md", map[string]any{"tags": "a", "status": "open"}, "A")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "b.md", nil, "B")
	saveNoteWithFrontmatter(t, db, getSecond(), "a0.md", map[string]any{"tags": "other"}, "Other")

	first, err := db.NoteQueryReads().ListFrontmatter(t.Context(), getFirstVault().ID, frontmatter.Query{Keys: []string{"tags"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Path != "a.md" || first[0].Values["tags"] != "a" || first[0].Values["status"] != nil ||
		first[1].Path != "b.md" || len(first[1].Values) != 0 {
		t.Fatalf("the first page is %+v", first)
	}
	second, err := db.NoteQueryReads().ListFrontmatter(t.Context(), getFirstVault().ID, frontmatter.Query{After: "b.md", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Path != "c.md" || second[0].Values["status"] != "done" {
		t.Errorf("the second page is %+v", second)
	}
}

func TestTheFrontmatterOfNamedNotesOmitsAPathWithNoNote(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "a.md", map[string]any{"tags": "a"}, "A")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "b.md", map[string]any{"tags": "b"}, "B")

	got, err := db.NoteQueryReads().ListFrontmatter(t.Context(), getFirstVault().ID, frontmatter.Query{
		Paths: []string{"b.md", "gone.md", "a.md", "b.md"}, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "a.md" || got[1].Path != "b.md" || got[1].Values["tags"] != "b" {
		t.Errorf("read %+v, want a.md and b.md", got)
	}
}
