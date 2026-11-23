package commandrun_test

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/commandrun"
)

// fileTypes gives a file the type of its lowercase extension.
type fileTypes map[string]domain.ContentType

func (f fileTypes) GetContentType(name string) domain.ContentType {
	if found, ok := f[strings.ToLower(path.Ext(name))]; ok {
		return found
	}
	return domain.UnknownContentType
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

func TestRunCommandNormalizesBooleanArguments(t *testing.T) {
	declared := newDeclaredCommands()

	result, err := newUseCases(declared, declared, getTypes(), nil, nil).Runner.Run(t.Context(), getVault(), plugin.CommandCall{
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
	cases := map[string]plugin.CommandCall{
		"a command over a file given none": {
			PluginID: "md.numen.a", Command: "convert", Arguments: map[string]string{"pages": "1"},
		},
		"a file of a type it does not run over": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.epub", Arguments: map[string]string{"pages": "1"},
		},
		"a required argument left out": {PluginID: "md.numen.a", Command: "convert", Path: "a.pdf"},
		"an argument it does not take": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.pdf",
			Arguments: map[string]string{"pages": "1", "color": "red"},
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
			declared := newDeclaredCommands()

			_, err := newUseCases(declared, declared, getTypes(), nil, nil).Runner.Run(t.Context(), getVault(), call)

			if !errors.Is(err, plugin.ErrBadArguments) {
				t.Errorf("the call answered %v, want ErrBadArguments", err)
			}
			if len(declared.calls) != 0 {
				t.Errorf("the plugin received %+v", declared.calls)
			}
		})
	}
}

func TestACommandIsRunInThePluginTheCallNames(t *testing.T) {
	declared := newDeclaredCommands()
	commands := newUseCases(declared, declared, getTypes(), nil, nil)

	if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.b", Command: "tidy"}); err != nil {
		t.Fatal(err)
	}
	_, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.b", Command: "convert"})

	if declared.calls[0].PluginID != "md.numen.b" {
		t.Errorf("the run went to %s", declared.calls[0].PluginID)
	}
	if !errors.Is(err, plugin.ErrUnknownCommand) {
		t.Errorf("a command the plugin does not declare is %v, want ErrUnknownCommand", err)
	}
}

func TestACommandWithoutAConditionRunsOverAnyPathOrNone(t *testing.T) {
	declared := newDeclaredCommands()
	commands := newUseCases(declared, declared, getTypes(), nil, nil)

	for _, path := range []string{"", "talks", "a.md"} {
		if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.a", Command: "tidy", Path: path}); err != nil {
			t.Errorf("%q answered %v", path, err)
		}
	}
}

func TestRunCommandRejectsEmptyPathForFileCommand(t *testing.T) {
	for _, when := range []string{"target.isFile || target.isNote", "version.isLive && version.kind != ''"} {
		declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "md.numen.a", Command: manifest.Command{
			ID: "proofreading.create", Title: "Proofread", When: when,
		}}}}
		commands := newUseCases(declared, declared, getTypes(), nil, nil)
		if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.a", Command: "proofreading.create"}); !errors.Is(err, plugin.ErrBadArguments) {
			t.Errorf("%q: the root folder answered %v, want ErrBadArguments", when, err)
		}
		if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.a", Command: "proofreading.create", Path: "talks/a.mp3"}); err != nil {
			t.Errorf("%q: a file answered %v", when, err)
		}
		if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.a", Command: "proofreading.create", Path: "README"}); err != nil {
			t.Errorf("%q: a path of no known type, which the core cannot tell from a folder, answered %v", when, err)
		}
	}
}

// noteFiles opens a vault whose files are the map.
type noteFiles map[string]string

func (n noteFiles) Open(domain.Vault) (edit.Reader, error) {
	return noteReader{files: n}, nil
}

type noteReader struct {
	edit.Reader
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
	declared := &declaredCommands{entries: []plugin.CommandEntry{
		{PluginID: "md.numen.links", Command: manifest.Command{ID: "fetch", Title: "Fetch", When: "target.noteType == 'link'"}},
	}}
	files := noteFiles{
		"links/talk.md":  "---\ntype: link\nurl: https://video.example/talk\n---\n",
		"notes/plain.md": "# A plain note\n",
	}
	commands := newUseCases(declared, declared, getTypes(), files, nil)

	if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.links", Command: "fetch", Path: "links/talk.md"}); err != nil {
		t.Fatalf("a link note was refused: %v", err)
	}
	if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.links", Command: "fetch", Path: "notes/plain.md"}); !errors.Is(err, plugin.ErrBadArguments) {
		t.Fatalf("a plain note was run over: %v", err)
	}
	if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.links", Command: "fetch"}); !errors.Is(err, plugin.ErrBadArguments) {
		t.Fatalf("a command over notes ran over no file: %v", err)
	}
	if len(declared.calls) != 1 || declared.calls[0].Path != "links/talk.md" {
		t.Errorf("ran %+v, want the link note alone", declared.calls)
	}
}

// settingsOf is the config of when conditions it holds.
type settingsOf map[string]any

func (s settingsOf) GetWhenConfig() map[string]any { return s }

func TestRunRejectsDisabledCommandSetting(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "org.example.scans", Command: manifest.Command{
		ID: "scan", Title: "Scan", When: "config.org.example.scans.isEnabled == true",
	}}}}
	config := settingsOf{"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isEnabled": false}}}}
	run := newUseCases(declared, declared, getTypes(), nil, config).Runner

	_, err := run.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "org.example.scans", Command: "scan"})

	if !errors.Is(err, plugin.ErrBadArguments) || len(declared.calls) != 0 {
		t.Errorf("got %v, ran %v", err, declared.calls)
	}
}

func TestRunAnswersTheErrorOfANoteItCannotRead(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{
		{PluginID: "md.numen.links", Command: manifest.Command{ID: "fetch", Title: "Fetch", When: "target.noteType == 'link'"}},
	}}
	commands := newUseCases(declared, declared, getTypes(), noteFiles{}, nil)

	_, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.links", Command: "fetch", Path: "links/gone.md"})

	if !errors.Is(err, fs.ErrNotExist) || errors.Is(err, plugin.ErrBadArguments) || len(declared.calls) != 0 {
		t.Errorf("got %v, ran %v, want the read error", err, declared.calls)
	}
}

func TestARunOverPathsIsCheckedOverEachPathAndForwardsThem(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "md.numen.a", Command: manifest.Command{
		ID: "tags.update", Title: "Update tags", When: "target.isNote",
	}}}}
	commands := newUseCases(declared, declared, getTypes(), nil, nil)

	if _, err := commands.Runner.Run(t.Context(), getVault(), plugin.CommandCall{PluginID: "md.numen.a", Command: "tags.update", Paths: []string{"a.md", "b.md"}}); err != nil {
		t.Fatalf("two notes answered %v", err)
	}
	if got := declared.calls[len(declared.calls)-1].Paths; !slices.Equal(got, []string{"a.md", "b.md"}) {
		t.Errorf("the plugin was given %v", got)
	}
	mixed := plugin.CommandCall{PluginID: "md.numen.a", Command: "tags.update", Paths: []string{"a.md", "talks/a.mp3"}}
	if _, err := commands.Runner.Run(t.Context(), getVault(), mixed); !errors.Is(err, plugin.ErrBadArguments) {
		t.Errorf("a recording among the notes answered %v, want ErrBadArguments", err)
	}
}

func TestRunOverPathsReadsRemovedNoteAsNoteWithoutFrontmatter(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "md.numen.a", Command: manifest.Command{
		ID: "tags.update", Title: "Update tags", When: "target.isNote && target.noteType != 'link'",
	}}}}
	files := noteFiles{"notes/kept.md": "---\ntags: [river]\n---\n"}
	commands := newUseCases(declared, declared, getTypes(), files, nil)

	call := plugin.CommandCall{PluginID: "md.numen.a", Command: "tags.update", Paths: []string{"notes/gone.md", "notes/kept.md"}}
	if _, err := commands.Runner.Run(t.Context(), getVault(), call); err != nil {
		t.Fatalf("a removed note among the paths answered %v", err)
	}
	if got := declared.calls[len(declared.calls)-1].Paths; !slices.Equal(got, []string{"notes/gone.md", "notes/kept.md"}) {
		t.Errorf("the plugin was given %v", got)
	}
}

// handlingPlugins declares a handler of a shared command of a plugin with a backend.
type handlingPlugins struct{ mixedCommands }

func (h *handlingPlugins) ListHandlers() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.cards", Command: manifest.Command{
			ID: "app:file.createText", Title: "Read the deck", When: "target.contentType == 'application/pdf'",
		}},
	}
}

func TestRun_RunsTheHandlerOfASharedCommandInItsPlugin(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := newUseCases(plugins, plugins, getTypes(), nil, nil)

	result, err := commands.Runner.Run(t.Context(), domain.Vault{}, plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "app:file.createText", Path: "decks/tides.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "ran app:file.createText" || len(plugins.calls) != 1 || plugins.calls[0].Path != "decks/tides.pdf" {
		t.Errorf("got %+v, calls %+v", result, plugins.calls)
	}
}

func TestRun_RefusesAHandlerWhoseConditionDoesNotHold(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := newUseCases(plugins, plugins, getTypes(), nil, nil)

	_, err := commands.Runner.Run(t.Context(), domain.Vault{}, plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "app:file.createText", Path: "notes/tides.md",
	})
	if !errors.Is(err, plugin.ErrBadArguments) || len(plugins.calls) != 0 {
		t.Errorf("got %v, calls %+v", err, plugins.calls)
	}
}

// mixedCommands declares commands of plugins with a backend and a frontend, with a frontend only,
// with neither, and of one the host holds suspended.
type mixedCommands struct{ calls []plugin.CommandCall }

func (m *mixedCommands) ListHandlers() []plugin.CommandEntry { return nil }

func (m *mixedCommands) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.cards", Command: manifest.Command{ID: "deck.create", Title: "Create deck"}},
		{PluginID: "md.numen.tree", Command: manifest.Command{ID: "tree.open", Title: "Open file tree", When: "target.isNote"}},
		{PluginID: "md.numen.empty", Command: manifest.Command{ID: "empty.run", Title: "Run"}},
		{PluginID: "md.numen.down", Command: manifest.Command{ID: "down.run", Title: "Run"}},
	}
}

func (m *mixedCommands) GetParts(pluginID string) plugin.Parts {
	return map[string]plugin.Parts{
		"md.numen.cards": {HasBackend: true, HasFrontend: true},
		"md.numen.tree":  {HasFrontend: true},
		"md.numen.down":  {HasBackend: true, IsSuspended: true},
	}[pluginID]
}

func (m *mixedCommands) RunCommand(_ context.Context, call plugin.CommandCall) (plugin.CommandResult, error) {
	m.calls = append(m.calls, call)
	return plugin.CommandResult{Message: "ran " + call.Command}, nil
}

func TestACommandOfAPluginWithoutABackendIsNotRunInABackend(t *testing.T) {
	declared := &mixedCommands{}

	_, err := newUseCases(declared, declared, getTypes(), nil, nil).Runner.Run(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "md.numen.tree", Command: "tree.open",
	})

	if !errors.Is(err, plugin.ErrUnknownCommand) || len(declared.calls) != 0 {
		t.Fatalf("the run answered %v and reached the plugin with %+v", err, declared.calls)
	}
}

func TestACommandOfAPluginWithABackendIsRunInItsBackend(t *testing.T) {
	declared := &mixedCommands{}

	result, err := newUseCases(declared, declared, getTypes(), nil, nil).Runner.Run(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "deck.create",
	})

	if err != nil || result.Message != "ran deck.create" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
}

func TestRunPassesTheVersionArgumentToACommandWithItems(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "org.example.proofs", Command: manifest.Command{
		ID: "proofread", Title: "Proofread", When: "version.isLive",
	}}}}
	run := newUseCases(declared, declared, getTypes(), nil, nil).Runner

	_, err := run.Run(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "org.example.proofs", Command: "proofread", Path: "talks/a.mp3",
		Arguments: map[string]string{manifest.CommandArgumentVersion: "org.example.ears/ears"},
	})

	if err != nil || len(declared.calls) != 1 || declared.calls[0].Arguments[manifest.CommandArgumentVersion] != "org.example.ears/ears" {
		t.Errorf("got %v, ran %+v", err, declared.calls)
	}
}

// useCases is the runner built the way the container builds it.
type useCases struct{ Runner commandrun.Runner }

func newUseCases(commands commandrun.Commands, runs commandrun.Runs, types commandrun.ContentTypes, files noteFiles, config commandrun.Config) useCases {
	var open commandrun.OpenReader
	if files != nil {
		open = files.Open
	}
	return useCases{Runner: commandrun.New(commands, runs, types, open, config, when.NewConditions())}
}
