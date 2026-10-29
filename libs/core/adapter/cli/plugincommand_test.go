package cli_test

import (
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// pluginCommands declares the commands it is given and records every call it is asked to run.
type pluginCommands struct {
	entries []plugincommand.Entry
	calls   []plugincommand.Call
}

func (p *pluginCommands) ListCommands() []plugincommand.Entry { return p.entries }

func (p *pluginCommands) RunCommand(_ context.Context, call plugincommand.Call) (plugincommand.Result, error) {
	p.calls = append(p.calls, call)
	return plugincommand.Result{Message: "ran " + call.Command, OpenedPath: call.Path}, nil
}

// createPluginCommandDeps is a terminal over the notebook whose plugins declare a command over no
// file and one over PDFs, and the vaults the plugins were opened over.
func createPluginCommandDeps() (cli.Deps, *pluginCommands, *[]domain.Vault) {
	declared := &pluginCommands{entries: []plugincommand.Entry{
		{PluginID: "org.example.sample", Command: plugincommand.Command{
			ID: "create-sample", Title: "Create a sample",
			Arguments: []plugincommand.Argument{
				{Name: "title", Title: "Title", Type: "string", IsRequired: true},
				{Name: "folder", Title: "Folder", Type: "string"},
				{Name: "open", Title: "Open", Type: plugincommand.ArgumentBoolean},
			},
		}},
		{PluginID: "md.numen.pages", Command: plugincommand.Command{
			ID: "count-pages", Title: "Count the pages", ContentTypes: []domain.ContentType{"application/pdf"},
		}},
	}}
	var vaults []domain.Vault
	commands := plugincommand.NewCommands(declared, fileTypes{".pdf": "application/pdf", ".wav": "audio/vnd.wave"}, nil)
	deps := cli.Deps{
		Vaults: createVaultList(notebook),
		PluginCommands: func() cli.PluginCommands {
			return cli.PluginCommands{
				List: commands.List,
				Open: func(_ context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
					vaults = append(vaults, v)
					return commands.Run, nil, nil
				},
			}
		},
	}
	return deps, declared, &vaults
}

func TestPluginCommand_DeclaredCommandRunsWithArgumentFlags(t *testing.T) {
	deps, declared, vaults := createPluginCommandDeps()

	out, err := runWithDeps(deps, "create-sample", "notebook", "--title", "Verbs", "--open", "--folder=samples")

	if err != nil || out != "ran create-sample\n" {
		t.Fatalf("the command answered %q, %v", out, err)
	}
	if len(*vaults) != 1 || (*vaults)[0].ID != notebook.ID {
		t.Errorf("the plugins were opened over %+v", *vaults)
	}
	call := declared.calls[0]
	if call.PluginID != "org.example.sample" || call.Path != "" ||
		call.Arguments["title"] != "Verbs" || call.Arguments["folder"] != "samples" || call.Arguments["open"] != "true" {
		t.Errorf("the plugin was sent %+v", call)
	}
}

func TestPluginCommand_FileArgumentFollowsVault(t *testing.T) {
	deps, declared, _ := createPluginCommandDeps()

	out, err := runWithDeps(deps, "count-pages", "notebook", "books/Bees.pdf")

	if err != nil || out != "ran count-pages\nbooks/Bees.pdf\n" {
		t.Fatalf("the command answered %q, %v", out, err)
	}

	if declared.calls[0].Path != "books/Bees.pdf" {
		t.Errorf("the plugin was sent %+v", declared.calls[0])
	}
}

func TestPluginCommand_InvalidArgumentsPrintUsage(t *testing.T) {
	cases := map[string][]string{
		"no vault":                      {"create-sample"},
		"a file to a command over none": {"create-sample", "notebook", "a.md", "--title", "T"},
		"no file to a command over one": {"count-pages", "notebook"},
		"a flag it does not take":       {"create-sample", "notebook", "--title", "T", "--colour", "red"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			deps, declared, vaults := createPluginCommandDeps()

			_, err := runWithDeps(deps, args...)

			if err == nil || !strings.HasPrefix(err.Error(), "usage: numen-cli "+args[0]+" <vault>") {
				t.Errorf("the command answered %v", err)
			}
			if len(declared.calls) != 0 || len(*vaults) != 0 {
				t.Errorf("the plugin was sent %+v over %+v", declared.calls, *vaults)
			}
		})
	}
}

func TestACommandMissingARequiredArgumentRunsNothing(t *testing.T) {
	deps, declared, _ := createPluginCommandDeps()

	_, err := runWithDeps(deps, "create-sample", "notebook", "--folder", "samples")

	if !errors.Is(err, plugincommand.ErrBadArguments) || len(declared.calls) != 0 {
		t.Errorf("the command answered %v and sent %+v", err, declared.calls)
	}
}

func TestTheHelpListsTheCommandsThePluginsDeclare(t *testing.T) {
	deps, _, _ := createPluginCommandDeps()

	out, err := runWithDeps(deps, "help")

	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"numen-cli create-sample <vault> --title <string> [--folder <string>] [--open]",
		"numen-cli count-pages <vault> <file>",
		"Count the pages",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("the help does not say %q:\n%s", line, out)
		}
	}
}

func TestACommandNoPluginDeclaresIsUnknown(t *testing.T) {
	deps, _, _ := createPluginCommandDeps()

	_, err := runWithDeps(deps, "shred", "notebook")

	if err == nil || !strings.Contains(err.Error(), `unknown command "shred"`) {
		t.Errorf("the command answered %v", err)
	}
}

// The commands of a domain come from the plugin that owns it: without plugins the core's own command
// line neither runs nor mentions one.
func TestTheCoreHasNoCommandOfADomain(t *testing.T) {
	domainCommands := []string{"transcribe", "recognise", "import", "proofread"}
	deps := cli.Deps{Vaults: createVaultList(notebook)}

	help, err := runWithDeps(deps, "help")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range domainCommands {
		if strings.Contains(help, name) {
			t.Errorf("the help of the core mentions %q:\n%s", name, help)
		}
		if _, err := runWithDeps(deps, name, "notebook", "talks/lecture.wav"); err == nil ||
			!strings.Contains(err.Error(), `unknown command "`+name+`"`) {
			t.Errorf("the core answered %s with %v", name, err)
		}
	}
}

func TestADomainCommandRunsInThePluginThatDeclaresIt(t *testing.T) {
	declared := &pluginCommands{entries: []plugincommand.Entry{{
		PluginID: "md.numen.engines.transcription",
		Command:  plugincommand.Command{ID: "transcribe", Title: "Transcribe a recording", ContentTypes: []domain.ContentType{"audio/*"}},
	}}}
	commands := plugincommand.NewCommands(declared, fileTypes{".pdf": "application/pdf", ".wav": "audio/vnd.wave"}, nil)
	deps := cli.Deps{
		Vaults: createVaultList(notebook),
		PluginCommands: func() cli.PluginCommands {
			return cli.PluginCommands{
				List: commands.List,
				Open: func(context.Context, domain.Vault) (plugincommand.Run, func() error, error) {
					return commands.Run, nil, nil
				},
			}
		},
	}

	help, err := runWithDeps(deps, "help")
	if err != nil || !strings.Contains(help, "numen-cli transcribe <vault> <file>\n      Transcribe a recording") {
		t.Fatalf("the help reads %q (%v)", help, err)
	}
	if _, err := runWithDeps(deps, "transcribe", "notebook", "talks/lecture.wav"); err != nil {
		t.Fatal(err)
	}
	if len(declared.calls) != 1 || declared.calls[0].PluginID != "md.numen.engines.transcription" ||
		declared.calls[0].Path != "talks/lecture.wav" {
		t.Errorf("the plugin was sent %+v", declared.calls)
	}
}

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
	return out
}

func TestACommandOverNotesOfATypeIsTypedWithTheNote(t *testing.T) {
	declared := &pluginCommands{entries: []plugincommand.Entry{{
		PluginID: "md.numen.links",
		Command:  plugincommand.Command{ID: "delete-copy", Title: "Delete the copy", NoteTypes: []string{"link"}},
	}}}
	commands := plugincommand.NewCommands(declared, fileTypes{}, nil)
	deps := cli.Deps{
		Vaults: createVaultList(notebook),
		PluginCommands: func() cli.PluginCommands {
			return cli.PluginCommands{List: commands.List}
		},
	}

	help, err := runWithDeps(deps, "help")
	if err != nil || !strings.Contains(help, "numen-cli delete-copy <vault> <file>\n") {
		t.Fatalf("the help reads %q (%v)", help, err)
	}
}
