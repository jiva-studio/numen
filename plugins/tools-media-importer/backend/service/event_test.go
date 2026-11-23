package service

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/queuetest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func TestAFilesChangedEventDownloadsTheMissingCaptionsOfALinkNote(t *testing.T) {
	vault := hosttest.New("Mills")
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, vault.Store())
	tasks := queuetest.NewLog()
	links := link.New(link.Host{
		PluginID: PluginID, Vaults: vaultsOf(vault), Store: foldersOf(vault), Texts: textsOf(texts),
		Queue: tasks, Clock: newFakeClock(time.Unix(1_700_000_000, 0)), Entropy: random.Entropy{},
	},
		func() ports.Fetcher { return download.NewMock(webpage.New()) },
		func() fetch.Settings { return fetch.Settings{IsTextDownloadAutomatic: true} })
	t.Cleanup(links.Close)
	note := "---\ntype: link\nurl: " + videoAddress + "\n---\n"
	vault.WriteFile("links/mill.md", note)
	texts.WriteFile("links/mill.md", []byte(note))
	app := &App{links: links}

	if err := app.HandleEvent(t.Context(), hosttest.GetIndexChangedEvent(string(vault.Vault().ID))); err != nil {
		t.Fatal(err)
	}
	if err := app.HandleEvent(t.Context(), "onFilesChanged:"+string(vault.Vault().ID)); err != nil {
		t.Fatal(err)
	}
	tasks.WaitForEnd(t, "text:"+string(vault.Vault().ID)+":links/mill.md")
	links.Close()

	listing, err := texts.ListVersions(t.Context(), "links/mill.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 {
		t.Errorf("the link note has %d versions, want 1", len(listing.Versions))
	}
}
