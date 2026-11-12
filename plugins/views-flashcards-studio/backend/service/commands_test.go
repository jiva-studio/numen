package service_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	flashcardsstudio "github.com/jiva-studio/numen/plugins/views-flashcards-studio"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

func newCommands(t *testing.T) (*command.Registry, string) {
	t.Helper()
	env := vaulttest.New(t, map[string]string{})
	commands, err := service.New(service.Config{Vaults: vaulttest.Vaults(env.Host), Now: vaulttest.NewClock()}).Commands()
	if err != nil {
		t.Fatal(err)
	}
	return commands, env.Vault.Path
}

func TestTheCreateCommandsMakeTheFileAndOpenIt(t *testing.T) {
	for id, kind := range map[string]string{
		"deck.create": "type: deck", "cardModel.create": "type: card-model", "preset.create": "type: preset",
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

// A create command run over a row of the file tree makes the file in the folder of the row: the
// row itself where it is a folder of the vault, and the folder holding it where it is a file.
func TestTheCreateCommandsMakeTheFileInTheFolderOfTheRowTheyRunOver(t *testing.T) {
	for _, id := range []string{"deck.create", "cardModel.create", "preset.create"} {
		for _, row := range []string{"Glaciers", "Glaciers/Lakes.md"} {
			t.Run(id+" over "+row, func(t *testing.T) {
				env := vaulttest.New(t, map[string]string{"Glaciers/Lakes.md": "# Lakes\n"})
				commands, err := service.New(service.Config{Vaults: vaulttest.Vaults(env.Host), Now: vaulttest.NewClock()}).Commands()
				if err != nil {
					t.Fatal(err)
				}

				result, err := commands.RunCommand(t.Context(), command.Request{
					Command: id, Path: row, Arguments: command.Arguments{"title": "Rivers"},
				})

				if err != nil || result.OpenedPath != "Glaciers/Rivers.md" {
					t.Fatalf("the command answered %+v, %v", result, err)
				}
				onDisk(t, env.Vault.Path, "Glaciers/Rivers.md")
			})
		}
	}
}

// A create command run over a path the vault holds no file or folder at makes nothing.
func TestTheCreateCommandsMakeNothingOverAMissingRow(t *testing.T) {
	for _, row := range []string{"Gone", "Glaciers/Gone.md"} {
		t.Run(row, func(t *testing.T) {
			env := vaulttest.New(t, map[string]string{"Glaciers/Lakes.md": "# Lakes\n"})
			commands, err := service.New(service.Config{Vaults: vaulttest.Vaults(env.Host), Now: vaulttest.NewClock()}).Commands()
			if err != nil {
				t.Fatal(err)
			}

			result, err := commands.RunCommand(t.Context(), command.Request{
				Command: "deck.create", Path: row, Arguments: command.Arguments{"title": "Rivers"},
			})

			if err == nil || !strings.Contains(err.Error(), "no such file or folder") || result.OpenedPath != "" {
				t.Fatalf("the command answered %+v, %v", result, err)
			}
			for _, made := range []string{"Rivers.md", "Gone/Rivers.md", "Glaciers/Rivers.md"} {
				if _, statErr := os.Stat(filepath.Join(env.Vault.Path, made)); statErr == nil {
					t.Errorf("the command made %s", made)
				}
			}
		})
	}
}

func TestACardModelMadeByItsCommandDeclaresAField(t *testing.T) {
	commands, root := newCommands(t)

	if _, err := commands.RunCommand(t.Context(), command.Request{
		Command: "cardModel.create", Arguments: command.Arguments{"title": "Mountain"},
	}); err != nil {
		t.Fatal(err)
	}

	if made := onDisk(t, root, "Mountain.md"); !strings.Contains(made, "Field 1") {
		t.Errorf("the card model holds %q", made)
	}
}

func TestCreateDeck_ErrorsWhenFileExists(t *testing.T) {
	commands, _ := newCommands(t)
	request := command.Request{Command: "deck.create", Arguments: command.Arguments{"title": "Lakes"}}
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
		Contributes struct {
			Commands []struct {
				ID string `json:"id"`
			} `json:"commands"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(flashcardsstudio.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	commands, _ := newCommands(t)
	runByTheFrontend := map[string]bool{"preset.open": true}

	for _, declared := range manifest.Contributes.Commands {
		if runByTheFrontend[declared.ID] {
			continue
		}
		_, err := commands.RunCommand(t.Context(), command.Request{
			Command: declared.ID, Arguments: command.Arguments{"title": "Probe " + declared.ID},
		})
		if errors.Is(err, command.ErrUnknownCommand) {
			t.Errorf("the manifest declares %s, and the backend does not run it", declared.ID)
		}
	}
}
