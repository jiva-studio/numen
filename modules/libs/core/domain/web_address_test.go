package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Every way one video is written reaches the same address, so the words fetched
// for it are held once however the person came by the link.
func TestOneVideoIsOneAddressHoweverItWasWritten(t *testing.T) {
	const one = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	for _, written := range []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/watch?v=dQw4w9WgXcQ",
		"http://m.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=dQw4w9WgXcQ&list=RDdQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?si=Ab1Cd2Ef3",
		"https://www.youtube.com/embed/dQw4w9WgXcQ",
		"https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ",
		"https://www.youtube.com/live/dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=90s",
		"  https://www.youtube.com/watch?v=dQw4w9WgXcQ#top  ",
	} {
		got, err := domain.ParseWebAddress(written)
		if err != nil {
			t.Errorf("%q: %v", written, err)
			continue
		}
		if got.URL != one {
			t.Errorf("%q reached %q, want %q", written, got.URL, one)
		}
		if got.Video != "dQw4w9WgXcQ" {
			t.Errorf("%q names the video %q", written, got.Video)
		}
		if !got.IsVideo() {
			t.Errorf("%q is not a video", written)
		}
	}
}

// A page of the same site is a page. What tells them apart is the shape of the
// path and an identifier of the length one has, and nothing else.
func TestAPageOfTheSiteIsNotAVideo(t *testing.T) {
	for _, written := range []string{
		"https://www.youtube.com/",
		"https://www.youtube.com/feed/subscriptions",
		"https://www.youtube.com/@somebody",
		"https://www.youtube.com/watch?v=tooshort",
		"https://www.youtube.com/watch?v=waytoolongforanidentifier",
		"https://www.youtube.com/embed/not+an+identifier",
	} {
		got, err := domain.ParseWebAddress(written)
		if err != nil {
			t.Errorf("%q: %v", written, err)
			continue
		}
		if got.IsVideo() {
			t.Errorf("%q names the video %q", written, got.Video)
		}
	}
}

// A page keeps what says which page it is and loses what says where the person
// came from.
func TestAPageKeepsWhatSaysWhichPageItIs(t *testing.T) {
	for _, one := range []struct {
		written string
		want    string
	}{
		{"https://example.com/a?utm_source=news&id=7", "https://example.com/a?id=7"},
		{"https://example.com/a?fbclid=xyz", "https://example.com/a"},
		{"https://example.com/a?b=2&a=1", "https://example.com/a?a=1&b=2"},
		{"HTTPS://EXAMPLE.COM/Path", "https://example.com/Path"},
		{"https://example.com:443/a", "https://example.com/a"},
		{"http://example.com:80/a", "http://example.com/a"},
		{"https://example.com/a#section", "https://example.com/a"},
	} {
		got, err := domain.ParseWebAddress(one.written)
		if err != nil {
			t.Errorf("%q: %v", one.written, err)
			continue
		}
		if got.URL != one.want {
			t.Errorf("%q reached %q, want %q", one.written, got.URL, one.want)
		}
	}
}

// What a link note may point at is what can be fetched over the network. A
// scheme reaching this machine is refused where it is read, so nothing later
// has to remember to.
func TestOnlyWhatCanBeFetchedIsAnAddress(t *testing.T) {
	for _, written := range []string{
		"",
		"   ",
		"file:///etc/passwd",
		"ftp://example.com/a",
		"note://01J8F3K2M9QRSTVWXYZ0123456",
		"javascript:alert(1)",
		"data:text/html,<b>hi</b>",
		"example.com/a",
		"https://",
		"https:///a",
		"http://\x00",
	} {
		if _, err := domain.ParseWebAddress(written); !errors.Is(err, domain.ErrNotAWebAddress) {
			t.Errorf("%q was read as an address", written)
		}
	}
}

// Reading an address twice reaches what reading it once did, so a note rewritten
// from what it holds keeps the artifacts it names.
func TestReadingWhatWasReadReachesTheSame(t *testing.T) {
	for _, written := range []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://example.com/a?b=2&a=1",
		"https://example.com/",
	} {
		once, err := domain.ParseWebAddress(written)
		if err != nil {
			t.Fatalf("%q: %v", written, err)
		}
		twice, err := domain.ParseWebAddress(once.URL)
		if err != nil {
			t.Fatalf("%q: %v", once.URL, err)
		}
		if once != twice {
			t.Errorf("%q reached %v, and reading that reached %v", written, once, twice)
		}
	}
}
