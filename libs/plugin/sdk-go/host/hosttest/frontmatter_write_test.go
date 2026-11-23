package hosttest_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

const riverNote = "---\ntitle: River\ntags: [lake]\nstatus: open\n---\nThe river runs.\n"

func TestWriteFrontmatterSetsAKeyAndKeepsTheOtherKeysAndTheBody(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("river.md", riverNote)

	results, err := v.WriteFrontmatter(t.Context(), []host.FrontmatterWrite{
		{Path: "river.md", Values: map[string]any{"tags": []string{"river", "lake"}}},
	})

	if err != nil || len(results) != 1 || results[0].Err != nil {
		t.Fatalf("wrote %+v (%v), want one written note", results, err)
	}
	page, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{"river.md"}})
	if err != nil {
		t.Fatal(err)
	}
	got := page.Notes[0].Values
	if !reflect.DeepEqual(got["tags"], []any{"river", "lake"}) || got["title"] != "River" || got["status"] != "open" {
		t.Errorf("the frontmatter is %v, want the new tags beside the title and status", got)
	}
	if content, _ := v.ReadContent("river.md"); !strings.HasSuffix(content, "---\nThe river runs.\n") {
		t.Errorf("the note is %q, want the body kept", content)
	}
	if now, err := v.StatFile(t.Context(), "river.md"); err != nil || results[0].Stat.Size != now.Size || !results[0].Stat.ModTime.Equal(now.ModTime) {
		t.Errorf("the result has the stat %+v, want the stat of the note now, %+v (%v)", results[0].Stat, now, err)
	}
}

func TestWriteFrontmatterRemovesAKeyWhoseValueIsNull(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("river.md", riverNote)

	if _, err := v.WriteFrontmatter(t.Context(), []host.FrontmatterWrite{{Path: "river.md", Values: map[string]any{"tags": nil}}}); err != nil {
		t.Fatal(err)
	}

	page, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{"river.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, hasTags := page.Notes[0].Values["tags"]; hasTags || page.Notes[0].Values["status"] != "open" {
		t.Errorf("the frontmatter is %v, want status kept and tags removed", page.Notes[0].Values)
	}
}

func TestWriteFrontmatterShowsOnTheFeedAsAChangedFrontmatter(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("river.md", riverNote)
	start := readFeed(t, v, host.FileChangeCursor{}).Next

	if _, err := v.WriteFrontmatter(t.Context(), []host.FrontmatterWrite{{Path: "river.md", Values: map[string]any{"tags": "river"}}}); err != nil {
		t.Fatal(err)
	}

	if got := describe(readFeed(t, v, start).Changes); !slices.Contains(got, "river.md note.frontmatterChanged") {
		t.Errorf("the feed holds %v, want the change of the frontmatter of river.md", got)
	}
}

func TestWriteFrontmatterRefusesAStaleNoteAndWritesTheOthers(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("river.md", riverNote)
	v.WriteFile("lake.md", "---\ntitle: Lake\n---\nStill.\n")
	stale, err := v.StatFile(t.Context(), "river.md")
	if err != nil {
		t.Fatal(err)
	}
	v.WriteFile("river.md", riverNote+"More water.\n")

	results, err := v.WriteFrontmatter(t.Context(), []host.FrontmatterWrite{
		{Path: "river.md", Values: map[string]any{"tags": "river"}, Baseline: stale},
		{Path: "lake.md", Values: map[string]any{"tags": "lake"}},
		{Path: "gone.md", Values: map[string]any{"tags": "gone"}},
	})

	if err != nil || len(results) != 3 {
		t.Fatalf("wrote %+v (%v), want one result for each note", results, err)
	}
	if !errors.Is(results[0].Err, host.ErrStale) {
		t.Errorf("the stale note returned %v, want ErrStale", results[0].Err)
	}
	if results[1].Err != nil {
		t.Errorf("the lake note failed with %v, a failed note costs the others nothing", results[1].Err)
	}
	if results[2].Err == nil {
		t.Error("a note that is not there was written")
	}
	if content, _ := v.ReadContent("river.md"); strings.Contains(content, "tags: river") {
		t.Errorf("the stale note was written: %q", content)
	}
}
