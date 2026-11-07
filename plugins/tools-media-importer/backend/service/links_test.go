package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
)

func TestCreateLinksMakesANoteOfEachWebAddressInTheFolder(t *testing.T) {
	links, _, hosted := newMockLinks(t)
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
	if found.URL != videoAddress {
		t.Errorf("the note created reads %+v", found)
	}
	if raw, _, err := hosted.Vault.Read(t.Context(), created.Paths[0]); err != nil || !strings.Contains(string(raw), "page: ") || !strings.Contains(string(raw), "media: ") {
		t.Errorf("the note created holds %s (%v), want its page and media", raw, err)
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
