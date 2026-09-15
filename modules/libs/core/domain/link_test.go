package domain_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestARoleNobodyDecidedOnIsNotActedOn(t *testing.T) {
	for _, role := range []domain.LinkRole{
		domain.RoleParent, domain.RoleChild, domain.RoleJump, domain.RoleRef, domain.RoleAttachment,
	} {
		if !domain.IsKnownRole(role) {
			t.Errorf("%q is not acted on", role)
		}
	}
	for _, role := range []domain.LinkRole{"", "PARENT", "sibling", "see-also"} {
		if domain.IsKnownRole(role) {
			t.Errorf("%q is acted on", role)
		}
	}
}

// The addresses in a note travel on the wire as they were written, so the
// window reads them too. Both read this one corpus, and neither owns it.
const corpus = "../../protocol/testdata/addresses.json"

type written struct {
	Written string `json:"written"`
	Scheme  string `json:"scheme"`
	Value   string `json:"value"`
}

// A title carries a colon of its own, an alias and a fragment are not part of
// where a link points, and only a written scheme tells any of them apart.
func TestAnAddressIsReadTheWayTheSchemaSaysItIs(t *testing.T) {
	raw, err := os.ReadFile(corpus)
	if err != nil {
		t.Fatal(err)
	}
	var cases []written
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("the corpus is empty")
	}

	for _, one := range cases {
		want := domain.Address{Scheme: one.Scheme, Value: one.Value}
		if got := domain.ParseAddress(one.Written); got != want {
			t.Errorf("%q read as %v, want %v", one.Written, got, want)
		}
	}
}

func TestAnAddressSaysItselfBackTheWayItIsWritten(t *testing.T) {
	raw := "note://01ABC"
	if got := domain.ParseAddress(raw).String(); got != raw {
		t.Errorf("got %q", got)
	}
}

// A name that cannot be written as an address is reached by its identifier or
// not at all.
func TestANameIsReachableOrItIsNot(t *testing.T) {
	for _, name := range []string{"Entropy", "Lecture 3: entropy", "Zoë Brontë", "Холм"} {
		if !domain.Nameable(name) {
			t.Errorf("%q cannot be reached", name)
		}
	}
	// `#` starts a fragment, `|` an alias, `://` a scheme and `]]` ends a
	// wikilink: each is read as punctuation of the link.
	for _, name := range []string{
		"",
		" Entropy",
		"Entropy ",
		"Entropy\n",
		"a|b",
		"a#b",
		"a[[b",
		"a]]b",
		"note://01ABC",
	} {
		if domain.Nameable(name) {
			t.Errorf("%q is said to be reachable", name)
		}
	}
}

// An identifier names one note in the world, so a link may cross a boundary the
// person put there.
func TestALinkSaysWhichVaultItLandedIn(t *testing.T) {
	here := domain.ResolvedLink{ToVault: "one"}
	if vault, crossed := here.InVault("one"); vault != "one" || crossed {
		t.Errorf("got %q, crossed %v", vault, crossed)
	}
	unsaid := domain.ResolvedLink{}
	if vault, crossed := unsaid.InVault("one"); vault != "one" || crossed {
		t.Errorf("got %q, crossed %v", vault, crossed)
	}
	over := domain.ResolvedLink{ToVault: "two"}
	if vault, crossed := over.InVault("one"); vault != "two" || !crossed {
		t.Errorf("got %q, crossed %v", vault, crossed)
	}
}
