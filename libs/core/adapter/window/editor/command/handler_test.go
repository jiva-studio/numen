package command_test

import (
	"context"
	"path"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

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
		{PluginID: "org.example.sample", Command: plugincommand.Command{
			ID: "create-sample", Title: "Create a sample", ContentTypes: []domain.ContentType{"text/markdown"},
			Arguments: []plugincommand.Argument{{Name: "pages", Title: "Pages", Type: plugincommand.ArgumentNumber, IsRequired: true}},
		}},
		{PluginID: "org.example.sample", Command: plugincommand.Command{
			ID: "print-sample", Title: "Print a sample", ContentTypes: []domain.ContentType{"text/markdown"}, IsHiddenInPalette: true,
		}},
	}
}

func (p pluginCommands) RunCommand(context.Context, plugincommand.Call) (plugincommand.Result, error) {
	return p.result, p.err
}

func run(t *testing.T, commands pluginCommands, request *v1.RunPluginCommandRequest) *v1.RunPluginCommandResponse {
	t.Helper()
	res, err := command.NewHandler(plugincommand.NewCommands(commands, fileTypes{".md": "text/markdown"})).RunPluginCommand(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestTheCommandsAreListedWithoutThoseHiddenInThePalette(t *testing.T) {
	res, err := command.NewHandler(plugincommand.NewCommands(pluginCommands{}, fileTypes{".md": "text/markdown", ".markdown": "text/markdown"})).ListCommands(
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
	if one.GetPluginId() != "org.example.sample" || one.GetId() != "create-sample" || !slices.Equal(one.GetContentTypes(), []string{"text/markdown"}) || !slices.Equal(one.GetExtensions(), []string{"markdown", "md"}) ||
		!argument.GetIsRequired() {
		t.Errorf("got %v", one)
	}
}

func TestARunIsAnsweredWithWhatAPersonReadsAndWhyItDidNotSucceed(t *testing.T) {
	valid := &v1.RunPluginCommandRequest{
		PluginId: "org.example.sample", Command: "create-sample", Path: "a.md", Arguments: map[string]string{"pages": "2"},
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
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "org.example.sample", Command: "shred"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND, "",
		},
		"arguments that do not meet it": {
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "org.example.sample", Command: "create-sample", Path: "a.md"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS, "",
		},
		"a command that failed": {
			pluginCommands{err: &pluginport.CommandFailedError{Message: "the disk is full"}}, valid,
			v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, "the disk is full",
		},
		"a plugin that cannot start": {
			pluginCommands{err: &pluginport.UnavailableError{PluginID: "org.example.sample", Reason: pluginport.PluginFailed}}, valid,
			v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE, "plugin org.example.sample failed to start",
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
