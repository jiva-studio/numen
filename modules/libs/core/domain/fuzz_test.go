package domain_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
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

// Every string is an address of something: the addressing is total, so no
// string of a file gives an undefined answer or none.
//
// A note's links are written by hand in a file somebody else's editor wrote, so
// what is put through here is a stranger's bytes.
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

		// A name is written as itself and a scheme carries its own name, so
		// what a link is written as says which of the two it is.
		if written := a.Written(); a.Scheme == domain.SchemeName {
			if written != a.Value {
				t.Fatalf("the name %q is written as %q", a.Value, written)
			}
		} else if !strings.HasPrefix(written, a.Scheme+"://") {
			t.Fatalf("%+v is written as %q", a, written)
		}
	})
}
