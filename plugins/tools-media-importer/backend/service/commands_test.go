package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

const pluginID = "md.numen.tools.media.importer"

// videoAddress is an address the mock downloader takes for a video site.
const videoAddress = "https://www.youtube.com/watch?v=tidalmill01"

// newMockLinks is the links of an empty vault, reaching video sites through the mock downloader.
func newMockLinks(t *testing.T) (*link.Links, *hosttest.Store) {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, pluginID)
	texts.SetStore(pluginID, store)
	links := link.New(link.Host{Vault: vault, Store: store, Texts: texts, Queue: hosttest.NewQueue()},
		func() link.Fetcher { return download.NewMock() },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	return links, store
}

// newMockCommands is the plugin's commands over an empty vault, reaching video sites through the
// mock downloader.
func newMockCommands(t *testing.T) (*command.Registry, *hosttest.Store) {
	t.Helper()
	links, store := newMockLinks(t)
	commands, err := newCommands(links)
	if err != nil {
		t.Fatal(err)
	}
	return commands, store
}

func TestImportWithWaitKeepsTheCopyBeforeItAnswers(t *testing.T) {
	commands, store := newMockCommands(t)

	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": videoAddress, "wait": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(made.Message, "a copy of") || made.OpenedPath == "" {
		t.Fatalf("import answered %+v, want a kept copy and the note to open", made)
	}
	if !hasCopy(t, store) {
		t.Error("the copy is not in the plugin's folder once import answered")
	}
}

func TestImportCommands_DeleteAndDownloadCopyAndText(t *testing.T) {
	commands, store := newMockCommands(t)
	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": videoAddress, "wait": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range []struct {
		id      string
		hasCopy bool
	}{{"delete-copy", false}, {"download-copy", true}, {"download-text", true}} {
		answer, err := commands.RunCommand(t.Context(), command.Request{Command: step.id, Path: made.OpenedPath})
		if err != nil || answer.Message == "" {
			t.Fatalf("%s answered %+v (%v)", step.id, answer, err)
		}
		if hasCopy(t, store) != step.hasCopy {
			t.Errorf("after %s the copy is kept: %v, want %v", step.id, !step.hasCopy, step.hasCopy)
		}
	}
}

// hasCopy reports whether the plugin's folder keeps a copy of a video.
func hasCopy(t *testing.T, store *hosttest.Store) bool {
	t.Helper()
	files, err := store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range files {
		if strings.HasSuffix(one.Name, ".wav") {
			return true
		}
	}
	return false
}
