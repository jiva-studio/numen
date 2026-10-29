package address_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

// webSeeds are seed address shapes: the two fetched schemes and several
// unfetched ones, a video address in each sharing format, a page address with a
// tracking parameter, an address with credentials, and bytes that are not an
// address.
var webSeeds = []string{
	"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	"https://youtu.be/dQw4w9WgXcQ?si=Ab1Cd2Ef3&t=90",
	"http://example.com:80/a?utm_source=news#top",
	"https://user:secret@example.com/a",
	"https://пример.рф/статья",
	"file:///etc/passwd",
	"javascript:alert(1)",
	"note://01J8F3K2M9QRSTVWXYZ0123456",
	"https://",
	"://",
	"  ",
	"",
	"\x00\xff",
}

// hasUserInfo reports whether an address carries a name or a password before its host.
func hasUserInfo(raw string) bool {
	parsed, err := url.Parse(raw)
	return err != nil || parsed.User != nil
}

// back is fetchable and stays itself when it is read again.
//
// A link note is authored manually in a file that other tools also edit, so the input
// is untrusted bytes.
func FuzzParseWebAddress(f *testing.F) {
	for _, seed := range webSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := address.Parse(raw)
		if err != nil {
			return
		}

		// The parsed address must not refer to a local resource, and must not carry
		// a username and password into a run's arguments.
		switch {
		case !strings.HasPrefix(string(parsed), "http://") &&
			!strings.HasPrefix(string(parsed), "https://"):
			t.Fatalf("%q was read as %q", raw, string(parsed))
		case strings.ContainsAny(string(parsed), "#\x00\n\r \t"):
			t.Fatalf("%q was read as %q", raw, string(parsed))
		case hasUserInfo(string(parsed)):
			t.Fatalf("%q was read as %q, which carries a name", raw, string(parsed))
		}

		// The one form is the form the name of every artifact is made from, so
		// reading it again has to reach it again.
		again, err := address.Parse(string(parsed))
		if err != nil {
			t.Fatalf("%q was read as %q, which is no address", raw, string(parsed))
		}
		if again != parsed {
			t.Fatalf("%q was read as %+v, and that as %+v", raw, parsed, again)
		}

		// Surrounding whitespace is not part of the address.
		if spaced, err := address.Parse(" \t" + raw + "\n "); err != nil || spaced != parsed {
			t.Fatalf("%q is %+v, and the same with space around it is %+v (%v)",
				raw, parsed, spaced, err)
		}
	})
}
