package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func addLinkReadTools(server *sdk.Server, core Core) {
	addLinkList(server, core)
}

type linkListInput struct {
	Path string `json:"path" jsonschema:"the note to ask about"`
}

type linkListOutput struct {
	Links     []ResolvedLink `json:"links"`
	Backlinks []ResolvedLink `json:"backlinks"`
}

func addLinkList(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "link_list",
		Title: "List links",
		Description: "What one note points at and what points at it. A link is a " +
			"backlink because it resolves here, whichever end wrote it.",
	}, newHandler(func(ctx context.Context, in linkListInput) (linkListOutput, error) {
		found, err := core.Notes.Links.Execute(ctx, core.getCurrentVault().Vault, in.Path)
		if err != nil {
			return linkListOutput{}, err
		}
		return linkListOutput{Links: linksOf(found.Links), Backlinks: linksOf(found.Backlinks)}, nil
	}))
}

func addLinkWriteTools(server *sdk.Server, core Core) {
	addLinkAdd(server, core)
	addLinkUpdate(server, core)
	addLinkRemove(server, core)
}

type linkAddInput struct {
	Links []Addition `json:"links" jsonschema:"the relationships to write"`
}

type linkAddOutput struct {
	Added []AddOutcome `json:"added"`
}

func addLinkAdd(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "link_add",
		Title: "Add links",
		Description: "Join notes to other notes. A link is written in the note it goes " +
			"from and shows at both ends. `parent` and `child` are the hierarchy the " +
			"product draws; `jump` is a shortcut across it; `ref` is a plain mention. " +
			"Write the target as the other note's name — a name follows a note that " +
			"moves. Links going into the same note are written together, in one change " +
			"the person sees once, so every link of one note carries the same " +
			"fingerprint. The fingerprint from `note_read` is required, and a write " +
			"lands only on the note that fingerprint names. A note being made takes " +
			"its links in note_create instead, so that it never exists unjoined.",
	}, newHandler(func(ctx context.Context, in linkAddInput) (linkAddOutput, error) {
		return addLinks(ctx, core, in)
	}))
}

func addLinks(ctx context.Context, core Core, in linkAddInput) (linkAddOutput, error) {
	if len(in.Links) > maxRefs {
		return linkAddOutput{}, fmt.Errorf("write at most %d links at a time", maxRefs)
	}
	// The count is checked for the whole call before any file is opened.
	size := 0
	for _, add := range in.Links {
		size += countLinkBytes(add.Link)
	}
	if size > maxBytes {
		return linkAddOutput{}, fmt.Errorf(
			"a call writing %d bytes is more than this carries at once, which is %d", size, maxBytes)
	}

	additions := make([]note.LinkAddition, 0, len(in.Links))
	for _, add := range in.Links {
		baseline, err := parseFingerprint(add.Fingerprint)
		additions = append(additions, note.LinkAddition{
			From: add.From, Fingerprint: baseline, Link: newDomainLink(add.Link), FingerprintError: err,
		})
	}
	added, err := core.Notes.AddLinks.Execute(ctx, core.getCurrentVault().Vault, additions)
	if err != nil {
		return linkAddOutput{}, err
	}

	outcomes := make([]AddOutcome, 0, len(in.Links))
	for i, add := range in.Links {
		outcome := AddOutcome{From: add.From, To: add.To}
		if added[i].Error != nil {
			outcome.Error = sayError(added[i].Error)
		} else {
			outcome.Fingerprint = fingerprintOf(added[i].Fingerprint)
		}
		outcomes = append(outcomes, outcome)
	}
	return linkAddOutput{Added: outcomes}, nil
}

type linkUpdateInput struct {
	From        string `json:"from" jsonschema:"the path of the note the link is written in"`
	To          string `json:"to" jsonschema:"the target as it is written now"`
	Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses a write over somebody else's edit"`
	Role        string `json:"role,omitempty" jsonschema:"the role it should carry: parent, child, jump, ref or attachment"`
	Type        string `json:"type,omitempty" jsonschema:"leave this out: a value is introduced together with the code that reads it, and none is defined yet"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship"`
	Why         string `json:"note,omitempty" jsonschema:"why the link exists"`
}

func addLinkUpdate(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "link_update",
		Title: "Update links",
		Description: "Change what an existing link says about itself — its role, its " +
			"type, its label, why it exists — without changing where it goes. Fields " +
			"left out are cleared; role left out is kept. Because a field left out is " +
			"cleared, what this writes is decided from what you read: the fingerprint " +
			"from `note_read` is required, and the change lands only on the note that " +
			"fingerprint names.",
	}, newHandler(func(ctx context.Context, in linkUpdateInput) (ChangeOutcome, error) {
		baseline, err := parseFingerprint(in.Fingerprint)
		if err != nil {
			return ChangeOutcome{}, err
		}
		content, err := core.Notes.Linking.Update(
			ctx, core.getCurrentVault().Vault, in.From, domain.ParseAddress(in.To), domain.Link{
				Role:  domain.LinkRole(in.Role),
				Type:  in.Type,
				Label: in.Label,
				Why:   in.Why,
			}, baseline)
		if err != nil {
			return ChangeOutcome{}, err
		}
		return ChangeOutcome{Path: in.From, Fingerprint: fingerprintOf(content)}, nil
	}))
}

type linkRemoveInput struct {
	From        string `json:"from" jsonschema:"the path of the note the link is written in"`
	To          string `json:"to" jsonschema:"the target as it is written"`
	Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses a write over somebody else's edit"`
	Role        string `json:"role,omitempty" jsonschema:"only remove the link carrying this role; every role by default"`
}

func addLinkRemove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "link_remove",
		Title: "Remove links",
		Description: "Take a link out of the note it is written in. The note at the " +
			"other end is untouched: what goes is one end's account of the " +
			"relationship. The fingerprint from `note_read` is required, and the " +
			"removal lands only on the note that fingerprint names.",
	}, newHandler(func(ctx context.Context, in linkRemoveInput) (ChangeOutcome, error) {
		baseline, err := parseFingerprint(in.Fingerprint)
		if err != nil {
			return ChangeOutcome{}, err
		}
		content, err := core.Notes.Linking.Remove(ctx, core.getCurrentVault().Vault, in.From,
			domain.ParseAddress(in.To), domain.LinkRole(in.Role), baseline)
		if err != nil {
			return ChangeOutcome{}, err
		}
		return ChangeOutcome{Path: in.From, Fingerprint: fingerprintOf(content)}, nil
	}))
}
