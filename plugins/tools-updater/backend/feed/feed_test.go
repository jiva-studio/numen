package feed_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/feed"
)

func serve(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

func TestFetchReadsTheManifestTheEndpointPublishes(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			http.Error(w, "wrong accept", http.StatusBadRequest)
			return
		}
		if _, err := w.Write([]byte(`{"version":"2026.10.1","build":950,"channel":"stable"}`)); err != nil {
			t.Error(err)
		}
	})

	manifest, err := feed.New().Fetch(t.Context(), endpoint)

	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "2026.10.1" || manifest.Build != 950 || manifest.Channel != "stable" {
		t.Errorf("manifest = %+v", manifest)
	}
}

func TestFetchFailsOnAnAnswerOtherThan200(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})

	_, err := feed.New().Fetch(t.Context(), endpoint)

	if !errors.Is(err, feed.ErrStatus) {
		t.Errorf("error = %v, want a status error", err)
	}
}

func TestFetchFailsOnABodyThatIsNoManifest(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("<html>")); err != nil {
			t.Error(err)
		}
	})

	if _, err := feed.New().Fetch(t.Context(), endpoint); err == nil {
		t.Error("a page that is no manifest was read")
	}
}

func TestFetchFailsWhereNothingListens(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()

	if _, err := feed.New().Fetch(t.Context(), endpoint); err == nil {
		t.Error("a closed endpoint was read")
	}
}

func TestFetchFailsOnAnEndpointThatIsNoWebAddress(t *testing.T) {
	for _, endpoint := range []string{"", "latest.json", "file:///etc/passwd", "ftp://example.org/latest.json", "http://"} {
		_, err := feed.New().Fetch(t.Context(), endpoint)

		if !errors.Is(err, feed.ErrNotWebAddress) {
			t.Errorf("Fetch(%q) error = %v, want a refusal of the address", endpoint, err)
		}
	}
}
