package note_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func manyRelatedVault() map[string]string {
	return map[string]string{
		"Entropy.md": noteWith("Entropy", "parent Physics", "child Heat", "child Work", "jump Order"),
		"Heat.md":    noteWith("Heat", "child Engine", "parent Entropy"),
		"Work.md":    noteWith("Work", "child Engine"),
		"Engine.md":  noteWith("Engine"),
		"Order.md":   noteWith("Order"),
		"Physics.md": noteWith("Physics", "child Optics"),
		"Optics.md":  noteWith("Optics"),
	}
}

func noteWith(title string, links ...string) string {
	body := "---\ntitle: " + title + "\n"
	if len(links) > 0 {
		body += "links:\n"
		for _, l := range links {
			role, to, _ := strings.Cut(l, " ")
			body += "  - to: " + to + "\n    role: " + role + "\n"
		}
	}
	return body + "---\n\n# " + title + "\n"
}

// recordingLinks keeps the paths of every call to the link batches, in call order.
type recordingLinks struct {
	noteport.LinkBatches
	linkCalls     [][]string
	backlinkCalls [][]string
	err           error
}

func (r *recordingLinks) ListLinksOf(ctx context.Context, v domain.VaultID, paths []string) (map[string][]domain.ResolvedLink, error) {
	r.linkCalls = append(r.linkCalls, slices.Clone(paths))
	if r.err != nil {
		return nil, r.err
	}
	return r.LinkBatches.ListLinksOf(ctx, v, paths)
}

func (r *recordingLinks) ListBacklinksOf(ctx context.Context, v domain.VaultID, paths []string) (map[string][]domain.ResolvedLink, error) {
	r.backlinkCalls = append(r.backlinkCalls, slices.Clone(paths))
	if r.err != nil {
		return nil, r.err
	}
	return r.LinkBatches.ListBacklinksOf(ctx, v, paths)
}

// recordingNotes keeps the paths of every query for the notes to display.
type recordingNotes struct {
	query.RefQueries
	calls [][]string
}

func (r *recordingNotes) GetNotes(ctx context.Context, v domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	r.calls = append(r.calls, slices.Clone(paths))
	return r.RefQueries.GetNotes(ctx, v, paths)
}

func listManyRelated(t *testing.T, paths []string, depth domain.RelationDepth) []domain.NoteGraph {
	t.Helper()
	db, v := newIndexedVault(t, manyRelatedVault())
	got, err := query.NewListRelatedNotes(db.NoteQueryReads(), db.NoteQueries()).Execute(t.Context(), v, paths, depth)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestOneGraphIsReturnedForEachPathInTheOrderOfTheRequest(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, manyRelatedVault())
	related := query.NewListRelatedNotes(db.NoteQueryReads(), db.NoteQueries())
	paths := []string{"Heat.md", "Entropy.md", "Order.md"}

	got, err := related.Execute(t.Context(), v, paths, domain.DepthNear)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != len(paths) {
		t.Fatalf("%d graphs for %d paths", len(got), len(paths))
	}
	for i, path := range paths {
		alone, err := related.Execute(t.Context(), v, []string{path}, domain.DepthNear)
		if err != nil {
			t.Fatal(err)
		}
		if got[i].Focus.Path != path {
			t.Errorf("graph %d is the graph of %q, want %q", i, got[i].Focus.Path, path)
		}
		if len(alone) != 1 || !reflect.DeepEqual(got[i], alone[0]) {
			t.Errorf("the graph of %s differs from the one asked alone:\n%+v\n%+v", path, got[i], alone)
		}
	}
}

func TestAnUnknownPathHasAnEmptyGraphAndTheOtherGraphsAreWhole(t *testing.T) {
	t.Parallel()

	got := listManyRelated(t, []string{"Entropy.md", "Missing.md", "Heat.md"}, domain.DepthNear)

	if len(got) != 3 {
		t.Fatalf("%d graphs for 3 paths", len(got))
	}
	if !reflect.DeepEqual(got[1], domain.NoteGraph{}) {
		t.Errorf("the graph of the unknown note is %+v, want an empty graph", got[1])
	}
	if got[0].Focus.Path != "Entropy.md" || len(got[0].Related) == 0 {
		t.Errorf("the graph before the unknown note is %+v", got[0])
	}
	if got[2].Focus.Path != "Heat.md" || len(got[2].Related) == 0 {
		t.Errorf("the graph after the unknown note is %+v", got[2])
	}
}

func TestDepthTwoGathersTheNotesRelatedToTheRelatedNotesOfEachPath(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, manyRelatedVault())
	related := query.NewListRelatedNotes(db.NoteQueryReads(), db.NoteQueries())
	paths := []string{"Entropy.md", "Physics.md"}

	got, err := related.Execute(t.Context(), v, paths, domain.DepthFar)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != len(paths) {
		t.Fatalf("%d graphs for %d paths", len(got), len(paths))
	}
	for i, path := range paths {
		alone, err := related.Execute(t.Context(), v, []string{path}, domain.DepthFar)
		if err != nil {
			t.Fatal(err)
		}
		if len(alone) != 1 || !slices.ContainsFunc(alone[0].Related, func(n domain.RelatedNote) bool { return n.RelatedTo != "" }) {
			t.Fatalf("%s has no note related to a related note, the fixture is wrong: %+v", path, alone)
		}
		if !reflect.DeepEqual(got[i], alone[0]) {
			t.Errorf("the graph of %s at depth 2 differs from the one asked alone", path)
		}
	}
}

func TestAPathAskedTwiceIsAnsweredTwice(t *testing.T) {
	t.Parallel()

	got := listManyRelated(t, []string{"Heat.md", "Heat.md"}, domain.DepthNear)

	if len(got) != 2 {
		t.Fatalf("%d graphs for 2 paths", len(got))
	}
	if got[0].Focus.Path != "Heat.md" || !reflect.DeepEqual(got[0], got[1]) {
		t.Errorf("the two graphs of one path differ or are empty:\n%+v\n%+v", got[0], got[1])
	}
}

func TestNoPathsGiveNoGraphs(t *testing.T) {
	t.Parallel()

	got := listManyRelated(t, nil, domain.DepthNear)

	if len(got) != 0 {
		t.Errorf("%d graphs for no path", len(got))
	}
}

func TestTheLinksOfEveryPathAreReadTogetherInOneCallForEachStep(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, manyRelatedVault())
	paths := []string{"Entropy.md", "Heat.md", "Work.md", "Order.md"}
	count := func(asked []string) (*recordingLinks, *recordingNotes) {
		links := &recordingLinks{LinkBatches: db.NoteQueryReads()}
		notes := &recordingNotes{RefQueries: db.NoteQueries()}
		if _, err := query.NewListRelatedNotes(links, notes).Execute(t.Context(), v, asked, domain.DepthFar); err != nil {
			t.Fatal(err)
		}
		return links, notes
	}

	batch, batchNotes := count(paths)

	if len(batch.linkCalls) == 0 || !slices.Equal(slices.Sorted(slices.Values(batch.linkCalls[0])), slices.Sorted(slices.Values(paths))) {
		t.Errorf("the first read of links asked for %v, want all of %v", batch.linkCalls, paths)
	}
	if len(batch.backlinkCalls) == 0 || !slices.Equal(slices.Sorted(slices.Values(batch.backlinkCalls[0])), slices.Sorted(slices.Values(paths))) {
		t.Errorf("the first read of backlinks asked for %v, want all of %v", batch.backlinkCalls, paths)
	}
	separate := 0
	for _, path := range paths {
		alone, _ := count([]string{path})
		separate += len(alone.linkCalls)
	}
	if len(batch.linkCalls) >= separate {
		t.Errorf("%d reads of links for all the paths, %d for the paths one by one", len(batch.linkCalls), separate)
	}
	seen := map[string]int{}
	for _, call := range batch.linkCalls {
		for _, path := range call {
			seen[path]++
		}
	}
	for path, n := range seen {
		if n != 1 {
			t.Errorf("the links of %s were read %d times, want once", path, n)
		}
	}
	if len(batchNotes.calls) != 1 {
		t.Errorf("the notes were queried %d times for all the paths, want once", len(batchNotes.calls))
	}
}

func TestAFailingIndexFailsTheWholeCall(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, manyRelatedVault())
	links := &recordingLinks{LinkBatches: db.NoteQueryReads(), err: errors.New("index closed")}

	_, err := query.NewListRelatedNotes(links, db.NoteQueries()).Execute(t.Context(), v, []string{"Entropy.md", "Heat.md"}, domain.DepthNear)

	if err == nil {
		t.Error("a closed index answered")
	}
}
