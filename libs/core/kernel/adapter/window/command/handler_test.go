package command_test

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// shownVault is a window showing one vault.
type shownVault struct{}

func (shownVault) GetCurrentVault() domain.Vault { return domain.Vault{ID: "v1", Name: "v1"} }

// pluginCommands declares one command and answers every run with the outcome it is given.
type pluginCommands struct {
	result plugin.CommandResult
	err    error
}

func (p pluginCommands) ListKeybindings() []plugin.KeybindingEntry { return nil }

func (p pluginCommands) ListSlots() []plugin.SlotEntry { return nil }

func (p pluginCommands) ListHandlers() []plugin.CommandEntry { return nil }

func (p pluginCommands) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "org.example.sample", Command: manifest.Command{
			ID: "sample.create", Title: "Create a sample", Icon: "layers", When: "target.contentType == 'text/markdown'",
			Arguments: []manifest.CommandArgument{
				{Name: "pages", Title: "Pages", Type: manifest.CommandArgumentNumber, IsRequired: true},
				{Name: "preview", Title: "Preview", Type: manifest.CommandArgumentBoolean},
			},
		}},
		{PluginID: "org.example.sample", Command: manifest.Command{
			ID: "print-sample", Title: "Print a sample", When: "target.contentType == 'text/markdown'",
		}},
		{PluginID: "md.numen.cards", Command: manifest.Command{ID: "shuffle-deck", Title: "Shuffle a deck", When: "target.noteType == 'deck'"}},
	}
}

func (p pluginCommands) GetParts(string) plugin.Parts {
	return plugin.Parts{HasBackend: true}
}

func (p pluginCommands) RunCommand(context.Context, plugin.CommandCall) (plugin.CommandResult, error) {
	return p.result, p.err
}

func run(t *testing.T, commands pluginCommands, request *v1.RunPluginCommandRequest) *v1.RunPluginCommandResponse {
	t.Helper()
	res, err := command.NewHandler(shownVault{}, plugincommand.NewCommands(commands, commands, fileTypes{".md": "text/markdown"}, nil, nil)).RunPluginCommand(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestListCommands_ListsTheCommandsWithTheirConditionAndTheContentTypesByExtension(t *testing.T) {
	res, err := command.NewHandler(shownVault{}, plugincommand.NewCommands(pluginCommands{}, pluginCommands{}, fileTypes{".md": "text/markdown", ".markdown": "text/markdown"}, nil, nil)).ListCommands(
		t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	commands := res.Msg.GetCommands()
	if len(commands) != 3 {
		t.Fatalf("got %v", commands)
	}
	one := commands[0]
	argument := one.GetArguments()[0]
	if one.GetPluginId() != "org.example.sample" || one.GetId() != "sample.create" || one.GetIcon() != "layers" || one.GetWhen() != "target.contentType == 'text/markdown'" ||
		!argument.GetIsRequired() || len(one.GetArguments()) != 2 {
		t.Errorf("got %v", one)
	}
	if types := res.Msg.GetContentTypeByExtension(); !maps.Equal(types, map[string]string{"md": "text/markdown", "markdown": "text/markdown"}) {
		t.Errorf("the content types came out as %v", types)
	}
}

func TestRunPluginCommand_ReturnsMessageAndErrorCode(t *testing.T) {
	valid := &v1.RunPluginCommandRequest{
		PluginId: "org.example.sample", Command: "sample.create", Path: "a.md", Arguments: map[string]string{"pages": "2"},
	}
	cases := map[string]struct {
		commands pluginCommands
		request  *v1.RunPluginCommandRequest
		want     v1.CommandErrorCode
		message  string
	}{
		"a command that ran": {
			pluginCommands{result: plugin.CommandResult{Message: "made a.md", OpenedPath: "a.md"}}, valid, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED, "made a.md",
		},
		"a command no plugin declares": {
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "org.example.sample", Command: "shred"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND, "",
		},
		"arguments that do not meet it": {
			pluginCommands{}, &v1.RunPluginCommandRequest{PluginId: "org.example.sample", Command: "sample.create", Path: "a.md"},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS, "",
		},
		"a command that failed": {
			pluginCommands{err: &plugin.CommandFailedError{Message: "the disk is full"}}, valid,
			v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, "the disk is full",
		},
		"a plugin that cannot start": {
			pluginCommands{err: &plugin.UnavailableError{PluginID: "org.example.sample", Reason: plugin.Failed}}, valid,
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

func TestListCommands_CarriesEveryArgumentWithItsRequirement(t *testing.T) {
	res, err := command.NewHandler(shownVault{}, plugincommand.NewCommands(pluginCommands{}, pluginCommands{}, fileTypes{".md": "text/markdown"}, nil, nil)).ListCommands(
		t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	type carried struct {
		name       string
		isRequired bool
	}
	var got []carried
	for _, one := range res.Msg.GetCommands()[0].GetArguments() {
		got = append(got, carried{one.GetName(), one.GetIsRequired()})
	}
	want := []carried{{"pages", true}, {"preview", false}}
	if !slices.Equal(got, want) {
		t.Errorf("the wire carries %v, want %v", got, want)
	}
}
