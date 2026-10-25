package service

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the book service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheBookService(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
		"NUMEN_VAULT_PATH":   dir,
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := numenv1connect.NewBookServiceClient(web.Client(), web.URL)
	req := connect.NewRequest(&numenv1.GetBookRequest{Path: name})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.GetBook(t.Context(), req)
	if err != nil {
		t.Fatalf("GetBook: %v", err)
	}
	if len(resp.Msg.GetDocuments()) == 0 {
		t.Error("the book has no documents")
	}
}

// A linked handler holds the paths of requests to the vault of its variables.
func TestTheLinkedHandlerRefusesABookOutsideTheVaultOfItsVariables(t *testing.T) {
	t.Setenv("NUMEN_VAULT_PATH", "")
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": "1", "NUMEN_VAULT_PATH": t.TempDir()})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	req := connect.NewRequest(&numenv1.GetBookRequest{Path: filepath.Join(t.TempDir(), "book.epub")})
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = numenv1connect.NewBookServiceClient(web.Client(), web.URL).GetBook(t.Context(), req)

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("GetBook outside the vault = %v, want permission denied", err)
	}
}
