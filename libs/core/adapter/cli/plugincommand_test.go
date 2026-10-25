package cli_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// pluginCommands declares the commands it is given and keeps every call it is asked to run.
type pluginCommands struct {
	listed []plugincommand.Entry
	calls  []plugincommand.Call
}

func (p *pluginCommands) ListCommands() []plugincommand.Entry { return p.listed }

func (p *pluginCommands) RunCommand(_ context.Context, call plugincommand.Call) (plugincommand.Result, error) {
	p.calls = append(p.calls, call)
	return plugincommand.Result{Message: "ran " + call.Command, OpenedPath: call.Path}, nil
}

// createPluginCommandDeps is a terminal over the notebook whose plugins declare a command over no
// file and one over PDFs, and the vaults the plugins were opened over.
func createPluginCommandDeps() (cli.Deps, *pluginCommands, *[]domain.Vault) {
	declared := &pluginCommands{listed: []plugincommand.Entry{
		{PluginID: "md.numen.cards", Command: plugincommand.Command{
			ID: "create-deck", Title: "Create a deck",
			Arguments: []plugincommand.Argument{
				{Name: "title", Title: "Title", Type: "string", IsRequired: true},
				{Name: "folder", Title: "Folder", Type: "string"},
				{Name: "open", Title: "Open", Type: plugincommand.ArgumentBoolean},
			},
		}},
		{PluginID: "md.numen.pages", Command: plugincommand.Command{
			ID: "count-pages", Title: "Count the pages", FileExtensions: []string{"pdf"},
		}},
	}}
	var opened []domain.Vault
	commands := plugincommand.NewCommands(declared)
	deps := cli.Deps{
		Vaults: createVaultList(notebook),
		PluginCommands: func() cli.PluginCommands {
			return cli.PluginCommands{
				List: commands.List,
				Open: func(_ context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
					opened = append(opened, v)
					return commands.Run, nil, nil
				},
			}
		},
	}
	return deps, declared, &opened
}

func TestACommandAPluginDeclaresIsASubcommandWithItsArgumentsAsFlags(t *testing.T) {
	deps, declared, opened := createPluginCommandDeps()

	out, err := runWithDeps(deps, "create-deck", "notebook", "--title", "Verbs", "--open", "--folder=decks")

	if err != nil || out != "ran create-deck\n" {
		t.Fatalf("the command answered %q, %v", out, err)
	}
	if len(*opened) != 1 || (*opened)[0].ID != notebook.ID {
		t.Errorf("the plugins were opened over %+v", *opened)
	}
	call := declared.calls[0]
	if call.PluginID != "md.numen.cards" || call.Path != "" ||
		call.Arguments["title"] != "Verbs" || call.Arguments["folder"] != "decks" || call.Arguments["open"] != "true" {
		t.Errorf("the plugin was sent %+v", call)
	}
}

func TestACommandOverAFileTakesTheFileAfterTheVaultAndSaysTheFileItOpens(t *testing.T) {
	deps, declared, _ := createPluginCommandDeps()

	out, err := runWithDeps(deps, "count-pages", "notebook", "books/Bees.pdf")

	if err != nil || out != "ran count-pages\nbooks/Bees.pdf\n" {
		t.Fatalf("the command answered %q, %v", out, err)
	}

	if declared.calls[0].Path != "books/Bees.pdf" {
		t.Errorf("the plugin was sent %+v", declared.calls[0])
	}
}

func TestACommandTypedWrongSaysHowItIsTypedAndRunsNothing(t *testing.T) {
	cases := map[string][]string{
		"no vault":                      {"create-deck"},
		"a file to a command over none": {"create-deck", "notebook", "a.md", "--title", "T"},
		"no file to a command over one": {"count-pages", "notebook"},
		"a flag it does not take":       {"create-deck", "notebook", "--title", "T", "--colour", "red"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			deps, declared, opened := createPluginCommandDeps()

			_, err := runWithDeps(deps, args...)

			if err == nil || !strings.HasPrefix(err.Error(), "usage: numen-cli "+args[0]+" <vault>") {
				t.Errorf("the command answered %v", err)
			}
			if len(declared.calls) != 0 || len(*opened) != 0 {
				t.Errorf("the plugin was sent %+v over %+v", declared.calls, *opened)
			}
		})
	}
}

func TestACommandMissingARequiredArgumentRunsNothing(t *testing.T) {
	deps, declared, _ := createPluginCommandDeps()

	_, err := runWithDeps(deps, "create-deck", "notebook", "--folder", "decks")

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
		"numen-cli create-deck <vault> --title <string> [--folder <string>] [--open]",
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
