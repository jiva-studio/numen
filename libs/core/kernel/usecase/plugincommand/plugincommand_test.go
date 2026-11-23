package plugincommand_test

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// fileTypes gives a file the type of its lowercase extension.
type fileTypes map[string]domain.ContentType

func (f fileTypes) GetContentType(name string) domain.ContentType {
	if found, ok := f[strings.ToLower(path.Ext(name))]; ok {
		return found
	}
	return domain.UnknownContentType
}

func (f fileTypes) ListExtensions(mask domain.ContentType) []string {
	var out []string
	for extension, one := range f {
		if mask.IsMatch(one) {
			out = append(out, extension[1:])
		}
	}
	slices.Sort(out)
	return out
}

func getVault() domain.Vault { return domain.Vault{ID: "v1", Name: "v1"} }

func getTypes() fileTypes {
	return fileTypes{".pdf": "application/pdf", ".epub": "application/epub+zip", ".mp3": "audio/mpeg", ".wav": "audio/vnd.wave"}
}

// declaredCommands lists the commands it is given and records every call it is given to run.
type declaredCommands struct {
	entries []plugin.CommandEntry
	calls   []plugin.CommandCall
}

func (d *declaredCommands) ListKeybindings() []plugin.KeybindingEntry { return nil }

func (d *declaredCommands) ListSlots() []plugin.SlotEntry { return nil }

func (d *declaredCommands) ListHandlers() []plugin.CommandEntry { return nil }

func (d *declaredCommands) ListCommands() []plugin.CommandEntry { return d.entries }

func (d *declaredCommands) GetParts(string) plugin.Parts {
	return plugin.Parts{HasBackend: true}
}

func (d *declaredCommands) RunCommand(_ context.Context, call plugin.CommandCall) (plugin.CommandResult, error) {
	d.calls = append(d.calls, call)
	return plugin.CommandResult{Message: "ran " + call.Command, OpenedPath: call.Path}, nil
}

func newDeclaredCommands() *declaredCommands {
	return &declaredCommands{entries: []plugin.CommandEntry{
		{PluginID: "md.numen.a", Command: manifest.Command{
			ID: "convert", Title: "Convert", When: "target.contentType == 'application/pdf'",
			Arguments: []manifest.CommandArgument{
				{Name: "pages", Title: "Pages", Type: manifest.CommandArgumentNumber, IsRequired: true},
				{Name: "again", Title: "Again", Type: manifest.CommandArgumentBoolean},
			},
		}},
		{PluginID: "md.numen.a", Command: manifest.Command{ID: "tidy", Title: "Tidy"}},
		{PluginID: "md.numen.b", Command: manifest.Command{ID: "tidy", Title: "Tidy as well"}},
	}}
}

func TestACommandIsFoundByThePluginOfTheLowestID(t *testing.T) {
	found, isFound := newUseCases(newDeclaredCommands(), newDeclaredCommands(), getTypes()).List.Find("tidy")

	if !isFound || found.PluginID != "md.numen.a" {
		t.Errorf("found %+v, %v", found, isFound)
	}
}

func TestTheContentTypesAreListedByExtension(t *testing.T) {
	got := newUseCases(newDeclaredCommands(), newDeclaredCommands(), getTypes()).Listing.Execute().ContentTypeByExtension

	want := map[string]string{"pdf": "application/pdf", "epub": "application/epub+zip", "mp3": "audio/mpeg", "wav": "audio/vnd.wave"}
	if !maps.Equal(got, want) {
		t.Errorf("ContentTypes.Execute() = %v; want %v", got, want)
	}
}

func TestAConditionOverTheFileIsReportedAsTakingAFile(t *testing.T) {
	for _, when := range []string{"target.isFile || target.isNote", "version.isLive && version.kind != ''"} {
		declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "md.numen.a", Command: manifest.Command{
			ID: "proofreading.create", Title: "Proofread", When: when,
		}}}}

		if !newUseCases(declared, declared, getTypes()).List.HasTargetVariable(declared.entries[0].Command) {
			t.Errorf("%q was reported as taking no file", when)
		}
	}
}
