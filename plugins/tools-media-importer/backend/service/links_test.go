package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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
	if raw, _, err := hosted.Vaults("").Files.Read(t.Context(), created.Paths[0]); err != nil || !strings.Contains(string(raw), "page: ") || !strings.Contains(string(raw), "media: ") {
		t.Errorf("the note created holds %s (%v), want its page and media", raw, err)
	}
	if !strings.HasPrefix(created.Paths[0], "talks/") {
		t.Errorf("the note is created at %s, want in talks", created.Paths[0])
	}
}

func TestCreateLinksOpensTheNoteOfAnAddressThatHasOne(t *testing.T) {
	links, _, _ := newMockLinks(t)
	service := linkServer{links: links}
	first, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{videoAddress}})
	if err != nil || len(first.Paths) != 1 {
		t.Fatalf("first drop = %+v, %v", first, err)
	}
	links.Wait()

	again, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{videoAddress}})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Paths) != 1 || again.Paths[0] != first.Paths[0] || len(again.Failed) != 0 {
		t.Errorf("second drop = %+v, want the path %s", again, first.Paths[0])
	}
}

func TestCreateLinksReturnsThePathOfAnAddressDroppedTwiceOnce(t *testing.T) {
	links, _, _ := newMockLinks(t)
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

func TestCreateLinksHandlesEachAddressOnItsOwn(t *testing.T) {
	links, _, hosted := newMockLinks(t)
	service := linkServer{links: links}
	const second = "https://www.youtube.com/watch?v=reedlamp02"
	const blocked = "https://www.youtube.com/watch?v=copperfern03"
	if _, err := service.createLinks(t.Context(), api.CreateLinksRequest{Folder: "talks", URLs: []string{second}}); err != nil {
		t.Fatal(err)
	}
	links.Wait()
	name, _, err := domain.Filename(blocked)
	if err != nil {
		t.Fatal(err)
	}
	if err := hosted.Vaults("").Writer.Create(t.Context(), "talks/"+name+domain.NoteExtension, []byte("plain text")); err != nil {
		t.Fatal(err)
	}

	got, err := service.createLinks(t.Context(), api.CreateLinksRequest{
		Folder: "talks", URLs: []string{videoAddress, second, blocked},
	})
	if err != nil {
		t.Fatal(err)
	}
	links.Wait()
	if len(got.Paths) != 2 || !strings.HasPrefix(got.Paths[0], "talks/") || !strings.HasPrefix(got.Paths[1], "talks/") {
		t.Errorf("paths = %v, want the notes of the first and second address", got.Paths)
	}
	if len(got.Failed) != 1 || got.Failed[0] != blocked {
		t.Errorf("failed = %v, want [%s]", got.Failed, blocked)
	}
	if found, err := service.getLink(t.Context(), api.GetLinkRequest{Path: got.Paths[0]}); err != nil || found.URL != videoAddress {
		t.Errorf("the first note reads %+v (%v)", found, err)
	}
}
