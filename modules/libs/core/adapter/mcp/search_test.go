package mcp

import "testing"

// The address of a passage is the one the window opens. A model asked to put a
// link together writes one that opens nothing as often as not, so the search
// hands it over already written, and this is the shape the window reads back.
//
// The window's own reading of it is `shared/links.ts:parseLinkTarget`, which
// takes the path with decodeURIComponent and the two numbers off the query.
func TestThePassageAddressIsTheOneTheWindowOpens(t *testing.T) {
	for _, one := range []struct {
		name   string
		source string
		start  int
		length int
		want   string
	}{
		{
			name:   "a folder and a space",
			source: "library/A Book.pdf",
			start:  62690,
			length: 1246,
			want:   "numen:library%2FA%20Book.pdf?start=62690&length=1246",
		},
		{
			name:   "a book named in another script",
			source: "Книги/Махабхарата.epub",
			start:  0,
			length: 40,
			want: "numen:%D0%9A%D0%BD%D0%B8%D0%B3%D0%B8%2F" +
				"%D0%9C%D0%B0%D1%85%D0%B0%D0%B1%D1%85%D0%B0%D1%80%D0%B0%D1%82%D0%B0.epub" +
				"?start=0&length=40",
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			if got := addressOf(one.source, one.start, one.length); got != one.want {
				t.Errorf("a passage of %q is at\n%s\nwant\n%s", one.source, got, one.want)
			}
		})
	}
}
