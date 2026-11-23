package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/api"
)

func TestCreateLinksMakesANoteOfEachWebAddressInTheFolder(t *testing.T) {
	links, _, hosted, tasks := newMockLinks(t)
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
	waitForFetch(t, tasks, created.Paths[0])
	found, err := service.getLink(t.Context(), api.GetLinkRequest{Path: created.Paths[0]})
	if err != nil {
		t.Fatal(err)
	}
	if found.URL != videoAddress {
		t.Errorf("the note created reads %+v", found)
	}
	if raw, _, err := hosted.GetVault("").Files.Read(t.Context(), created.Paths[0]); err != nil || !strings.Contains(string(raw), "page: ") || !strings.Contains(string(raw), "media: ") {
		t.Errorf("the note created holds %s (%v), want its page and media", raw, err)
	}
	if !strings.HasPrefix(created.Paths[0], "talks/") {
		t.Errorf("the note is created at %s, want in talks", created.Paths[0])
	}
}

func TestCreateLinksOpensTheNoteOfAnAddressThatHasOne(t *testing.T) {
	links, _, _, tasks := newMockLinks(t)
	service := linkServer{links: links}
	first, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{videoAddress}})
	if err != nil || len(first.Paths) != 1 {
		t.Fatalf("first drop = %+v, %v", first, err)
	}
	waitForFetch(t, tasks, first.Paths[0])

	again, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{videoAddress}})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Paths) != 1 || again.Paths[0] != first.Paths[0] || len(again.Failed) != 0 {
		t.Errorf("second drop = %+v, want the path %s", again, first.Paths[0])
	}
}

func TestCreateLinksReturnsThePathOfAnAddressDroppedTwiceOnce(t *testing.T) {
	links, _, _, _ := newMockLinks(t)
	service := linkServer{links: links}
	got, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{videoAddress, videoAddress},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Paths) != 1 || len(got.Failed) != 0 {
		t.Errorf("drop = %+v, want one path and no failure", got)
	}
}

func TestCreateLinksNamesNotesOfOneTitleWithNumericSuffixes(t *testing.T) {
	links, _, _, tasks := newMockLinks(t)
	service := linkServer{links: links}
	const second = "https://www.youtube.com/watch?v=reedlamp02"
	early, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{second}})
	if err != nil || len(early.Paths) != 1 {
		t.Fatalf("the drop of %s = %+v, %v", second, early, err)
	}
	waitForFetch(t, tasks, early.Paths[0])

	got, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{videoAddress, second},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Paths) != 2 || got.Paths[1] != early.Paths[0] || got.Paths[0] == early.Paths[0] || len(got.Failed) != 0 {
		t.Fatalf("paths = %v, want a new note and the note of the second address", got.Paths)
	}
	waitForFetch(t, tasks, got.Paths[0])
	if found, err := service.getLink(t.Context(), api.GetLinkRequest{Path: got.Paths[0]}); err != nil || found.URL != videoAddress {
		t.Errorf("the first note reads %+v (%v)", found, err)
	}
}
