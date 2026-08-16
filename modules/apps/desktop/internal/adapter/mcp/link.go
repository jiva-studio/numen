package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

func addLinkTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "link_add",
		Description: "Join notes to other notes. A link is written in the note it goes " +
			"from and shows at both ends. `parent` and `child` are the hierarchy the " +
			"product draws; `jump` is a shortcut across it; `ref` is a plain mention. " +
			"Write the target as the other note's name — a name follows a note that " +
			"moves. Links going into the same note are written together, in one change " +
			"the person sees once. A note being made takes its links in note_create " +
			"instead, so that it never exists unjoined.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Links []Join `json:"links" jsonschema:"the relationships to write"`
	}) (*sdk.CallToolResult, struct {
		Added []JoinOutcome `json:"added"`
	}, error) {
		type out = struct {
			Added []JoinOutcome `json:"added"`
		}
		if len(in.Links) > maxRefs {
			return nil, out{}, fmt.Errorf("write at most %d links at a time", maxRefs)
		}

		res := out{Added: make([]JoinOutcome, 0, len(in.Links))}
		// A malformed link is set aside before the grouping, so it costs only
		// itself. What is left is grouped: links sharing a note share a write.
		var order []string
		batches := map[string][]domain.Link{}
		at := map[string][]int{}
		for i, join := range in.Links {
			res.Added = append(res.Added, JoinOutcome{From: join.From, To: join.To})
			link := written([]NewLink{join.NewLink})[0]
			if err := note.Writable(link); err != nil {
				res.Added[i].Refused = err.Error()
				continue
			}
			if _, seen := batches[join.From]; !seen {
				order = append(order, join.From)
			}
			batches[join.From] = append(batches[join.From], link)
			at[join.From] = append(at[join.From], i)
		}

		// One note's links are one write: they land or fail together, and each
		// carries the same reason. Another note's write is untouched by it.
		for _, from := range order {
			err := core.Linking.Add(ctx, core.Vault, from, batches[from]...)
			if err == nil {
				continue
			}
			for _, i := range at[from] {
				res.Added[i].Refused = err.Error()
			}
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: "link_update",
		Description: "Change what an existing link says about itself — its role, its " +
			"type, its label, why it exists — without changing where it goes. Fields " +
			"left out are cleared; role left out is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		From  string `json:"from" jsonschema:"the path of the note the link is written in"`
		To    string `json:"to" jsonschema:"the target as it is written now"`
		Role  string `json:"role,omitempty" jsonschema:"the role it should carry: parent, child, jump, ref or attachment"`
		Type  string `json:"type,omitempty" jsonschema:"what the link is for, as a feature reads it"`
		Label string `json:"label,omitempty" jsonschema:"a few words naming the relationship"`
		Note  string `json:"note,omitempty" jsonschema:"why the link exists"`
	}) (*sdk.CallToolResult, Done, error) {
		err := core.Linking.Update(ctx, core.Vault, in.From, domain.ParseAddress(in.To), domain.Link{
			Role:  domain.LinkRole(in.Role),
			Type:  in.Type,
			Label: in.Label,
			Note:  in.Note,
		})
		return nil, Done{Path: in.From}, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: "link_remove",
		Description: "Take a link out of the note it is written in. The note at the " +
			"other end is untouched: what goes is one end's account of the relationship.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		From string `json:"from" jsonschema:"the path of the note the link is written in"`
		To   string `json:"to" jsonschema:"the target as it is written"`
		Role string `json:"role,omitempty" jsonschema:"only remove the link carrying this role; every role by default"`
	}) (*sdk.CallToolResult, Done, error) {
		err := core.Linking.Remove(ctx, core.Vault, in.From,
			domain.ParseAddress(in.To), domain.LinkRole(in.Role))
		return nil, Done{Path: in.From}, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: "link_list",
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
		found, err := core.Links.Execute(ctx, core.Vault, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Links: linksOf(found.Links), Backlinks: linksOf(found.Backlinks)}, nil
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
	Note      string `json:"note,omitempty"`
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
			Note:      l.Note,
			Ambiguous: l.Ambiguous,
		})
	}
	return out
}

// Done is what a tool that changed one note says: which note it was.
type Done struct {
	Path string `json:"path"`
}

// Join is one relationship to write, and the note it is written in.
type Join struct {
	From string `json:"from" jsonschema:"the path of the note the link is written in"`
	NewLink
}

// JoinOutcome is what happened to one link in a batch. Refused is empty when it
// was written.
type JoinOutcome struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Refused string `json:"refused,omitempty" jsonschema:"why this one was not written, empty when it was"`
}
