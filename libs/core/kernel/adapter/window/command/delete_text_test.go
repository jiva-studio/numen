package command_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

func (v *versionItems) ListEveryVersion(context.Context, domain.Vault, string) ([]text.VersionItem, error) {
	return []text.VersionItem{
		{Version: "org.example.ears/ears", Kind: "transcript", IsLive: true},
		{Version: "org.example.viewer/edit-1", Kind: "transcript"},
	}, nil
}

func (v *versionItems) DeleteVersion(_ context.Context, _ domain.Vault, path string, version domain.TextVersionID) error {
	v.deleted = append(v.deleted, path+" "+string(version))
	return nil
}

func TestListCommandItems_OfDeleteTextIsEveryVersionOfTheFile(t *testing.T) {
	list := listItemsOfOne(t, newItemsHandler(&versionItems{}), "", "app:version.delete", "talks/a.mp3")

	var titles []string
	for _, one := range list.GetItems() {
		titles = append(titles, one.GetId()+" "+one.GetTitle())
	}
	want := []string{"org.example.ears/ears transcript (current)", "org.example.viewer/edit-1 transcript"}
	if list.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED || !reflect.DeepEqual(titles, want) {
		t.Errorf("got %v %v, want %v", list.GetError(), titles, want)
	}
}

func TestRunPluginCommand_OfDeleteTextDeletesTheVersionPicked(t *testing.T) {
	items := &versionItems{}

	res, err := newItemsHandler(items).RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{
		Command: "app:version.delete", Path: "talks/a.mp3", ItemId: "org.example.viewer/edit-1",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED ||
		!reflect.DeepEqual(items.deleted, []string{"talks/a.mp3 org.example.viewer/edit-1"}) {
		t.Errorf("got %v, deleted %v", res.Msg.GetError(), items.deleted)
	}
}

func TestRunPluginCommand_OfDeleteTextNeedsAnItem(t *testing.T) {
	items := &versionItems{}

	res, err := newItemsHandler(items).RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{
		Command: "app:version.delete", Path: "talks/a.mp3",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS || len(items.deleted) != 0 {
		t.Errorf("got %v, deleted %v", res.Msg.GetError(), items.deleted)
	}
}

// namedVersions offers the versions of two plugins, one declaring the names of its kinds.
type namedVersions struct{ versionItems }

func (namedVersions) ListEveryVersion(context.Context, domain.Vault, string) ([]text.VersionItem, error) {
	made := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)
	return []text.VersionItem{
		{Version: "org.example.ears/ears", PluginID: "org.example.ears", Kind: "transcript", IsLive: true, CreateTime: made},
		{Version: "org.example.viewer/edit-1", PluginID: "org.example.viewer", Kind: "edit", CreateTime: made},
	}, nil
}

type namingPlugins struct{ pluginCommands }

func (namingPlugins) GetParts(pluginID string) plugin.Parts {
	if pluginID != "org.example.ears" {
		return plugin.Parts{}
	}
	return plugin.Parts{Name: "Transcription", TextKinds: map[string]string{"transcript": "Transcript"}}
}

func TestListCommandItems_OfDeleteTextNamesAVersionByItsDeclaredKindAndDetailsItsPluginAndDate(t *testing.T) {
	handler := newHandler(newScenarios(namingPlugins{}, namingPlugins{}, fileTypes{}, nil, nil).withVersions(&namedVersions{}))

	list := listItemsOfOne(t, handler, "", "app:version.delete", "talks/a.mp3")

	var got [][2]string
	for _, one := range list.GetItems() {
		got = append(got, [2]string{one.GetTitle(), one.GetDetail()})
	}
	want := [][2]string{{"Transcript (current)", "Transcription · 2026-11-08"}, {"edit", "org.example.viewer · 2026-11-08"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
