package settings_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const listPluginID = "md.numen.views.todos"

var listSettings = []manifest.Setting{
	{Key: "ignoredFolders", Type: manifest.SettingTypeList, Title: "Ignored folders", Default: []any{"templates"}},
	{Key: "label", Type: "string", Title: "Label"},
}

func declaredWithList(owner string) ([]manifest.Setting, bool) {
	switch owner {
	case settingsdomain.AppOwner:
		return settings.GetAppDeclaration(), true
	case listPluginID:
		return listSettings, true
	}
	return nil, false
}

func checkList(value any) error {
	return settings.CheckWrite(declaredWithList, nil, []string{listPluginID, "ignoredFolders"}, value)
}

func openList(t *testing.T, body string) settings.File {
	t.Helper()
	cfg, err := openAt(write(t, body))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	cfg.DropInvalid(declaredWithList, nil)
	return cfg
}

func TestAListOfStringsWrittenToAListSettingIsSavedAndReadBackAsTheSameList(t *testing.T) {
	written := []any{"templates", "archive/old"}
	if err := checkList(written); err != nil {
		t.Fatalf("the list was refused: %v", err)
	}
	path := write(t, `{}`)
	if err := settings.NewFiles().Save(path, newDocument(), settings.Setting{At: []string{listPluginID, "ignoredFolders"}, Written: written}); err != nil {
		t.Fatal(err)
	}

	cfg, err := openAt(path)
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}
	cfg.DropInvalid(declaredWithList, nil)

	got, found := cfg.GetValue(listPluginID, "ignoredFolders")
	if !found || !reflect.DeepEqual(got, written) {
		t.Errorf("the list read back is %#v, %v, want %#v", got, found, written)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("the notices are %q, want none", cfg.Notices)
	}
}

func TestAnEmptyListWrittenToAListSettingIsSaved(t *testing.T) {
	if err := checkList([]any{}); err != nil {
		t.Fatalf("the empty list was refused: %v", err)
	}

	cfg := openList(t, `{"`+listPluginID+`":{"ignoredFolders":[]}}`)

	got, found := cfg.GetValue(listPluginID, "ignoredFolders")
	if !found || !reflect.DeepEqual(got, []any{}) {
		t.Errorf("the list read back is %#v, %v, want an empty list", got, found)
	}
}

func TestAStringWrittenToAListSettingIsRefusedAsTheWrongType(t *testing.T) {
	err := checkList("templates,archive")

	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a string for a list gave %v, want ErrWrongType", err)
	}
}

func TestAListHoldingANumberWrittenToAListSettingIsRefusedAsTheWrongType(t *testing.T) {
	err := checkList([]any{"templates", 1.0})

	if !errors.Is(err, settings.ErrWrongType) {
		t.Errorf("a list holding a number gave %v, want ErrWrongType", err)
	}
}

func TestAListHoldingAnEmptyStringWrittenToAListSettingIsRefused(t *testing.T) {
	for _, item := range []string{"", "  "} {
		err := checkList([]any{"templates", item})

		if !errors.Is(err, settings.ErrInvalidListItem) {
			t.Errorf("%q: err = %v, want ErrInvalidListItem", item, err)
		}
	}
}

func TestAListHoldingOneValueTwiceWrittenToAListSettingIsRefused(t *testing.T) {
	err := checkList([]any{"templates", "archive", "templates"})

	if !errors.Is(err, settings.ErrInvalidListItem) {
		t.Errorf("err = %v, want ErrInvalidListItem", err)
	}
}

func TestAListSettingStoredAsAStringInTheFileIsReadAndIgnoredWithANotice(t *testing.T) {
	cfg := openList(t, `{"`+listPluginID+`":{"ignoredFolders":"templates","label":"tasks"}}`)

	if got, found := cfg.GetValue(listPluginID, "ignoredFolders"); found {
		t.Errorf("the string is in force as %#v", got)
	}
	if got, _ := cfg.GetValue(listPluginID, "label"); got != "tasks" {
		t.Errorf("the label in force is %v, want the valid value of the same plugin", got)
	}
	if !hasNotice(cfg.Notices, "ignoredFolders") {
		t.Errorf("the notices are %q", cfg.Notices)
	}
	declared := manifest.Manifest{ID: listPluginID, Settings: listSettings}
	inForce := declared.ResolveSettings(cfg.GetOwner(listPluginID))
	if !reflect.DeepEqual(inForce["ignoredFolders"], []any{"templates"}) {
		t.Errorf("the value in force is %#v, want the default", inForce["ignoredFolders"])
	}
}

func TestAPluginIsConfiguredWithAListSettingAsItsJSONText(t *testing.T) {
	cfg := openList(t, `{"`+listPluginID+`":{"ignoredFolders":["a","b"]}}`)

	declared := manifest.Manifest{ID: listPluginID, Settings: listSettings}
	values := plugindomain.FormatSettingValues(declared.ResolveSettings(cfg.GetOwner(listPluginID)))

	if values["ignoredFolders"] != `["a","b"]` {
		t.Errorf("the value configured is %q, want the JSON text of the list", values["ignoredFolders"])
	}
}
