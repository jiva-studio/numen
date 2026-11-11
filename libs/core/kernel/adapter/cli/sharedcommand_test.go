package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func createSharedCommandDeps() (cli.Deps, *pluginCommands) {
	declared := &pluginCommands{handlers: []plugin.CommandEntry{
		{PluginID: "org.example.layers", Command: manifest.Command{
			ID: "app:file.createText", Title: "Extract text layer", When: "target.contentType == 'application/pdf'",
		}},
		{PluginID: "org.example.scans", Command: manifest.Command{
			ID: "app:file.createText", Title: "Recognize text", When: "target.contentType == 'application/pdf'",
		}},
		{PluginID: "org.example.ears", Command: manifest.Command{
			ID: "app:file.createText", Title: "Transcribe", When: "target.contentType.startsWith('audio/')",
		}},
		{PluginID: "org.example.links", Command: manifest.Command{
			ID: "app:link.import", Title: "Import link",
			Arguments: []manifest.CommandArgument{{Name: "address", Title: "Web address", Type: "string", IsRequired: true}},
		}},
	}}
	commands := plugincommand.NewCommands(declared, declared, fileTypes{".pdf": "application/pdf", ".wav": "audio/vnd.wave"}, nil, nil)
	find := registry.NewFind(singleVaultList{getNotebook()}).Execute
	deps := cli.Deps{
		Vaults: createVaultList(getNotebook()),
		PluginCommands: func() cli.PluginCommands {
			return cli.PluginCommands{
				List: commands.List, ByName: plugincommand.NewRunByName(find),
				Open: func(context.Context, domain.Vault) (plugincommand.Run, func() error, error) {
					return commands.Run, nil, nil
				},
			}
		},
	}
	return deps, declared
}

func TestASharedCommandRunsItsOneHandlerOverTheFile(t *testing.T) {
	deps, declared := createSharedCommandDeps()

	out, err := runWithDeps(deps, "app:file.createText", "notebook", "talks/lecture.wav")

	if err != nil || out != "ran app:file.createText\ntalks/lecture.wav\n" {
		t.Fatalf("the command answered %q, %v", out, err)
	}
	if len(declared.calls) != 1 || declared.calls[0].PluginID != "org.example.ears" ||
		declared.calls[0].Path != "talks/lecture.wav" {
		t.Errorf("the plugins were sent %+v", declared.calls)
	}
}

func TestASharedCommandWithSeveralHandlersRunsThePluginPicked(t *testing.T) {
	deps, declared := createSharedCommandDeps()

	if _, err := runWithDeps(deps, "app:file.createText", "notebook", "scans/mill.pdf", "--plugin", "org.example.scans"); err != nil {
		t.Fatal(err)
	}
	if len(declared.calls) != 1 || declared.calls[0].PluginID != "org.example.scans" {
		t.Errorf("the plugins were sent %+v", declared.calls)
	}
}

func TestASharedCommandWithSeveralHandlersAndNoPluginPickedRunsNothing(t *testing.T) {
	deps, declared := createSharedCommandDeps()

	_, err := runWithDeps(deps, "app:file.createText", "notebook", "scans/mill.pdf")

	if err == nil || !strings.Contains(err.Error(), "--plugin") ||
		!strings.Contains(err.Error(), "org.example.layers") || !strings.Contains(err.Error(), "org.example.scans") {
		t.Errorf("the command answered %v", err)
	}
	if len(declared.calls) != 0 {
		t.Errorf("the plugins were sent %+v", declared.calls)
	}
}

func TestSharedCommand_FileWithoutHandlerRunsNothing(t *testing.T) {
	deps, declared := createSharedCommandDeps()

	_, err := runWithDeps(deps, "app:file.createText", "notebook", "notes/tide.md")

	if err == nil || len(declared.calls) != 0 {
		t.Errorf("the command answered %v and sent %+v", err, declared.calls)
	}
}

func TestSharedCommand_DeclaredArgumentsAreFlags(t *testing.T) {
	deps, declared := createSharedCommandDeps()

	if _, err := runWithDeps(deps, "app:link.import", "notebook", "--address", "https://example.com/tides"); err != nil {
		t.Fatal(err)
	}
	if len(declared.calls) != 1 || declared.calls[0].PluginID != "org.example.links" ||
		declared.calls[0].Arguments["address"] != "https://example.com/tides" {
		t.Errorf("the plugins were sent %+v", declared.calls)
	}
	if _, err := runWithDeps(deps, "app:link.import", "notebook"); err == nil {
		t.Error("app:link.import without an address ran")
	}
}

func TestTheHelpListsTheSharedCommandsAndTheirHandlers(t *testing.T) {
	deps, _ := createSharedCommandDeps()

	out, err := runWithDeps(deps, "help")

	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"numen-cli app:file.createText <vault> <file> [--plugin <id>]",
		"Extract text layer (org.example.layers)",
		"Transcribe (org.example.ears)",
		"numen-cli app:link.import <vault> --address <string> [--plugin <id>]",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("the help does not say %q:\n%s", line, out)
		}
	}
}
