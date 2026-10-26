package settings

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// pathDialog answers every picker with the path it holds, and keeps what it was asked.
type pathDialog struct {
	path     string
	isFolder bool
	accept   []string
}

func (d *pathDialog) ChooseFolder(context.Context, string, string) (string, bool, error) {
	d.isFolder = true
	return d.path, d.path != "", nil
}

func (d *pathDialog) ChooseFile(_ context.Context, _, _ string, accept []string) (string, bool, error) {
	d.accept = accept
	return d.path, d.path != "", nil
}

func listPathPlugins() []PluginInfo {
	return []PluginInfo{{ID: "a.b", Settings: []PluginSetting{
		{Key: "model", Type: "enum"},
		{Key: "dictionary", Type: pluginport.SettingTypeFile, Accept: []string{".txt"}},
		{Key: "cache", Type: pluginport.SettingTypeFolder},
	}}}
}

// The picker is the one of the setting's kind, and the path chosen is written at the setting.
func TestChoosePluginPathWritesTheSetting(t *testing.T) {
	dialog := &pathDialog{path: "/home/a/letters.txt"}
	var written []Setting
	choose := NewChoosePluginPath(dialog, listPathPlugins, func(content []Setting) error {
		written = content
		return nil
	})
	chosen, err := choose.Execute(t.Context(), "a.b", "dictionary")
	if err != nil || !chosen.IsChosen || chosen.Path != dialog.path {
		t.Fatalf("chosen = %+v, %v", chosen, err)
	}
	if dialog.isFolder || !slices.Equal(dialog.accept, []string{".txt"}) {
		t.Errorf("the picker was a folder picker or showed %v", dialog.accept)
	}
	if len(written) != 1 || !slices.Equal(written[0].Path, []string{"plugins", "a.b", "dictionary"}) || written[0].JSON != `"/home/a/letters.txt"` {
		t.Errorf("written = %+v", written)
	}

	if _, err := choose.Execute(t.Context(), "a.b", "cache"); err != nil || !dialog.isFolder {
		t.Errorf("the folder setting was not given a folder picker: %v", err)
	}
}

// A closed picker writes nothing, and a setting that holds no path is not offered one.
func TestChoosePluginPathWritesNothingWithoutAPath(t *testing.T) {
	choose := NewChoosePluginPath(&pathDialog{}, listPathPlugins, func([]Setting) error {
		t.Error("something was written")
		return nil
	})
	if chosen, err := choose.Execute(t.Context(), "a.b", "dictionary"); err != nil || chosen.IsChosen {
		t.Errorf("chosen = %+v, %v", chosen, err)
	}
	if _, err := choose.Execute(t.Context(), "a.b", "model"); !errors.Is(err, ErrNoPathSetting) {
		t.Errorf("err = %v, want ErrNoPathSetting", err)
	}
}
