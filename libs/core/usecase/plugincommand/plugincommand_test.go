package plugincommand_test

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
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
		if mask.Matches(one) {
			out = append(out, extension[1:])
		}
	}
	slices.Sort(out)
	return out
}

// vault is the vault every command of these tests runs over.
func getVault() domain.Vault { return domain.Vault{ID: "v1", Name: "v1"} }

func getTypes() fileTypes {
	return fileTypes{".pdf": "application/pdf", ".epub": "application/epub+zip", ".mp3": "audio/mpeg", ".wav": "audio/vnd.wave"}
}

// declaredCommands lists the commands it is given and records every call it is given to run.
type declaredCommands struct {
	entries []plugincommand.Entry
	calls   []plugincommand.Call
}

func (d *declaredCommands) ListSlots() []plugincommand.SlotEntry { return nil }

func (d *declaredCommands) ListHandlers() []plugincommand.Entry { return nil }

func (d *declaredCommands) ListCommands() []plugincommand.Entry { return d.entries }

func (d *declaredCommands) GetParts(string) plugincommand.Parts {
	return plugincommand.Parts{HasBackend: true}
}

func (d *declaredCommands) RunCommand(_ context.Context, call plugincommand.Call) (plugincommand.Result, error) {
	d.calls = append(d.calls, call)
	return plugincommand.Result{Message: "ran " + call.Command, OpenedPath: call.Path}, nil
}

func newDeclared() *declaredCommands {
	return &declaredCommands{entries: []plugincommand.Entry{
		{PluginID: "md.numen.a", Command: plugincommand.Command{
			ID: "convert", Title: "Convert", When: "target.contentType == 'application/pdf'",
			Arguments: []plugincommand.Argument{
				{Name: "pages", Title: "Pages", Type: plugincommand.ArgumentNumber, IsRequired: true},
				{Name: "again", Title: "Again", Type: plugincommand.ArgumentBoolean},
			},
		}},
		{PluginID: "md.numen.a", Command: plugincommand.Command{ID: "tidy", Title: "Tidy"}},
		{PluginID: "md.numen.b", Command: plugincommand.Command{ID: "tidy", Title: "Tidy as well"}},
	}}
}

func TestRunCommandNormalizesBooleanArguments(t *testing.T) {
	declared := newDeclared()

	result, err := plugincommand.NewCommands(declared, getTypes(), nil).Run.Execute(t.Context(), getVault(), plugincommand.Call{
		PluginID: "md.numen.a", Command: "convert", Path: "books/Some.PDF",
		Arguments: map[string]string{"pages": "12", "again": "1"},
	})

	if err != nil || result.Message != "ran convert" || result.OpenedPath != "books/Some.PDF" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
	call := declared.calls[0]
	if call.Path != "books/Some.PDF" || call.Arguments["pages"] != "12" || call.Arguments["again"] != "true" {
		t.Errorf("the plugin received %+v", call)
	}
}

func TestACallThatDoesNotMeetItsDeclarationRunsNothing(t *testing.T) {
	cases := map[string]plugincommand.Call{
		"a command over a file given none": {
			PluginID: "md.numen.a", Command: "convert", Arguments: map[string]string{"pages": "1"},
		},
		"a file of a type it does not run over": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.epub", Arguments: map[string]string{"pages": "1"},
		},
		"a required argument left out":        {PluginID: "md.numen.a", Command: "convert", Path: "a.pdf"},
		"an argument it does not take": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.pdf",
			Arguments: map[string]string{"pages": "1", "colour": "red"},
		},
		"a number that is none": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.pdf", Arguments: map[string]string{"pages": "twelve"},
		},
		"a boolean that is none": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.pdf",
			Arguments: map[string]string{"pages": "1", "again": "maybe"},
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			declared := newDeclared()

			_, err := plugincommand.NewCommands(declared, getTypes(), nil).Run.Execute(t.Context(), getVault(), call)

			if !errors.Is(err, plugincommand.ErrBadArguments) {
				t.Errorf("the call answered %v, want ErrBadArguments", err)
			}
			if len(declared.calls) != 0 {
				t.Errorf("the plugin received %+v", declared.calls)
			}
		})
	}
}

func TestACommandIsRunInThePluginTheCallNames(t *testing.T) {
	declared := newDeclared()
	commands := plugincommand.NewCommands(declared, getTypes(), nil)

	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.b", Command: "tidy"}); err != nil {
		t.Fatal(err)
	}
	_, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.b", Command: "convert"})

	if declared.calls[0].PluginID != "md.numen.b" {
		t.Errorf("the run went to %s", declared.calls[0].PluginID)
	}
	if !errors.Is(err, plugincommand.ErrUnknownCommand) {
		t.Errorf("a command the plugin does not declare is %v, want ErrUnknownCommand", err)
	}
}

func TestACommandIsFoundByThePluginOfTheLowestID(t *testing.T) {
	found, isFound := plugincommand.NewCommands(newDeclared(), getTypes(), nil).List.Find("tidy")

	if !isFound || found.PluginID != "md.numen.a" {
		t.Errorf("found %+v, %v", found, isFound)
	}
}

func TestAnInstallationWithNoPluginHostListsNoCommandAndRunsNone(t *testing.T) {
	commands := plugincommand.NewCommands(nil, nil, nil)

	if entries := commands.List.Execute(); len(entries) != 0 {
		t.Errorf("got %+v", entries)
	}
	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{Command: "tidy"}); !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("a run answered %v, want unavailable", err)
	}
}

func TestTheContentTypesAreListedByExtension(t *testing.T) {
	got := plugincommand.NewCommands(newDeclared(), getTypes(), nil).List.ListContentTypes()

	want := map[string]string{"pdf": "application/pdf", "epub": "application/epub+zip", "mp3": "audio/mpeg", "wav": "audio/vnd.wave"}
	if !maps.Equal(got, want) {
		t.Errorf("ListContentTypes() = %v; want %v", got, want)
	}
}

func TestACommandWithoutAConditionRunsOverAnyPathOrNone(t *testing.T) {
	declared := newDeclared()
	commands := plugincommand.NewCommands(declared, getTypes(), nil)

	for _, path := range []string{"", "talks", "a.md"} {
		if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.a", Command: "tidy", Path: path}); err != nil {
			t.Errorf("%q answered %v", path, err)
		}
	}
}

func TestRunCommandRejectsTheRootFolderForACommandOverAFile(t *testing.T) {
	declared := &declaredCommands{entries: []plugincommand.Entry{{PluginID: "md.numen.a", Command: plugincommand.Command{
		ID: "proofreading.create", Title: "Proofread", When: "target.isFile || target.isNote",
	}}}}
	commands := plugincommand.NewCommands(declared, getTypes(), nil)

	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.a", Command: "proofreading.create"}); !errors.Is(err, plugincommand.ErrBadArguments) {
		t.Errorf("the root folder answered %v, want ErrBadArguments", err)
	}
	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.a", Command: "proofreading.create", Path: "talks/a.mp3"}); err != nil {
		t.Errorf("a file answered %v", err)
	}
	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.a", Command: "proofreading.create", Path: "README"}); err != nil {
		t.Errorf("a path of no known type, which the core cannot tell from a folder, answered %v", err)
	}
}

// noteFiles opens a vault whose files are the map.
type noteFiles map[string]string

func (n noteFiles) Open(domain.Vault) (vaultport.VaultReader, error) {
	return noteReader{files: n}, nil
}

type noteReader struct {
	vaultport.VaultReader
	files noteFiles
}

func (r noteReader) Read(_ context.Context, path string) ([]byte, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(raw), nil
}

func TestACommandOverNotesOfATypeRunsOverThoseNotesAlone(t *testing.T) {
	declared := &declaredCommands{entries: []plugincommand.Entry{
		{PluginID: "md.numen.links", Command: plugincommand.Command{ID: "fetch", Title: "Fetch", When: "target.noteType == 'link'"}},
	}}
	files := noteFiles{
		"links/talk.md":  "---\ntype: link\nurl: https://video.example/talk\n---\n",
		"notes/plain.md": "# A plain note\n",
	}
	commands := plugincommand.NewCommands(declared, getTypes(), files)

	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.links", Command: "fetch", Path: "links/talk.md"}); err != nil {
		t.Fatalf("a link note was refused: %v", err)
	}
	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.links", Command: "fetch", Path: "notes/plain.md"}); !errors.Is(err, plugincommand.ErrBadArguments) {
		t.Fatalf("a plain note was run over: %v", err)
	}
	if _, err := commands.Run.Execute(t.Context(), getVault(), plugincommand.Call{PluginID: "md.numen.links", Command: "fetch"}); !errors.Is(err, plugincommand.ErrBadArguments) {
		t.Fatalf("a command over notes ran over no file: %v", err)
	}
	if len(declared.calls) != 1 || declared.calls[0].Path != "links/talk.md" {
		t.Errorf("ran %+v, want the link note alone", declared.calls)
	}
}
