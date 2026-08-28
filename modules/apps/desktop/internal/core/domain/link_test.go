package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

func TestARoleNobodyDecidedOnIsNotActedOn(t *testing.T) {
	for _, role := range []domain.LinkRole{
		domain.RoleParent, domain.RoleChild, domain.RoleJump, domain.RoleRef, domain.RoleAttachment,
	} {
		if !domain.KnownRole(role) {
			t.Errorf("%q is not acted on", role)
		}
	}
	for _, role := range []domain.LinkRole{"", "PARENT", "sibling", "see-also"} {
		if domain.KnownRole(role) {
			t.Errorf("%q is acted on", role)
		}
	}
}

// A title carries a colon of its own, and only a written scheme tells the two
// apart.
func TestATitleWithAColonIsAName(t *testing.T) {
	for _, raw := range []string{
		"Lecture 3: entropy",
		"[[Lecture 3: entropy]]",
		"note:not-a-scheme",
		"http:/one-slash",
	} {
		got := domain.ParseAddress(raw)
		if got.Scheme != domain.SchemeName {
			t.Errorf("%q read as %v", raw, got)
		}
	}
}

func TestAWrittenSchemeIsRead(t *testing.T) {
	for raw, want := range map[string]domain.Address{
		"note://01ABC":        {Scheme: domain.SchemeNote, Value: "01ABC"},
		"[[note://01ABC]]":    {Scheme: domain.SchemeNote, Value: "01ABC"},
		"asset://picture.png": {Scheme: domain.SchemeAsset, Value: "picture.png"},
		"name://Entropy":      {Scheme: domain.SchemeName, Value: "Entropy"},
	} {
		if got := domain.ParseAddress(raw); got != want {
			t.Errorf("%q read as %v, want %v", raw, got, want)
		}
	}
}

// An alias and a fragment are not part of where a link points.
func TestAnAliasAndAFragmentAreNotPartOfTheAddress(t *testing.T) {
	want := domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}
	for _, raw := range []string{
		"[[Entropy|what disorder means]]",
		"[[Entropy#a-heading]]",
		"[[Entropy#a-heading|what disorder means]]",
		"  [[ Entropy ]]  ",
	} {
		if got := domain.ParseAddress(raw); got != want {
			t.Errorf("%q read as %v", raw, got)
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
