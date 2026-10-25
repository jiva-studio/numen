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

func TestCreateDeckMakesADeckOfNoCardsInTheFolderGiven(t *testing.T) {
	env := vaulttest.New(t, map[string]string{})
	commands, err := service.New(service.Config{Vault: env.Host}).Commands()
	if err != nil {
		t.Fatal(err)
	}

	message, err := commands.RunCommand(t.Context(), command.Request{
		Command:   "create-deck",
		Arguments: command.Arguments{"title": "Spanish verbs", "folder": "decks"},
	})

	if err != nil || message != "Created the deck decks/Spanish verbs.md" {
		t.Fatalf("the command answered %q, %v", message, err)
	}
	if deck := onDisk(t, env.Vault.Path, "decks/Spanish verbs.md"); !strings.Contains(deck, "type: deck") {
		t.Errorf("the deck holds %q", deck)
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
	commands, err := service.New(service.Config{Vault: vaulttest.New(t, map[string]string{}).Host}).Commands()
	if err != nil {
		t.Fatal(err)
	}

	for _, declared := range manifest.Commands {
		_, err := commands.RunCommand(t.Context(), command.Request{Command: declared.ID})
		if errors.Is(err, command.ErrUnknownCommand) {
			t.Errorf("the manifest declares %s, and the backend does not run it", declared.ID)
		}
	}
}
