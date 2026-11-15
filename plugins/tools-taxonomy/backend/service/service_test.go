package service_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/tagcache"
)

type plugin struct {
	vault    *hosttest.Vault
	commands *command.Registry
	tools    *tools.Registry
}

func newPlugin(t *testing.T, files map[string]string) plugin {
	t.Helper()
	v := hosttest.New("garden")
	for path, content := range files {
		v.File(path, content)
	}
	commands, registry, err := service.NewRegistries(service.Host{
		Shown: v,
		Open: func(id domain.VaultID) tagcache.Vault {
			return tagcache.Vault{Frontmatter: v.With(id), Feed: v.With(id), Store: v.Store().With(id)}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plugin{vault: v, commands: commands, tools: registry}
}

func (p plugin) listTags(t *testing.T, input string) service.TagListOutput {
	t.Helper()
	res, err := p.tools.InvokeTool(t.Context(), tools.Call{Name: "tag_list", Arguments: []byte(input), VaultID: "garden"})
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

func TestTheUpdateCommandReadsTheChangedTagsOfTheNote(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n"})
	p.listTags(t, `{}`)

	p.vault.File("a.md", "---\ntags: lake\n---\n")
	if _, err := p.commands.RunCommand(t.Context(), command.Request{Command: service.UpdateCommand, Path: "a.md"}); err != nil {
		t.Fatal(err)
	}

	if got := p.listTags(t, `{}`); len(got.Tags) != 1 || got.Tags[0].Tag != "lake" {
		t.Errorf("tags = %+v", got.Tags)
	}
}

func TestTheUpdateCommandReadsTheChangedTagsOfEveryNoteOfATrigger(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: river\n---\n"})
	p.listTags(t, `{}`)

	p.vault.File("a.md", "---\ntags: lake\n---\n")
	p.vault.File("b.md", "---\ntags: lake\n---\n")
	if _, err := p.commands.RunCommand(t.Context(), command.Request{Command: service.UpdateCommand, Paths: []string{"a.md", "b.md"}}); err != nil {
		t.Fatal(err)
	}

	if got := p.listTags(t, `{}`); len(got.Tags) != 1 || got.Tags[0].Tag != "lake" || got.Tags[0].Count != 2 {
		t.Errorf("tags = %+v", got.Tags)
	}
}
