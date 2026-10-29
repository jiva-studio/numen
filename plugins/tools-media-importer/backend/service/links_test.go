package service

import (
	"strings"
	"testing"

	"connectrpc.com/connect"

	linksv1 "github.com/jiva-studio/numen/plugins/tools-media-importer/gen/numen/links/v1"
)

func TestCreateLinksMakesANoteOfEachWebAddressInTheFolder(t *testing.T) {
	links, _, _ := newMockLinks(t)
	server := linkServer{links: links}

	answer, err := server.CreateLinks(t.Context(), connect.NewRequest(&linksv1.CreateLinksRequest{
		Folder: "talks", Urls: []string{videoAddress, "not an address"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	paths := answer.Msg.GetPaths()
	if answer.Msg.GetError() != linksv1.LinkErrorCode_LINK_ERROR_CODE_UNSPECIFIED || len(paths) != 1 {
		t.Fatalf("answered %v", answer.Msg)
	}
	made, err := server.GetLink(t.Context(), connect.NewRequest(&linksv1.GetLinkRequest{Path: paths[0]}))
	if err != nil {
		t.Fatal(err)
	}
	if made.Msg.GetUrl() != videoAddress || made.Msg.GetEmbedPage() == "" {
		t.Errorf("the note made reads %v", made.Msg)
	}
	links.Wait()
	if !strings.HasPrefix(paths[0], "talks/") {
		t.Errorf("the note is made at %s, want in talks", paths[0])
	}
}
