package address_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
)

func TestAPagePreservesWhatReturnsWhichPageItIs(t *testing.T) {
	for _, one := range []struct {
		input string
		want  string
	}{
		{"https://example.com/a?utm_source=news&id=7", "https://example.com/a?id=7"},
		{"https://example.com/a?fbclid=xyz", "https://example.com/a"},
		{"https://example.com/a?b=2&a=1", "https://example.com/a?a=1&b=2"},
		{"HTTPS://EXAMPLE.COM/Path", "https://example.com/Path"},
		{"https://example.com:443/a", "https://example.com/a"},
		{"http://example.com:80/a", "http://example.com/a"},
		{"http://example.com:443/a", "http://example.com:443/a"},
		{"https://example.com:80/a", "https://example.com:80/a"},
		{"http://example.com:8080/a", "http://example.com:8080/a"},
		{"https://example.com/a#section", "https://example.com/a"},
	} {
		got, err := address.Parse(one.input)
		if err != nil {
			t.Errorf("%q: %v", one.input, err)
			continue
		}
		if string(got) != one.want {
			t.Errorf("%q reached %q, want %q", one.input, string(got), one.want)
		}
	}
}

// What a link note may point at is what can be fetched over the network. A
// has to remember to.
func TestOnlyWhatCanBeFetchedIsAnAddress(t *testing.T) {
	for _, content := range []string{
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
		if _, err := address.Parse(content); !errors.Is(err, address.ErrNotAURL) {
			t.Errorf("%q was read as an address", content)
		}
	}
}

// Reading an address twice returns the same result as reading it once, so a note rewritten
// from its own content keeps the artifacts it names.
func TestReadingWhatWasReadReachesTheSame(t *testing.T) {
	for _, content := range []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://example.com/a?b=2&a=1",
		"https://example.com/",
	} {
		once, err := address.Parse(content)
		if err != nil {
			t.Fatalf("%q: %v", content, err)
		}
		twice, err := address.Parse(string(once))
		if err != nil {
			t.Fatalf("%q: %v", string(once), err)
		}
		if once != twice {
			t.Errorf("%q reached %v, and reading that reached %v", content, once, twice)
		}
	}
}

func TestParseAcceptsAnAddressOnThisMachine(t *testing.T) {
	for _, one := range []struct {
		input string
		want  string
	}{
		{"http://localhost:9200/x", "http://localhost:9200/x"},
		{"http://127.0.0.1:8080/", "http://127.0.0.1:8080/"},
		{"https://[::1]/admin", "https://[::1]/admin"},
		{"http://0.0.0.0:5432/", "http://0.0.0.0:5432/"},
		{"http://numen.localhost/", "http://numen.localhost/"},
		{"HTTP://LOCALHOST:80/x", "http://localhost/x"},
		{"https://[::1]:443/admin", "https://[::1]/admin"},
	} {
		got, err := address.Parse(one.input)
		if err != nil {
			t.Errorf("%q: %v", one.input, err)
			continue
		}
		if string(got) != one.want {
			t.Errorf("%q parsed as %q, want %q", one.input, string(got), one.want)
		}
	}
}
