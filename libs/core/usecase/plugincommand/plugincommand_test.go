package plugincommand_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// declaredCommands lists the commands it is given and keeps every call it is asked to run.
type declaredCommands struct {
	entries []plugincommand.Entry
	calls   []plugincommand.Call
}

func (d *declaredCommands) ListCommands() []plugincommand.Entry { return d.entries }

func (d *declaredCommands) RunCommand(_ context.Context, call plugincommand.Call) (plugincommand.Result, error) {
	d.calls = append(d.calls, call)
	return plugincommand.Result{Message: "ran " + call.Command, OpenedPath: call.Path}, nil
}

func newDeclared() *declaredCommands {
	return &declaredCommands{entries: []plugincommand.Entry{
		{PluginID: "md.numen.a", Command: plugincommand.Command{
			ID: "convert", Title: "Convert", FileExtensions: []string{"pdf"},
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

	result, err := plugincommand.NewCommands(declared).Run.Execute(t.Context(), plugincommand.Call{
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
		"a file of an extension it does not run over": {
			PluginID: "md.numen.a", Command: "convert", Path: "a.epub", Arguments: map[string]string{"pages": "1"},
		},
		"a file given to a command over none": {PluginID: "md.numen.a", Command: "tidy", Path: "a.md"},
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

			_, err := plugincommand.NewCommands(declared).Run.Execute(t.Context(), call)

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
	commands := plugincommand.NewCommands(declared)

	if _, err := commands.Run.Execute(t.Context(), plugincommand.Call{PluginID: "md.numen.b", Command: "tidy"}); err != nil {
		t.Fatal(err)
	}
	_, err := commands.Run.Execute(t.Context(), plugincommand.Call{PluginID: "md.numen.b", Command: "convert"})

	if declared.calls[0].PluginID != "md.numen.b" {
		t.Errorf("the run went to %s", declared.calls[0].PluginID)
	}
	if !errors.Is(err, plugincommand.ErrUnknownCommand) {
		t.Errorf("a command the plugin does not declare is %v, want ErrUnknownCommand", err)
	}
}

func TestACommandIsFoundByThePluginOfTheLowestID(t *testing.T) {
	found, isFound := plugincommand.NewCommands(newDeclared()).List.Find("tidy")

	if !isFound || found.PluginID != "md.numen.a" {
		t.Errorf("found %+v, %v", found, isFound)
	}
}

func TestAnInstallationWithNoPluginHostListsNoCommandAndRunsNone(t *testing.T) {
	commands := plugincommand.NewCommands(nil)

	if entries := commands.List.Execute(); len(entries) != 0 {
		t.Errorf("got %+v", entries)
	}
	if _, err := commands.Run.Execute(t.Context(), plugincommand.Call{Command: "tidy"}); !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("a run answered %v, want unavailable", err)
	}
}
