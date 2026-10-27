package service_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	flashcardsstudio "github.com/jiva-studio/numen/plugins/views-flashcards-studio"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

func newCommands(t *testing.T) (*command.Registry, string) {
	t.Helper()
	env := vaulttest.New(t, map[string]string{})
	commands, err := service.New(service.Config{Vault: env.Host}).Commands()
	if err != nil {
		t.Fatal(err)
	}
	return commands, env.Vault.Path
}

func TestTheCreateCommandsMakeTheFileAndOpenIt(t *testing.T) {
	for id, kind := range map[string]string{
		"create-deck": "type: deck", "create-card-model": "type: card-model", "create-preset": "type: preset",
	} {
		t.Run(id, func(t *testing.T) {
			commands, root := newCommands(t)

			result, err := commands.RunCommand(t.Context(), command.Request{
				Command: id, Arguments: command.Arguments{"title": "Rivers"},
			})

			if err != nil || result.OpenedPath != "Rivers.md" {
				t.Fatalf("the command answered %+v, %v", result, err)
			}
			if made := onDisk(t, root, "Rivers.md"); !strings.Contains(made, kind) {
				t.Errorf("the file holds %q", made)
			}
		})
	}
}

func TestACardModelMadeByItsCommandDeclaresAField(t *testing.T) {
	commands, root := newCommands(t)

	if _, err := commands.RunCommand(t.Context(), command.Request{
		Command: "create-card-model", Arguments: command.Arguments{"title": "Mountain"},
	}); err != nil {
		t.Fatal(err)
	}

	if made := onDisk(t, root, "Mountain.md"); !strings.Contains(made, "Field 1") {
		t.Errorf("the card model holds %q", made)
	}
}

func TestAFileOfThatNameAlreadyThereIsSaidAndNothingOpens(t *testing.T) {
	commands, _ := newCommands(t)
	request := command.Request{Command: "create-deck", Arguments: command.Arguments{"title": "Lakes"}}
	if _, err := commands.RunCommand(t.Context(), request); err != nil {
		t.Fatal(err)
	}

	result, err := commands.RunCommand(t.Context(), request)

	if err == nil || !strings.Contains(err.Error(), "filed there already") || result.OpenedPath != "" {
		t.Errorf("the second deck answered %+v, %v", result, err)
	}
}

func TestEveryCommandTheManifestDeclaresIsRun(t *testing.T) {
	var manifest struct {
		Commands []struct {
			ID string `json:"id"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(flashcardsstudio.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	commands, _ := newCommands(t)

	for _, declared := range manifest.Commands {
		_, err := commands.RunCommand(t.Context(), command.Request{
			Command: declared.ID, Arguments: command.Arguments{"title": "Probe " + declared.ID},
		})
		if errors.Is(err, command.ErrUnknownCommand) {
			t.Errorf("the manifest declares %s, and the backend does not run it", declared.ID)
		}
	}
}
