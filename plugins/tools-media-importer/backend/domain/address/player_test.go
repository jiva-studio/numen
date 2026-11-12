package address_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
)

func TestGetPlayerURLFramesTheVideoAnAddressNames(t *testing.T) {
	const want = address.PlayerOrigin + "/embed/azAZ09-_abc"
	for _, link := range []string{
		"https://www.youtube.com/watch?v=azAZ09-_abc",
		"https://youtu.be/azAZ09-_abc",
		"https://m.youtube.com/shorts/azAZ09-_abc",
		"https://www.youtube-nocookie.com/embed/azAZ09-_abc",
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
		"https://www.youtube.com/watch?v=a`bcdefghij",
		"https://www.youtube.com/watch?v=a{bcdefghij",
		"https://www.youtube.com/watch?v=a@bcdefghij",
		"https://www.youtube.com/watch?v=a[bcdefghij",
		"https://www.youtube.com/watch?v=a.bcdefghij",
		"https://www.youtube.com/watch?v=a:bcdefghij",
		"https://tidal-mills.example/watch?v=tidalmill01",
	} {
		if got, ok := address.GetPlayerURL(address.URL(link)); ok {
			t.Errorf("%s is played at %q", link, got)
		}
	}
}
