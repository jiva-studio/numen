package mcp

import (
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func addLinkTools(server *sdk.Server, core Core) {
	addLinkReadTools(server, core)
	addLinkWriteTools(server, core)
}

// ResolvedLink is one relationship as a tool reports it: what was written, and
// what it currently reaches.
type ResolvedLink struct {
	From string `json:"from" jsonschema:"the note the link is written in"`
	To   string `json:"to,omitempty" jsonschema:"the note it reaches now, absent when it reaches nothing"`
	// Address is what the file actually says, which is not the same question.
	Address     string `json:"address"`
	Role        string `json:"role"`
	Type        string `json:"type,omitempty"`
	Label       string `json:"label,omitempty"`
	Why         string `json:"note,omitempty"`
	IsAmbiguous bool   `json:"ambiguous,omitempty" jsonschema:"more than one note answers to this name, and it reached the nearest"`
}

func linksOf(links []domain.ResolvedLink) []ResolvedLink {
	out := make([]ResolvedLink, 0, len(links))
	for _, l := range links {
		out = append(out, ResolvedLink{
			From:        l.From,
			To:          l.To,
			Address:     l.Target.String(),
			Role:        string(l.Role),
			Type:        l.Type,
			Label:       l.Label,
			Why:         l.Why,
			IsAmbiguous: l.IsAmbiguous,
		})
	}
	return out
}

// Link is a relationship written into a note, whether the note is being made or
// is already there.
type Link struct {
	To    string `json:"to" jsonschema:"the other note's name, or note://<identifier> when the name is ambiguous"`
	Role  string `json:"role" jsonschema:"what kind of relationship this is: parent, child, jump, ref or attachment"`
	Type  string `json:"type,omitempty" jsonschema:"leave this out: a value is introduced together with the code that reads it, and none is defined yet"`
	Label string `json:"label,omitempty" jsonschema:"a few words naming the relationship, shown along the line"`
	Why   string `json:"note,omitempty" jsonschema:"why the link exists, in the person's words"`
}

// newDomainLink turns what was asked for into what the core writes.
func newDomainLink(l Link) domain.Link {
	return domain.Link{
		Target: domain.ParseAddress(l.To),
		Role:   domain.LinkRole(l.Role),
		Type:   l.Type,
		Label:  l.Label,
		Why:    l.Why,
	}
}

func newLinks(links []Link) []domain.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]domain.Link, 0, len(links))
	for _, l := range links {
		out = append(out, newDomainLink(l))
	}
	return out
}

// countLinkBytes is how many bytes a link will put in a file. Every field of one
// is written into the frontmatter, so every field is measured.
func countLinkBytes(l Link) int {
	return len(l.To) + len(l.Role) + len(l.Type) + len(l.Label) + len(l.Why)
}

// ChangeOutcome is what a tool that changed one note says: which note it was,
// and what to present at the next write of it.
type ChangeOutcome struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this note without reading it back"`
}

// Addition is one relationship to write, and the note it is written in.
type Addition struct {
	From string `json:"from" jsonschema:"the path of the note the link is written in"`
	// Fingerprint is that note as the caller read it. Links sharing a note are
	// one write, so they name one fingerprint between them.
	Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses a write over somebody else's edit; the same for every link written into one note"`
	Link
}

// AddOutcome is what happened to one link in a batch. Refused is empty when it
// was written.
type AddOutcome struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"what the note became, to hand to its next write; absent for a link that was not written"`
	Error string `json:"error,omitempty" jsonschema:"why this one was not written, empty when it was"`
}
