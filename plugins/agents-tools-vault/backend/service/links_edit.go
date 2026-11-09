package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// LinkAddition is one link to write into the frontmatter of a note.
type LinkAddition struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note; the same for every link written into one note"`
	Address     string `json:"address" jsonschema:"the other note's name, or note://<identifier> when the name is ambiguous"`
	Role        string `json:"role" jsonschema:"the kind of relationship: parent, child, jump, ref or attachment"`
	Type        string `json:"type,omitempty" jsonschema:"an open type of the link, such as source"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship"`
	Note        string `json:"note,omitempty" jsonschema:"why the link exists"`
}

// LinkAddInput is the input of the link_add tool.
type LinkAddInput struct {
	Links []LinkAddition `json:"links" jsonschema:"the links to write, at most 50"`
}

// LinkAddItem is the outcome of writing one link.
type LinkAddItem struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as given"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"fingerprint of the note after the write, for its next write; absent when the link was not written"`
	Error       string `json:"error,omitempty" jsonschema:"error message when the link was not written"`
}

// LinkAddOutput is the output of the link_add tool.
type LinkAddOutput struct {
	Results []LinkAddItem `json:"results" jsonschema:"one entry per link, in the order given"`
}

// LinkUpdateInput is the input of the link_update tool.
type LinkUpdateInput struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as it is written now"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note"`
	Role        string `json:"role,omitempty" jsonschema:"the link role: parent, child, jump, ref or attachment; kept when left out"`
	Type        string `json:"type,omitempty" jsonschema:"the type of the link; kept when left out"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship; kept when left out"`
	Note        string `json:"note,omitempty" jsonschema:"why the link exists; kept when left out"`
}

// LinkRemoveInput is the input of the link_remove tool.
type LinkRemoveInput struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as it is written"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note"`
	Role        string `json:"role,omitempty" jsonschema:"remove only the link of this role; every link to the target when left out"`
}

// LinkEditOutput is the output of the link_update and link_remove tools.
type LinkEditOutput struct {
	Path        string `json:"path" jsonschema:"path of the note written"`
	Fingerprint string `json:"fingerprint" jsonschema:"fingerprint of the note after the write, for its next write"`
}

func (t *Tools) addLinkAdd(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "link_add",
		Title: "Add Links",
		Description: "Write up to 50 links into the frontmatter links of notes. Each link needs the fingerprint " +
			"note_read returned for its note; the links for one note are written together and need the same " +
			"fingerprint. With a stale fingerprint, no link of that note is written and the note is unchanged.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{linkEditSentence},
	}, inVault(t, (*Tools).addLinks))
}

func (t *Tools) addLinkUpdate(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "link_update",
		Title: "Update Link",
		Description: "Change the role, type, label or note of the links a note has to a target, keeping the " +
			"target. Fields left out are kept. Needs the fingerprint note_read returned " +
			"for the note; a stale one is rejected.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{linkEditSentence},
	}, inVault(t, (*Tools).updateLink))
}

func (t *Tools) addLinkRemove(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Remove,
		Name:  "link_remove",
		Title: "Remove Links",
		Description: "Remove the links a note has to a target from its frontmatter, or only the link of one " +
			"role. The note at the other end is unchanged. Needs the fingerprint note_read returned for the note, " +
			"and a stale one is rejected.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{linkEditSentence},
	}, inVault(t, (*Tools).removeLinks))
}

func readAddition(in LinkAddition) (host.LinkAddition, error) {
	link := domain.Link{
		Target: domain.ParseAddress(in.Address),
		Role:   domain.LinkRole(in.Role),
		Type:   in.Type,
		Why:    in.Note,
		Label:  in.Label,
		Marker: domain.MarkerNone,
	}
	if err := checkWritable(link); err != nil {
		return host.LinkAddition{}, err
	}
	baseline, err := parseFingerprint(in.Fingerprint)
	if err != nil {
		return host.LinkAddition{}, err
	}
	baseline.Path = in.From
	return host.LinkAddition{From: in.From, Baseline: baseline, Link: link}, nil
}

func readBaseline(notePath, fingerprint string) (domain.FileStat, error) {
	if strings.TrimSpace(notePath) == "" {
		return domain.FileStat{}, errors.New("from is required")
	}
	baseline, err := parseFingerprint(fingerprint)
	if err != nil {
		return domain.FileStat{}, err
	}
	baseline.Path = notePath
	return baseline, nil
}

func checkWritable(link domain.Link) error {
	if !domain.IsKnownRole(link.Role) {
		return fmt.Errorf("invalid role %q: use parent, child, jump, ref or attachment", link.Role)
	}
	if !domain.IsWritableMarker(link.Marker) {
		return fmt.Errorf("invalid marker %q", link.Marker)
	}
	return nil
}

func checkRole(role domain.LinkRole) error {
	if role != "" && !domain.IsKnownRole(role) {
		return fmt.Errorf("invalid role %q: use parent, child, jump, ref or attachment", role)
	}
	return nil
}

func (t *Tools) addLinks(ctx context.Context, in LinkAddInput) (LinkAddOutput, error) {
	if len(in.Links) == 0 {
		return LinkAddOutput{}, errors.New("links is required")
	}
	out := LinkAddOutput{Results: make([]LinkAddItem, len(in.Links))}
	additions := make([]host.LinkAddition, 0, len(in.Links))
	positions := make([]int, 0, len(in.Links))
	for i, l := range in.Links {
		out.Results[i] = LinkAddItem{From: l.From, Address: l.Address, Fingerprint: "", Error: ""}
		addition, err := readAddition(l)
		if err != nil {
			out.Results[i].Error = err.Error()
			continue
		}
		additions = append(additions, addition)
		positions = append(positions, i)
	}
	if len(additions) == 0 {
		return out, nil
	}
	added, err := t.vault.Editor.AddLinks(ctx, additions)
	if err != nil {
		return LinkAddOutput{}, fmt.Errorf("write the links: %w", err)
	}
	if len(added) != len(additions) {
		return LinkAddOutput{}, fmt.Errorf("the host returned %d of %d links", len(added), len(additions))
	}
	for j, one := range added {
		i := positions[j]
		if one.Err != nil {
			out.Results[i].Error = describeWriteError(in.Links[i].From, one.Err).Error()
			continue
		}
		out.Results[i].Fingerprint = formatFingerprint(one.Stat)
	}
	return out, nil
}

func describeWriteError(notePath string, err error) error {
	if errors.Is(err, host.ErrStale) {
		return fmt.Errorf("%s changed since note_read returned the fingerprint; read it again: %w", notePath, err)
	}
	return err
}

func (t *Tools) updateLink(ctx context.Context, in LinkUpdateInput) (LinkEditOutput, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return LinkEditOutput{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := checkRole(role); err != nil {
		return LinkEditOutput{}, err
	}
	baseline, err := readBaseline(in.From, in.Fingerprint)
	if err != nil {
		return LinkEditOutput{}, err
	}
	change := domain.Link{Role: role, Type: in.Type, Why: in.Note, Label: in.Label, Marker: domain.MarkerNone}
	stat, err := t.vault.Editor.UpdateLinks(ctx, in.From, target, change, baseline)
	if err != nil {
		return LinkEditOutput{}, describeWriteError(in.From, err)
	}
	return LinkEditOutput{Path: in.From, Fingerprint: formatFingerprint(stat)}, nil
}

func (t *Tools) removeLinks(ctx context.Context, in LinkRemoveInput) (LinkEditOutput, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return LinkEditOutput{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := checkRole(role); err != nil {
		return LinkEditOutput{}, err
	}
	baseline, err := readBaseline(in.From, in.Fingerprint)
	if err != nil {
		return LinkEditOutput{}, err
	}
	stat, err := t.vault.Editor.RemoveLinks(ctx, in.From, target, role, baseline)
	if err != nil {
		return LinkEditOutput{}, describeWriteError(in.From, err)
	}
	return LinkEditOutput{Path: in.From, Fingerprint: formatFingerprint(stat)}, nil
}
