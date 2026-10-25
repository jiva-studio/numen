package command_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// pluginCommands declares one command and answers every run with the outcome it is given.
type pluginCommands struct {
	result plugincommand.Result
	err    error
}

func (p pluginCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.cards", Command: plugincommand.Command{
			ID: "create-deck", Title: "Create a deck", FileExtensions: []string{"md"},
			Arguments: []plugincommand.Argument{{Name: "pages", Title: "Pages", Type: plugincommand.ArgumentNumber, IsRequired: true}},
		}},
		{PluginID: "md.numen.cards", Command: plugincommand.Command{
			ID: "print-deck", Title: "Print a deck", FileExtensions: []string{"md"}, IsHiddenInPalette: true,
		}},
	}
}

func (p pluginCommands) RunCommand(context.Context, plugincommand.Call) (plugincommand.Result, error) {
	return p.result, p.err
}

func run(t *testing.T, commands pluginCommands, request *v1.RunPluginCommandRequest) *v1.RunPluginCommandResponse {
	t.Helper()
	res, err := command.NewHandler(plugincommand.NewCommands(commands)).RunPluginCommand(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestTheCommandsAreListedWithoutThoseHiddenInThePalette(t *testing.T) {
	res, err := command.NewHandler(plugincommand.NewCommands(pluginCommands{})).ListCommands(
		t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	commands := res.Msg.GetCommands()
	if len(commands) != 1 {
		t.Fatalf("got %v", commands)
	}
	one := commands[0]
	argument := one.GetArguments()[0]
	if one.GetPluginId() != "md.numen.cards" || one.GetId() != "create-deck" || one.GetFileExtensions()[0] != "md" ||
		!argument.GetIsRequired() {
		t.Errorf("got %v", one)
	}
}

func TestARunIsAnsweredWithWhatAPersonReadsAndWhyItDidNotSucceed(t *testing.T) {
	valid := &v1.RunPluginCommandRequest{
		PluginId: "md.numen.cards", Command: "create-deck", Path: "a.md", Arguments: map[string]string{"pages": "2"},
	}
	cases := map[string]struct {
		commands pluginCommands
		request  *v1.RunPluginCommandRequest
		want     v1.CommandErrorCode
		message  string
	}{
		"a command that ran": {
			pluginCommands{result: plugincommand.Result{Message: "made a.md", OpenedPath: "a.md"}}, valid, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED, "made a.md",
		},
		"a command no plugin declares": {
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "md.numen.cards", Command: "shred"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND, "",
		},
		"arguments that do not meet it": {
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "md.numen.cards", Command: "create-deck", Path: "a.md"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS, "",
		},
		"a command that failed": {
			pluginCommands{err: &pluginport.CommandFailedError{Message: "the disk is full"}}, valid,
			v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, "the disk is full",
		},
		"a plugin that cannot start": {
			pluginCommands{err: &pluginport.UnavailableError{PluginID: "md.numen.cards", Reason: pluginport.PluginFailed}}, valid,
			v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE, "plugin md.numen.cards failed to start",
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			res := run(t, one.commands, one.request)

			if res.GetError() != one.want {
				t.Errorf("error = %v, want %v", res.GetError(), one.want)
			}
			if one.message != "" && res.GetMessage() != one.message {
				t.Errorf("message = %q, want %q", res.GetMessage(), one.message)
			}
			if one.want == v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED && res.GetOpenedPath() != "a.md" {
				t.Errorf("opened path = %q", res.GetOpenedPath())
			}
			if res.GetMessage() == "" {
				t.Error("the answer says nothing")
			}
		})
	}
}
