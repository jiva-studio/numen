package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

func TestTheEmbedPageFramesThePlayerOfTheAddress(t *testing.T) {
	page := getEmbedPage(address.URL(videoAddress))
	if page == "" {
		t.Fatal("a video address has no embed page")
	}
	recorded := httptest.NewRecorder()
	serveEmbed(recorded, httptest.NewRequest(http.MethodGet, "/plugin/"+page, nil))

	body := recorded.Body.String()
	if recorded.Code != http.StatusOK || !strings.Contains(body, `src="`+address.PlayerOrigin+`/embed/tidalmill01"`) ||
		!strings.Contains(body, `referrerpolicy="strict-origin-when-cross-origin"`) {
		t.Errorf("answered %d with %q", recorded.Code, body)
	}
}

func TestAnAddressWithNoPlayerHasNoEmbedPage(t *testing.T) {
	if page := getEmbedPage("https://tidal-mills.example/history"); page != "" {
		t.Errorf("an article has the embed page %q", page)
	}
	recorded := httptest.NewRecorder()
	serveEmbed(recorded, httptest.NewRequest(http.MethodGet, "/plugin/embed?address=https%3A%2F%2Ftidal-mills.example%2F", nil))
	if recorded.Code != http.StatusNotFound {
		t.Errorf("an article's embed page answered %d", recorded.Code)
	}
}
