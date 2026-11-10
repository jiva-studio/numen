package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func runImport(t *testing.T, commands *command.Registry, address string) command.Result {
	t.Helper()
	answer, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: address},
	})
	if err != nil {
		t.Fatalf("Import link of %s failed: %v", address, err)
	}
	return answer
}

func TestImportOfAnAddressThatHasALinkNoteOpensThatNote(t *testing.T) {
	commands, _ := newMockCommands(t)
	first := runImport(t, commands, videoAddress)

	again := runImport(t, commands, videoAddress)

	if again.OpenedPath != first.OpenedPath || again.Message != "Opened "+first.OpenedPath {
		t.Errorf("Import link again answered %+v, want the note %s opened", again, first.OpenedPath)
	}
}

func TestImportOfAnAddressWhoseLinkNoteMovedOpensTheMovedNote(t *testing.T) {
	links, _, hosted := newMockLinks(t)
	commands, err := newCommands(links)
	if err != nil {
		t.Fatal(err)
	}
	first := runImport(t, commands, videoAddress)
	vault, isFake := hosted.Vaults("").Files.(*hosttest.Vault)
	if !isFake {
		t.Fatalf("the vault is %T", hosted.Vaults("").Files)
	}
	content, _ := vault.Content(first.OpenedPath)
	vault.Remove(first.OpenedPath)
	vault.File("reading/tidal mill.md", content)

	again := runImport(t, commands, videoAddress)

	if again.OpenedPath != "reading/tidal mill.md" {
		t.Errorf("Import link of a moved note answered %+v, want reading/tidal mill.md opened", again)
	}
	if _, isMade := vault.Content(first.OpenedPath); isMade {
		t.Errorf("Import link made a second note at %s", first.OpenedPath)
	}
}
