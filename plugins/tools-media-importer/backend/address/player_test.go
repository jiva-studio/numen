package address_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

func TestGetPlayerURLFramesTheVideoAnAddressNames(t *testing.T) {
	const want = address.PlayerOrigin + "/embed/tidalmill01"
	for _, link := range []string{
		"https://www.youtube.com/watch?v=tidalmill01",
		"https://youtu.be/tidalmill01",
		"https://m.youtube.com/shorts/tidalmill01",
		"https://www.youtube-nocookie.com/embed/tidalmill01",
	} {
		if got, ok := address.GetPlayerURL(address.URL(link)); !ok || got != want {
			t.Errorf("%s is played at %q (%v), want %q", link, got, ok, want)
		}
	}
}

func TestGetPlayerURLKnowsNoPlayerForAPageOrAnotherSite(t *testing.T) {
	for _, link := range []string{
		"https://www.youtube.com/feed/history",
		"https://www.youtube.com/watch?v=short",
		"https://tidal-mills.example/watch?v=tidalmill01",
	} {
		if got, ok := address.GetPlayerURL(address.URL(link)); ok {
			t.Errorf("%s is played at %q", link, got)
		}
	}
}
