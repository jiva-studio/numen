package domain_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// addressSeeds are the shapes a link is written in: a bare name, a name with an
// alias, one with a fragment, both together, the two schemes the index writes,
// a title carrying a colon, a name that is only punctuation, and the empty
// string.
var addressSeeds = []string{
	"Entropy",
	"[[Entropy]]",
	"[[Entropy|the measure]]",
	"[[Entropy#Recognise]]",
	"[[Entropy#Recognise|the measure]]",
	"note://01J8F3K2M9QRSTVWXYZ0123456",
	"asset://images/plate.png",
	"TCP/IP: a history",
	"://",
	"|#",
	"[[",
	"]]",
	"  ",
	"",
	"\x00\xff",
}

// Every string is an address of something: addressing is total, so no input
// string yields an undefined result or none.
//
// A link in a note file can be typed manually or produced by another editor, so
// the input here is arbitrary bytes.
func FuzzParseAddress(f *testing.F) {
	for _, seed := range addressSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		a := domain.ParseAddress(raw)
		switch {
		case a.Scheme == "":
			t.Fatalf("%q is an address of no scheme", raw)
		case strings.ContainsAny(a.Value, "|#"):
			// An alias and a fragment are read off the link and are no part of
			// what it points at.
			t.Fatalf("%q points at %q, which carries an alias or a fragment",
				raw, a.Value)
		}

		// The space around a link is the sentence's and not the link's.
		if spaced := domain.ParseAddress(" \t" + raw + "\n "); spaced != a {
			t.Fatalf("%q is %+v, and the same with space around it is %+v",
				raw, a, spaced)
		}

		// A name is written as itself and a scheme carries its own name, so the
		// written form of a link identifies which of the two it is.
		if content := a.GetWritten(); a.Scheme == domain.SchemeName {
			if content != a.Value {
				t.Fatalf("the name %q is written as %q", a.Value, content)
			}
		} else if !strings.HasPrefix(content, a.Scheme+"://") {
			t.Fatalf("%+v is written as %q", a, content)
		}
	})
}
