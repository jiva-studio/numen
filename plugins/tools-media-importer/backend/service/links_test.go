package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
)

func TestCreateLinksMakesANoteOfEachWebAddressInTheFolder(t *testing.T) {
	links, _ := newMockLinks(t)
	service := linkServer{links: links}

	created, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{videoAddress, "not an address"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Paths) != 1 {
		t.Fatalf("created %v", created.Paths)
	}
	found, err := service.getLink(t.Context(), api.GetLinkRequest{Path: created.Paths[0]})
	if err != nil {
		t.Fatal(err)
	}
	if found.URL != videoAddress || found.EmbedPage == "" {
		t.Errorf("the note created reads %+v", found)
	}
	links.Wait()
	if !strings.HasPrefix(created.Paths[0], "talks/") {
		t.Errorf("the note is created at %s, want in talks", created.Paths[0])
	}
}
