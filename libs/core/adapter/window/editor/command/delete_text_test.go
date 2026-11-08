package command_test

import (
	"context"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

func (v *versionItems) ListEveryVersion(context.Context, domain.Vault, string) ([]textversion.VersionItem, error) {
	return []textversion.VersionItem{
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
	want := []string{"org.example.ears/ears transcript (live)", "org.example.viewer/edit-1 transcript"}
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
