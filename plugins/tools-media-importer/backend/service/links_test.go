package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

func TestCreateLinksMakesANoteOfEachWebAddressInTheFolder(t *testing.T) {
	links, _, _ := newMockLinks(t)
	service := linkServer{links: links}

	created, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{"not an address", videoAddress},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Paths) != 1 {
		t.Fatalf("created %v", created.Paths)
	}
	links.Wait()
	found, err := service.getLink(t.Context(), api.GetLinkRequest{Path: created.Paths[0]})
	if err != nil {
		t.Fatal(err)
	}
	if found.URL != videoAddress || found.EmbedPage == "" || found.CopySize == 0 {
		t.Errorf("the note created reads %+v", found)
	}
	if !strings.HasPrefix(created.Paths[0], "talks/") {
		t.Errorf("the note is created at %s, want in talks", created.Paths[0])
	}
	// Duplicate URL in createLinks returns an API error.
	if _, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{videoAddress},
	}); err == nil {
		t.Error("duplicate createLinks succeeded")
	}
}

func TestGetLinkPropagatesStoreErrors(t *testing.T) {
	vault := hosttest.New("Mills")
	es := &errListStore{Store: vault.Store()}
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, vault.Store())
	hosted := link.Host{Vault: vault, Store: es, Texts: texts, Queue: hosttest.NewQueue(), Clock: clock.NewFake(time.Unix(1_700_000_000, 0))}
	links := link.New(hosted,
		func() link.Fetcher { return download.NewMock() },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	svc := linkServer{links: links}

	made, err := links.Create(t.Context(), "talks", videoAddress)
	if err != nil {
		t.Fatal(err)
	}
	boom := errors.New("store failure")
	es.listErr = boom
	if _, err := svc.getLink(t.Context(), api.GetLinkRequest{Path: made.Path}); !errors.Is(err, boom) {
		t.Errorf("getLink(FindCopy err) = %v, want %v", err, boom)
	}
	es.listErr = nil
	es.readErr = boom
	if _, err := svc.getLink(t.Context(), api.GetLinkRequest{Path: made.Path}); !errors.Is(err, boom) {
		t.Errorf("getLink(IsTextDeleted err) = %v, want %v", err, boom)
	}
}
