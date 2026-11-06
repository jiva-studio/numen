package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestTheWindowToolsListTheirInstructionsOnShowingAPlace(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "window-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)

	listing, err := tools.NewClient(hostcall.Endpoint{BaseURL: web.URL, Token: "window-token"}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != instructions {
		t.Errorf("the window tools list the instructions %q, want %q", listing.Instructions, instructions)
	}
	for _, want := range []string{"source_focus", "window_tabs_get"} {
		if !strings.Contains(listing.Instructions, want) {
			t.Errorf("the instructions do not name %s: %q", want, listing.Instructions)
		}
	}
	if len(listing.Tools) == 0 {
		t.Error("the window tools list no tools")
	}
}
