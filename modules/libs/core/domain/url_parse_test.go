package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

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
		got, err := domain.ParseURL(one.written)
		if err != nil {
			t.Errorf("%q: %v", one.written, err)
			continue
		}
		if string(got) != one.want {
			t.Errorf("%q reached %q, want %q", one.written, string(got), one.want)
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
		if _, err := domain.ParseURL(written); !errors.Is(err, domain.ErrNotAURL) {
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
		once, err := domain.ParseURL(written)
		if err != nil {
			t.Fatalf("%q: %v", written, err)
		}
		twice, err := domain.ParseURL(string(once))
		if err != nil {
			t.Fatalf("%q: %v", string(once), err)
		}
		if once != twice {
			t.Errorf("%q reached %v, and reading that reached %v", written, once, twice)
		}
	}
}

// A note points somewhere a browser would go, and this machine is not that.
// What listens here — an index, the window's own socket — answers nobody's
// paste, and an address reaching one is refused where every address is read.
func TestAnAddressOnThisMachineIsNoAddressToPointAt(t *testing.T) {
	for _, raw := range []string{
		"http://localhost:9200/_all/_search",
		"http://127.0.0.1:8080/",
		"https://[::1]/admin",
		"http://0.0.0.0:5432/",
		"http://numen.localhost/",
	} {
		if _, err := domain.ParseURL(raw); !errors.Is(err, domain.ErrNotAURL) {
			t.Errorf("%s is read as an address to point at", raw)
		}
	}
}
