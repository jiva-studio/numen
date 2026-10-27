package address_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

// webSeeds are the shapes an address is written in: the two schemes that are
// fetched and several that are not, a video written every way it is shared, a
// page carrying what says where a visitor came from, an address with credentials
// in it, and bytes no address is written in.
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
// A link note is written by hand in a file somebody else's editor wrote, so what
// is put through here is a stranger's bytes.
func FuzzParseWebAddress(f *testing.F) {
	for _, seed := range webSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := address.Parse(raw)
		if err != nil {
			return
		}

		// Nothing here reaches this machine, and nothing carries a name and a
		// password into a run's arguments.
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

		// The space around an address is whoever wrote it down's.
		if spaced, err := address.Parse(" \t" + raw + "\n "); err != nil || spaced != parsed {
			t.Fatalf("%q is %+v, and the same with space around it is %+v (%v)",
				raw, parsed, spaced, err)
		}
	})
}
