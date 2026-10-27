package service

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
	"github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1/epubv1connect"
)

// The handler a host that links the plugin in serves answers the book service with the token it
// was given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheBookService(t *testing.T) {
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := epubv1connect.NewBookServiceClient(web.Client(), web.URL)
	if _, err := client.GetBook(t.Context(), connect.NewRequest(&epubv1.GetBookRequest{Path: "book.epub"})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("GetBook without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&epubv1.GetBookRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	if _, err := client.GetBook(t.Context(), req); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetBook with no path = %v, want invalid argument", err)
	}
}

// A linked handler reaches the host at the port of its variables.
func TestTheLinkedHandlerReachesTheHostOfItsVariables(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": port})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	req := connect.NewRequest(&epubv1.GetBookRequest{Path: filepath.Join("books", "book.epub")})
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = epubv1connect.NewBookServiceClient(web.Client(), web.URL).GetBook(t.Context(), req)

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetBook = %v, want not found", err)
	}
}
