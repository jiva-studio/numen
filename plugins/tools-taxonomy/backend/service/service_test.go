package service_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

type plugin struct {
	vault *hosttest.Vault
	app   *service.App
}

func newPlugin(t *testing.T, files map[string]string) plugin {
	t.Helper()
	v := hosttest.New("garden")
	for path, content := range files {
		v.WriteFile(path, content)
	}
	app, err := service.NewApp(func(id domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: v.With(id), Feed: v.With(id), Store: v.Store().With(id)}
	})
	if err != nil {
		t.Fatal(err)
	}
	return plugin{vault: v, app: app}
}

func (p plugin) listTags(t *testing.T, input string) service.TagListOutput {
	t.Helper()
	res, err := p.app.Tools().InvokeTool(t.Context(), tools.Call{Name: "tag_list", Arguments: []byte(input), VaultID: "garden"})
	if err != nil {
		t.Fatal(err)
	}
	result := res.GetResult()
	if result == nil || result.IsError {
		t.Fatalf("tag_list answered %+v", res)
	}
	raw := result.Structured
	if len(raw) == 0 {
		raw = []byte(result.Text)
	}
	var out service.TagListOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTagListReturnsTheTreeWithTheNotesOfEachTag(t *testing.T) {
	p := newPlugin(t, map[string]string{
		"a.md": "---\ntags: [Projects/Web, '#projects']\n---\n",
		"b.md": "---\ntags: projects/web/api\n---\n",
		"c.md": "---\ntags: [River]\n---\n",
		"d.md": "# no tags\n",
	})

	got := p.listTags(t, `{}`)

	want := []service.TagEntry{
		{Tag: "Projects", Name: "Projects", Depth: 0, Count: 2},
		{Tag: "Projects/Web", Name: "Web", Depth: 1, Count: 2},
		{Tag: "Projects/Web/api", Name: "api", Depth: 2, Count: 1},
		{Tag: "River", Name: "River", Depth: 0, Count: 1},
	}
	if !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %+v\nwant %+v", got.Tags, want)
	}
}

func TestTagListWithATagListsOnlyThatTagAndItsNestedTags(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: [Projects/Web, River]\n---\n"})

	got := p.listTags(t, `{"under":"projects"}`)

	want := []service.TagEntry{{Tag: "Projects", Name: "Projects", Depth: 0, Count: 1}, {Tag: "Projects/Web", Name: "Web", Depth: 1, Count: 1}}
	if !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %+v", got.Tags)
	}
}

func (p plugin) deliver(t *testing.T, event string) {
	t.Helper()
	if err := p.app.HandleEvent(t.Context(), event); err != nil {
		t.Fatal(err)
	}
}

func readStoredTags(t *testing.T, p plugin) string {
	t.Helper()
	content, err := p.vault.Store().With("garden").Read(t.Context(), tagcache.FileName)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestAFilesChangedEventReadsTheChangedTagsWithoutATrigger(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	p.deliver(t, "onFilesChanged:garden")
	if stored := readStoredTags(t, p); !strings.Contains(stored, "river") {
		t.Fatalf("the first event stored %s", stored)
	}

	p.vault.WriteFile("a.md", "---\ntags: lake\n---\n")
	p.deliver(t, "onFilesChanged:garden")

	stored := readStoredTags(t, p)
	if !strings.Contains(stored, "lake") || strings.Count(stored, "river") != 1 {
		t.Errorf("the event stored %s", stored)
	}
}

func TestAnEventOfAnotherKindReadsNothing(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n"})
	p.deliver(t, hosttest.GetIndexChangedEvent("garden"))
	if _, err := p.vault.Store().With("garden").Read(t.Context(), tagcache.FileName); err == nil {
		t.Error("an onIndexChanged event read the tags")
	}
}
