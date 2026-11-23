package index

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
)

func TestParseTagsReadsAListAndASingleValue(t *testing.T) {
	for name, c := range map[string]struct {
		value any
		want  []string
	}{
		"a list":                        {[]any{"port", "sea"}, []string{"port", "sea"}},
		"a single value":                {"port", []string{"port"}},
		"a hash is dropped":             {[]any{"#port"}, []string{"port"}},
		"spaces are trimmed":            {[]any{"  port  "}, []string{"port"}},
		"case repeats are dropped":      {[]any{"Port", "port"}, []string{"Port"}},
		"a nested tag is one tag":       {[]any{"projects/web"}, []string{"projects/web"}},
		"segments are trimmed":          {"projects / web", []string{"projects/web"}},
		"a number is a tag":             {[]any{float64(2026)}, []string{"2026"}},
		"empty and other items are out": {[]any{"", "#", true, nil}, nil},
		"nothing":                       {nil, nil},
	} {
		if got := parseTags(c.value); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: parseTags(%v) = %q, want %q", name, c.value, got, c.want)
		}
	}
}

func TestListFieldsCountsTheTagsOfNotesCaseInsensitivelyAndReplacesThemWhenTheFrontmatterChanges(t *testing.T) {
	f := newFixture(map[string]string{
		"a.md": "---\ntags: [Port, sea]\n---\n- [ ] one",
		"b.md": "---\ntags: port\n---\nbody",
		"c.md": "no frontmatter",
	})
	if err := f.index.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got, want := formatCounts(f.index.ListFields().Tags), []string{"Port=2", "sea=1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}

	f.vault.WriteFile("a.md", "---\ntags: [wharf]\n---\n- [ ] one")
	if err := f.index.Refresh(t.Context(), []string{"a.md"}, nil); err != nil {
		t.Fatal(err)
	}

	if got, want := formatCounts(f.index.ListFields().Tags), []string{"Port=1", "wharf=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tags = %v, want %v", got, want)
	}
}

func TestTagsOfANoteAreNotTagsOfItsTasks(t *testing.T) {
	f := newFixture(map[string]string{"a.md": "---\ntags: [port]\n---\n- [ ] one"})
	if err := f.index.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}

	got := f.index.List(today, filter.Filter{}, 10).Tasks

	if len(got) != 1 || len(got[0].Contexts) != 0 || len(got[0].Projects) != 0 {
		t.Errorf("tasks = %+v, want one without projects and contexts", got)
	}
}

func TestAWarmStartKeepsTheTagsAndReadsAFrontmatterChangeWithNoTaskChange(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "---\ntags: [port]\n---\n- [ ] one\n")
	vault.WriteFile("b.md", "---\ntags: [sea]\n---\n- [ ] two\n")
	snapshots, clock := new(memorySnapshots), new(timers)
	first, _ := persisted(vault, snapshots, clock)
	start(t, first)
	flush(t, first)

	vault.WriteFile("b.md", "---\ntags: [quay]\n---\n- [ ] two\n")
	second, files := persisted(vault, snapshots, clock)
	start(t, second)

	if got := files.readsOf("a.md"); got != 0 {
		t.Errorf("reads of the unchanged note = %d, want 0", got)
	}
	if got, want := formatCounts(second.ListFields().Tags), []string{"port=1", "quay=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tags = %v, want %v", got, want)
	}
}

func TestTagsFollowANoteThatIsRemovedOrMoved(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("a.md", "---\ntags: [port]\n---\nbody")
	vault.WriteFile("b.md", "---\ntags: [sea]\n---\nbody")
	snapshots, clock := new(memorySnapshots), new(timers)
	one, _ := persisted(vault, snapshots, clock)
	start(t, one)

	vault.Remove("b.md")
	if _, err := vault.MoveNotes(t.Context(), []string{"a.md"}, "inbox"); err != nil {
		t.Fatal(err)
	}
	flush(t, one)

	if got, want := formatCounts(one.ListFields().Tags), []string{"port=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tags = %v, want %v", got, want)
	}
}
