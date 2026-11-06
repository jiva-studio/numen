package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// LinkAddition is one link to write into the frontmatter of a note.
type LinkAddition struct {
	From        string `json:"from" jsonschema:"path of the note the link is written in"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read gave for the note; the same for every link written into one note"`
	To          string `json:"to" jsonschema:"the other note's name, or note://<identifier> when the name is ambiguous"`
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
	From        string `json:"from" jsonschema:"path of the note the link is written in"`
	To          string `json:"to" jsonschema:"the target as given"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"fingerprint of the note after the write, for its next write; absent when the link was not written"`
	Error       string `json:"error,omitempty" jsonschema:"why the link was not written"`
}

// LinkAddOutput is the output of the link_add tool.
type LinkAddOutput struct {
	Results []LinkAddItem `json:"results" jsonschema:"one entry per link, in the order given"`
}

// LinkUpdateInput is the input of the link_update tool.
type LinkUpdateInput struct {
	From        string `json:"from" jsonschema:"path of the note the link is written in"`
	To          string `json:"to" jsonschema:"the target as it is written now"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read gave for the note"`
	Role        string `json:"role,omitempty" jsonschema:"the role the link carries: parent, child, jump, ref or attachment; kept when left out"`
	Type        string `json:"type,omitempty" jsonschema:"the type of the link; kept when left out"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship; kept when left out"`
	Note        string `json:"note,omitempty" jsonschema:"why the link exists; kept when left out"`
}

// LinkRemoveInput is the input of the link_remove tool.
type LinkRemoveInput struct {
	From        string `json:"from" jsonschema:"path of the note the link is written in"`
	To          string `json:"to" jsonschema:"the target as it is written"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read gave for the note"`
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
			"note_read gave for its note; the links into one note are one write and carry the same fingerprint. " +
			"A stale fingerprint refuses the links of that note and leaves it unchanged.",
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
			"target. Fields left out are kept. Needs the fingerprint note_read gave " +
			"for the note, and a stale one is refused.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{linkEditSentence},
	}, inVault(t, (*Tools).updateLink))
}

func (t *Tools) addLinkRemove(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "link_remove",
		Title: "Remove Links",
		Description: "Remove the links a note has to a target from its frontmatter, or only the link of one " +
			"role. The note at the other end is unchanged. Needs the fingerprint note_read gave for the note, " +
			"and a stale one is refused.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{linkEditSentence},
	}, inVault(t, (*Tools).removeLinks))
}

// linkGroups is the links of one call by the note they are written in, in order of first appearance.
type linkGroups struct {
	order        []string
	links        map[string][]domain.Link
	fingerprints map[string]string
	positions    map[string][]int
	// conflicts is why a note is not written: its links carry two different fingerprints.
	conflicts map[string]error
}

// add groups the link at position i under its note, or returns why it cannot be written.
func (g *linkGroups) add(i int, in LinkAddition) error {
	link := domain.Link{
		Target: domain.ParseAddress(in.To),
		Role:   domain.LinkRole(in.Role),
		Type:   in.Type,
		Why:    in.Note,
		Label:  in.Label,
		Marker: domain.MarkerNone,
	}
	if err := checkWritable(link); err != nil {
		return err
	}
	if strings.TrimSpace(in.From) == "" {
		return errors.New("from is required")
	}
	if _, err := parseFingerprint(in.Fingerprint); err != nil {
		return err
	}
	if first, isGrouped := g.fingerprints[in.From]; isGrouped && first != in.Fingerprint {
		g.conflicts[in.From] = fmt.Errorf("the links into %s carry two different fingerprints, and they are one write", in.From)
		return g.conflicts[in.From]
	}
	if _, isGrouped := g.links[in.From]; !isGrouped {
		g.order = append(g.order, in.From)
	}
	g.fingerprints[in.From] = in.Fingerprint
	g.links[in.From] = append(g.links[in.From], link)
	g.positions[in.From] = append(g.positions[in.From], i)
	return nil
}

// checkWritable returns why the link cannot be written, as the notes library requires of a link.
func checkWritable(link domain.Link) error {
	if !domain.IsKnownRole(link.Role) {
		return fmt.Errorf("%q is not a role a link can have: parent, child, jump, ref or attachment", link.Role)
	}
	if !domain.IsWritableMarker(link.Marker) {
		return fmt.Errorf("%q is not a marker a link can have", link.Marker)
	}
	if link.Target.Value == "" {
		return errors.New("to is required")
	}
	return nil
}

func (t *Tools) addLinks(ctx context.Context, in LinkAddInput) (LinkAddOutput, error) {
	switch {
	case len(in.Links) == 0:
		return LinkAddOutput{}, errors.New("links is required")
	case len(in.Links) > maxBatch:
		return LinkAddOutput{}, fmt.Errorf("%d links given, at most %d are allowed", len(in.Links), maxBatch)
	}
	out := LinkAddOutput{Results: make([]LinkAddItem, len(in.Links))}
	groups := &linkGroups{
		order:        nil,
		links:        map[string][]domain.Link{},
		fingerprints: map[string]string{},
		positions:    map[string][]int{},
		conflicts:    map[string]error{},
	}
	for i, l := range in.Links {
		out.Results[i] = LinkAddItem{From: l.From, To: l.To, Fingerprint: "", Error: ""}
		if err := groups.add(i, l); err != nil {
			out.Results[i].Error = err.Error()
		}
	}
	for _, from := range groups.order {
		fingerprint, err := t.writeLinkGroup(ctx, groups, from)
		for _, i := range groups.positions[from] {
			if err != nil {
				out.Results[i].Error = err.Error()
				continue
			}
			out.Results[i].Fingerprint = fingerprint
		}
	}
	return out, nil
}

// writeLinkGroup writes the links of the call into the note, unless they carry two different
// fingerprints, and returns the fingerprint of the note written.
func (t *Tools) writeLinkGroup(ctx context.Context, groups *linkGroups, from string) (string, error) {
	if err := groups.conflicts[from]; err != nil {
		return "", err
	}
	return t.editNote(ctx, from, groups.fingerprints[from], addLinksTo(groups.links[from]))
}

// addLinksTo returns the change that writes the links into a document.
func addLinksTo(links []domain.Link) func(*markdown.Document) error {
	return func(doc *markdown.Document) error {
		for _, link := range links {
			if err := doc.AddLink(link); err != nil {
				return err
			}
		}
		return nil
	}
}

func (t *Tools) updateLink(ctx context.Context, in LinkUpdateInput) (LinkEditOutput, error) {
	to := domain.ParseAddress(in.To)
	if to.Value == "" {
		return LinkEditOutput{}, errors.New("to is required")
	}
	role := domain.LinkRole(in.Role)
	if role != "" && !domain.IsKnownRole(role) {
		return LinkEditOutput{}, fmt.Errorf("%q is not a role a link can have: parent, child, jump, ref or attachment", role)
	}
	change := domain.Link{Target: to, Role: role, Type: in.Type, Why: in.Note, Label: in.Label, Marker: domain.MarkerNone}
	fingerprint, err := t.editNote(ctx, in.From, in.Fingerprint, func(doc *markdown.Document) error {
		count, err := doc.UpdateLink(to, change)
		if err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("no link to %s is written here", in.To)
		}
		return nil
	})
	if err != nil {
		return LinkEditOutput{}, err
	}
	return LinkEditOutput{Path: in.From, Fingerprint: fingerprint}, nil
}

func (t *Tools) removeLinks(ctx context.Context, in LinkRemoveInput) (LinkEditOutput, error) {
	to := domain.ParseAddress(in.To)
	if to.Value == "" {
		return LinkEditOutput{}, errors.New("to is required")
	}
	role := domain.LinkRole(in.Role)
	fingerprint, err := t.editNote(ctx, in.From, in.Fingerprint, func(doc *markdown.Document) error {
		return removeLinksFrom(doc, in.From, to, role)
	})
	if err != nil {
		return LinkEditOutput{}, err
	}
	return LinkEditOutput{Path: in.From, Fingerprint: fingerprint}, nil
}

// removeLinksFrom removes the frontmatter links of the document to the target, of every type, and only
// those of the role when one is given.
func removeLinksFrom(doc *markdown.Document, notePath string, to domain.Address, role domain.LinkRole) error {
	current, err := getFrontmatterLinks(doc, notePath, to, role)
	if err != nil {
		return err
	}
	if len(current) == 0 {
		return fmt.Errorf("no link to %s is written here", to.GetWritten())
	}
	var types []string
	for _, l := range current {
		if !slices.Contains(types, l.Type) {
			types = append(types, l.Type)
		}
	}
	for _, linkType := range types {
		if _, err := doc.RemoveLink(domain.LinkKey{Target: to, Role: role, Type: linkType}); err != nil {
			return err
		}
	}
	return nil
}

// getFrontmatterLinks returns the links of the document's frontmatter to the target, only those of the
// role when one is given. Links in the prose are left out.
func getFrontmatterLinks(
	doc *markdown.Document, notePath string, to domain.Address, role domain.LinkRole,
) ([]domain.Link, error) {
	front, err := markdown.Open(doc.Bytes())
	if err != nil {
		return nil, err
	}
	if err := front.SetBody(""); err != nil {
		return nil, err
	}
	var stat domain.FileStat
	stat.Path = notePath
	var out []domain.Link
	for _, l := range markdown.Parse(stat, front.Bytes()).Links {
		if l.Target == to && (role == "" || l.Role == role) {
			out = append(out, l)
		}
	}
	return out, nil
}

// editNote makes one change to the note at path through the notes library, against the fingerprint
// note_read gave, and returns the fingerprint of the note written.
func (t *Tools) editNote(
	ctx context.Context, notePath, fingerprint string, change func(*markdown.Document) error,
) (string, error) {
	if strings.TrimSpace(notePath) == "" {
		return "", errors.New("from is required")
	}
	baseline, err := parseFingerprint(fingerprint)
	if err != nil {
		return "", err
	}
	files := &vaultFiles{vault: t.vault, releaseErr: nil}
	e := edit.Edit{
		Open: func(domain.Vault) (edit.Files, error) { return files, nil },
		Index: func(ctx context.Context, _ domain.Vault, paths []string) error {
			return t.vault.Scan(ctx, paths)
		},
		Clock:           clock.System{},
		Entropy:         rand.Reader,
		FileStat:        baseline,
		ShouldOverwrite: false,
		Bound:           0,
		Baseline:        nil,
	}
	v := domain.Vault{ID: domain.VaultID(tools.GetVault(ctx)), Name: "", Path: ""}
	stat, err := e.Apply(ctx, v, notePath, change)
	if err = errors.Join(err, files.releaseErr); err != nil {
		if errors.Is(err, host.ErrStale) || errors.Is(err, edit.ErrStale) {
			return "", fmt.Errorf("%s changed since note_read gave the fingerprint; read it again: %w", notePath, err)
		}
		return "", err
	}
	return formatFingerprint(stat), nil
}

// vaultFiles is the vault of a call as the notes library edits it.
type vaultFiles struct {
	vault host.Vault
	// releaseErr is the failure to release the vault's write lock, which edit.Files releases without
	// an error.
	releaseErr error
}

func (f *vaultFiles) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return f.vault.StatFile(ctx, path)
}

func (f *vaultFiles) Read(ctx context.Context, path string) ([]byte, error) {
	data, _, err := f.vault.Read(ctx, path)
	return data, err
}

func (f *vaultFiles) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	return f.vault.Write(ctx, path, content, stat)
}

func (f *vaultFiles) Lock(ctx context.Context) (func(), error) {
	release, err := f.vault.Lock(ctx)
	if err != nil {
		return nil, err
	}
	return func() { f.releaseErr = errors.Join(f.releaseErr, release()) }, nil
}
