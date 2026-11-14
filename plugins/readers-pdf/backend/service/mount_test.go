package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

func postDocument(t *testing.T, baseURL, token, body string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/"+api.Service+"/GetDocument", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	return res.StatusCode
}

// The handler a host that links the plugin in answers the document API with the token it was given
// and rejects a call without it.
func TestTheLinkedHandlerAnswersTheDocumentAPI(t *testing.T) {
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if status := postDocument(t, web.URL, "", `{"path":"tiny.pdf"}`); status != http.StatusUnauthorized {
		t.Fatalf("GetDocument without the token = %d, want 401", status)
	}
	if status := postDocument(t, web.URL, "linked-token", `{}`); status != http.StatusBadRequest {
		t.Fatalf("GetDocument with no path = %d, want 400", status)
	}
	pageReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, web.URL+"/plugin/pages/0", nil)
	if err != nil {
		t.Fatal(err)
	}
	pageReq.Header.Set("Authorization", "Bearer linked-token")
	pageRes, err := http.DefaultClient.Do(pageReq)
	if err != nil {
		t.Fatal(err)
	}
	if closeErr := pageRes.Body.Close(); closeErr != nil {
		t.Error(closeErr)
	}
	if pageRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /plugin/pages/0 = %d, want 400", pageRes.StatusCode)
	}
	if _, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"}); !errors.Is(err, hostcall.ErrNoHost) {
		t.Fatalf("NewHandler without host port = %v, want ErrNoHost", err)
	}
}

func TestMountClosesTheWriterAndPropagatesRegistrationErrors(t *testing.T) {
	texts, writer, _ := testsupport.NewVault(t, "books/tiny.pdf", "tiny.pdf")
	reader := newReader(t, newTestdataVault(t), texts)
	srv := server.New()
	if err := mount(srv, reader, writer, func(string, error) {}); err != nil {
		t.Fatalf("mount: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	testsupport.CreateText(t, writer, extract.Request{Vault: "vault", Path: "books/tiny.pdf"})
	listing, err := texts.ListVersions(t.Context(), "books/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Fatalf("versions after server Close = %v, want none", listing.Versions)
	}

	dup := server.New()
	server.Handle(dup.NewAPI(api.Service), "GetDocument", func(context.Context, api.GetDocumentRequest) (api.GetDocumentResponse, error) {
		return api.GetDocumentResponse{}, nil
	})
	if err := mount(dup, reader, writer, func(string, error) {}); err == nil {
		t.Fatal("mount succeeded on duplicate GetDocument handler")
	}
}

// A linked handler connects to the host at the port set in its variables.
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

	status := postDocument(t, web.URL, "linked-token", `{"path":"books/tiny.pdf"}`)

	if status != http.StatusNotFound {
		t.Fatalf("GetDocument = %d, want 404", status)
	}
}
