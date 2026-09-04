package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func addLinkTools(server *sdk.Server, core Core) {
	addLinkReadingTools(server, core)
	addLinkWritingTools(server, core)
}

func addLinkReadingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "link_list",
		Title: "List links",
		Description: "What one note points at and what points at it. A link is a " +
			"backlink because it resolves here, whichever end wrote it.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the note to ask about"`
	}) (*sdk.CallToolResult, struct {
		Links     []Link `json:"links"`
		Backlinks []Link `json:"backlinks"`
	}, error) {
		type out = struct {
			Links     []Link `json:"links"`
			Backlinks []Link `json:"backlinks"`
		}
		found, err := core.Notes.Links.Execute(ctx, core.shown().Vault, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Links: linksOf(found.Links), Backlinks: linksOf(found.Backlinks)}, nil
	})
}

func addLinkWritingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "link_add",
		Title: "Add links",
		Description: "Join notes to other notes. A link is written in the note it goes " +
			"from and shows at both ends. `parent` and `child` are the hierarchy the " +
			"product draws; `jump` is a shortcut across it; `ref` is a plain mention. " +
			"Write the target as the other note's name — a name follows a note that " +
			"moves. Links going into the same note are written together, in one change " +
			"the person sees once. A note being made takes its links in note_create " +
			"instead, so that it never exists unjoined.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Links []Addition `json:"links" jsonschema:"the relationships to write"`
	}) (*sdk.CallToolResult, struct {
		Added []AddOutcome `json:"added"`
	}, error) {
		type out = struct {
			Added []AddOutcome `json:"added"`
		}
		if len(in.Links) > maxRefs {
			return nil, out{}, fmt.Errorf("write at most %d links at a time", maxRefs)
		}
		// Asked of the whole call, before a file is opened.
		size := 0
		for _, add := range in.Links {
			size += carried(add.NewLink)
		}
		if size > maxBytes {
			return nil, out{}, fmt.Errorf(
				"a call writing %d bytes is more than this carries at once, which is %d", size, maxBytes)
		}

		res := out{Added: make([]AddOutcome, 0, len(in.Links))}
		// A malformed link is set aside before the grouping, so it costs only
		// itself. What is left is grouped: links sharing a note share a write.
		var order []string
		batches := map[string][]domain.Link{}
		at := map[string][]int{}
		for i, add := range in.Links {
			res.Added = append(res.Added, AddOutcome{From: add.From, To: add.To})
			link := writes(add.NewLink)
			if err := note.Writable(link); err != nil {
				res.Added[i].Refused = refusing(err)
				continue
			}
			if _, seen := batches[add.From]; !seen {
				order = append(order, add.From)
			}
			batches[add.From] = append(batches[add.From], link)
			at[add.From] = append(at[add.From], i)
		}

		// One note's links are one write: they land or fail together, and each
		// carries the same reason. Another note's write is untouched by it.
		for _, from := range order {
			if err := ctx.Err(); err != nil {
				return nil, out{}, err
			}
			group := batches[from]
			err := core.Notes.Linking.Add(ctx, core.shown().Vault, from, group[0], group[1:]...)
			if err == nil {
				continue
			}
			for _, i := range at[from] {
				res.Added[i].Refused = refusing(err)
			}
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "link_update",
		Title: "Change links",
		Description: "Change what an existing link says about itself — its role, its " +
			"type, its label, why it exists — without changing where it goes. Fields " +
			"left out are cleared; role left out is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		From  string `json:"from" jsonschema:"the path of the note the link is written in"`
		To    string `json:"to" jsonschema:"the target as it is written now"`
		Role  string `json:"role,omitempty" jsonschema:"the role it should carry: parent, child, jump, ref or attachment"`
		Type  string `json:"type,omitempty" jsonschema:"leave this out: a value is introduced together with the code that reads it, and none is defined yet"`
		Label string `json:"label,omitempty" jsonschema:"a few words naming the relationship"`
		Why   string `json:"note,omitempty" jsonschema:"why the link exists"`
	}) (*sdk.CallToolResult, Done, error) {
		err := core.Notes.Linking.Update(ctx, core.shown().Vault, in.From, domain.ParseAddress(in.To), domain.Link{
			Role:  domain.LinkRole(in.Role),
			Type:  in.Type,
			Label: in.Label,
			Why:   in.Why,
		})
		return nil, Done{Path: in.From}, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "link_remove",
		Title: "Remove links",
		Description: "Take a link out of the note it is written in. The note at the " +
			"other end is untouched: what goes is one end's account of the relationship.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		From string `json:"from" jsonschema:"the path of the note the link is written in"`
		To   string `json:"to" jsonschema:"the target as it is written"`
		Role string `json:"role,omitempty" jsonschema:"only remove the link carrying this role; every role by default"`
	}) (*sdk.CallToolResult, Done, error) {
		err := core.Notes.Linking.Remove(ctx, core.shown().Vault, in.From,
			domain.ParseAddress(in.To), domain.LinkRole(in.Role))
		return nil, Done{Path: in.From}, err
	})
}

// Link is one relationship as a tool reports it: what was written, and what it
// currently reaches.
type Link struct {
	From string `json:"from" jsonschema:"the note the link is written in"`
	To   string `json:"to,omitempty" jsonschema:"the note it reaches now, absent when it reaches nothing"`
	// Address is what the file actually says, which is not the same question.
	Address   string `json:"address"`
	Role      string `json:"role"`
	Type      string `json:"type,omitempty"`
	Label     string `json:"label,omitempty"`
	Why       string `json:"note,omitempty"`
	Ambiguous bool   `json:"ambiguous,omitempty" jsonschema:"more than one note answers to this name, and it reached the nearest"`
}

func linksOf(links []domain.ResolvedLink) []Link {
	out := make([]Link, 0, len(links))
	for _, l := range links {
		out = append(out, Link{
			From:      l.From,
			To:        l.To,
			Address:   l.Target.String(),
			Role:      string(l.Role),
			Type:      l.Type,
			Label:     l.Label,
			Why:       l.Why,
			Ambiguous: l.Ambiguous,
		})
	}
	return out
}

// Done is what a tool that changed one note says: which note it was.
type Done struct {
	Path string `json:"path"`
}

// Addition is one relationship to write, and the note it is written in.
type Addition struct {
	From string `json:"from" jsonschema:"the path of the note the link is written in"`
	NewLink
}

// AddOutcome is what happened to one link in a batch. Refused is empty when it
// was written.
type AddOutcome struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Refused string `json:"refused,omitempty" jsonschema:"why this one was not written, empty when it was"`
}
