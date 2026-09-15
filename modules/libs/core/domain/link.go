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

// IsKnownRole reports whether a role is one the application acts on. An unknown
// role in a file is shown as a problem.
func IsKnownRole(r LinkRole) bool {
	switch r {
	case RoleParent, RoleChild, RoleJump, RoleRef, RoleAttachment:
		return true
	}
	return false
}

// Address is what a link points at. Every address has a scheme, so reading one
// is a split at the first colon: a title like "Lecture 3: entropy" carries one
// of its own, and only a written scheme tells the two apart.
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

// GetWritten is an address as it goes into a file, and as it is shown to the
// person who wrote it. A name is written as itself: `name://` is how the index
// holds it and never appears in a note.
func (a Address) GetWritten() string {
	if a.Scheme == SchemeName {
		return a.Value
	}
	return a.String()
}

// ParseAddress reads what was written in a file.
//
// A wikilink carries two things the address does not: an alias after `|`, which
// is how the link is read in the sentence, and a fragment after `#`, which names
// a place inside the note. Both are dropped here.
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

	// Why the link exists, in the author's words.
	Why string

	// Label is what the person calls this relationship, in a few words, drawn
	// along the line between the two notes. It is theirs, not a copy of
	// anything: no field carries the target's name, because a name kept
	// alongside an address is a cache in the file the address is written in.
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

	// ToVault is the vault the target turned out to be in. Empty when the link
	// resolved to nothing, and equal to the vault the link was written in for
	// every ordinary link.
	ToVault VaultID

	// Ambiguous is set when several notes answer to the name. The link still
	// resolves — to the nearest one — but the vault has a question in it.
	Ambiguous bool
}

// InVault is the vault the link resolved into, which is not always the vault it
// was written in: an identifier names one note in the world, so a link may cross
// a boundary the user put there on purpose.
func (r ResolvedLink) InVault(writtenIn VaultID) (vault VaultID, crossed bool) {
	if r.ToVault == "" || r.ToVault == writtenIn {
		return writtenIn, false
	}
	return r.ToVault, true
}

// AmbiguousLink is a link that several notes answer to. It resolved to the
// nearest of them, and the rest are what makes it worth showing: which one it
// means is a fact about where the notes sit.
type AmbiguousLink struct {
	ResolvedLink
	// Candidates is every note that answers to the name, the chosen one among them.
	Candidates []string
}

// Nameable reports whether a filename can be written as an address that reaches
// it back.
//
// A file may be called almost anything, and an address may not. `#` starts a
// fragment, `|` starts an alias, `://` starts a scheme and `]]` ends a
// wikilink, so a note named with any of them cannot be reached by its name:
// the character is read as punctuation of the link. Such a note is addressed
// by its identifier or not at all.
func Nameable(name string) bool {
	if strings.TrimSpace(name) != name || name == "" || strings.ContainsAny(name, "\n\r") {
		return false
	}
	if strings.Contains(name, "[[") || strings.Contains(name, "]]") {
		return false
	}
	return ParseAddress(name) == Address{Scheme: SchemeName, Value: name}
}
