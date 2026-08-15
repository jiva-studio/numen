package domain

import "strings"

// LinkRole says what a link is for. The list is closed: navigation and rendering
// read it, so a role nobody decided on has no behaviour.
type LinkRole string

const (
	RoleParent     LinkRole = "parent"
	RoleChild      LinkRole = "child"
	RoleJump       LinkRole = "jump"
	RoleRef        LinkRole = "ref"
	RoleAttachment LinkRole = "attachment"
)

// KnownRole reports whether a role is one the application acts on. An unknown
// role in a file is a problem to show rather than a value to store.
func KnownRole(r LinkRole) bool {
	switch r {
	case RoleParent, RoleChild, RoleJump, RoleRef, RoleAttachment:
		return true
	}
	return false
}

// Address is what a link points at. Every address has a scheme, so reading one
// is always a split rather than a guess: a title like "Lecture 3: entropy" looks
// like an unknown scheme, and a rule of the form "no prefix means a name" gets
// it wrong.
type Address struct {
	Scheme string
	Value  string
}

// Schemes. `name` never appears in a file: it is what a `[[wikilink]]` becomes
// once it is stored.
const (
	SchemeName  = "name"
	SchemeNote  = "note"
	SchemeAsset = "asset"
)

func (a Address) String() string { return a.Scheme + "://" + a.Value }

// ParseAddress reads what was written in a file.
//
// A wikilink carries two things the address does not: an alias after `|`, which
// is how the link is displayed, and a fragment after `#`, which points at a
// block. Both are dropped here — nothing reads them yet, and storing what
// nothing reads is how a guess becomes permanent.
func ParseAddress(raw string) Address {
	target := strings.TrimSpace(raw)
	target = strings.TrimPrefix(target, "[[")
	target = strings.TrimSuffix(target, "]]")

	if alias := strings.IndexByte(target, '|'); alias >= 0 {
		target = target[:alias]
	}
	if fragment := strings.IndexByte(target, '#'); fragment >= 0 {
		target = target[:fragment]
	}
	target = strings.TrimSpace(target)

	if scheme, value, found := strings.Cut(target, "://"); found && isScheme(scheme) {
		return Address{Scheme: scheme, Value: value}
	}
	return Address{Scheme: SchemeName, Value: target}
}

// isScheme keeps a title with a colon in it from being read as an address. A
// scheme is letters, digits, `+`, `-` and `.`, and a name is anything else.
func isScheme(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '+', r == '-', r == '.':
		default:
			return false
		}
	}
	return true
}

// Link is one edge as it was written. Where it resolves to is a question asked
// of the index, not a fact stored in the file.
type Link struct {
	Target Address
	Role   LinkRole

	// Type is open vocabulary, and a value is only introduced together with the
	// code that reads it.
	Type string

	// Note is why the link exists, in the author's words.
	Note string

	// Label is what an identifier link shows to a human reading the file. It
	// takes no part in resolution and is the source of truth for nothing.
	Label string
}

// ResolvedLink is a link together with what it currently points at. "Currently"
// is the word that matters: resolution is a query, so adding a file can resolve
// a link that was dangling and removing one can break a link that worked.
type ResolvedLink struct {
	Link

	// From is the note the link is written in; To is the note it resolves to,
	// empty when nothing does.
	From string
	To   string

	// Ambiguous is set when several notes answer to the name. The link still
	// resolves — to the nearest one — but the vault has a question in it.
	Ambiguous bool
}
