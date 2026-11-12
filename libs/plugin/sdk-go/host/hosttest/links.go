package hosttest

import (
	"context"
	"errors"
	"fmt"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/note/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// AddLinks writes the links into the frontmatter of their notes, as the host does: the links of one
// note are one write against its baseline, and a link that cannot be written fails only itself.
func (v *Vault) AddLinks(ctx context.Context, additions []host.LinkAddition) ([]host.LinkAdded, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	out := make([]host.LinkAdded, len(additions))
	var order []string
	groups := map[string][]int{}
	for i, one := range additions {
		if err := checkLinkWritable(one, additions, groups[one.From]); err != nil {
			out[i].Err = err
			continue
		}
		if _, isGrouped := groups[one.From]; !isGrouped {
			order = append(order, one.From)
		}
		groups[one.From] = append(groups[one.From], i)
	}
	for _, from := range order {
		positions := groups[from]
		stat, err := v.editLinks(ctx, from, additions[positions[0]].Baseline, func(doc *notesmarkdown.Document) error {
			for _, i := range positions {
				if err := doc.AddLink(additions[i].Link); err != nil {
					return err
				}
			}
			return nil
		})
		for _, i := range positions {
			out[i] = host.LinkAdded{Stat: stat, Err: err}
		}
	}
	return out, nil
}

func checkLinkWritable(one host.LinkAddition, additions []host.LinkAddition, grouped []int) error {
	switch {
	case !domain.IsKnownRole(one.Link.Role):
		return fmt.Errorf("%q is not a role a link can have", one.Link.Role)
	case !domain.IsWritableMarker(one.Link.Marker):
		return fmt.Errorf("%q is not a marker a link can have", one.Link.Marker)
	case one.Link.Target.Value == "":
		return errors.New("a link needs somewhere to go")
	case len(grouped) > 0 && additions[grouped[0]].Baseline != one.Baseline:
		return fmt.Errorf("links for %s have different fingerprints", one.From)
	}
	return nil
}

// UpdateLinks changes the role, type, label and note of the links the note has to the target, and
// keeps every field left empty.
func (v *Vault) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	if change.Role != "" && !domain.IsKnownRole(change.Role) {
		return domain.FileStat{}, fmt.Errorf("%q is not a role a link can have", change.Role)
	}
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return v.editLinks(ctx, from, baseline, func(doc *notesmarkdown.Document) error {
		changed, err := doc.UpdateLink(to, change)
		if err != nil {
			return err
		}
		if changed == 0 {
			return fmt.Errorf("no link to %s is written here", to.GetWritten())
		}
		return nil
	})
}

// RemoveLinks removes the frontmatter links the note has to the target, of every type, and only those
// of the role when one is given.
func (v *Vault) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return v.editLinks(ctx, from, baseline, func(doc *notesmarkdown.Document) error {
		removed, err := doc.RemoveTargetLinks(to, role)
		if err != nil {
			return err
		}
		if removed == 0 {
			return fmt.Errorf("no link to %s is written here", to.GetWritten())
		}
		return nil
	})
}

func (v *Vault) editLinks(
	ctx context.Context, from string, baseline domain.FileStat, change func(*notesmarkdown.Document) error,
) (domain.FileStat, error) {
	if err := v.checkNote(ctx, from); err != nil {
		return domain.FileStat{}, err
	}
	if err := v.unwritable[from]; err != nil {
		return domain.FileStat{}, err
	}
	now, err := v.files.stat(from)
	if err != nil {
		return domain.FileStat{}, err
	}
	if !baseline.IsZero() && !now.IsUnchanged(baseline) {
		return domain.FileStat{}, notesdomain.NewDetailError(fmt.Sprintf("%s changed since it was read", from), host.ErrStale)
	}
	raw, err := v.files.read(from)
	if err != nil {
		return domain.FileStat{}, err
	}
	doc, err := notesmarkdown.Open(raw)
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := change(doc); err != nil {
		return domain.FileStat{}, err
	}
	stat, err := v.files.write(from, doc.Bytes())
	if err != nil {
		return domain.FileStat{}, err
	}
	v.rescan(from)
	return stat, nil
}

func (v *Vault) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	n, ok := notes[from]
	if !ok {
		return nil, nil
	}
	out := make([]domain.ResolvedLink, 0, len(n.Links))
	for _, l := range n.Links {
		r := domain.ResolvedLink{Link: l, From: from}
		v.resolve(notes, from, &r)
		out = append(out, r)
	}
	return dedupe(out), nil
}

func (v *Vault) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	return v.getBacklinks(v.parsed(), to), nil
}

func (v *Vault) getBacklinks(notes map[string]domain.Note, to string) []domain.ResolvedLink {
	target, ok := notes[to]
	if !ok {
		return nil
	}
	base := notesdomain.FoldName(domain.Basename(to))
	var out []domain.ResolvedLink
	for _, from := range sortedPaths(notes) {
		for _, l := range notes[from].Links {
			mayReach := (l.Target.Scheme == domain.SchemeNote && target.ID != "" && l.Target.Value == target.ID) ||
				(l.Target.Scheme == domain.SchemeName && notesdomain.FoldName(notesdomain.LinkName(l.Target.Value)) == base)
			if !mayReach {
				continue
			}
			r := domain.ResolvedLink{Link: l, From: from}
			v.resolve(notes, from, &r)
			if r.To == to && r.ToVault == v.vault.ID {
				out = append(out, r)
			}
		}
	}
	return out
}

// ResolveLinks answers each written link by the links the note holds whose target is written so.
func (v *Vault) ResolveLinks(ctx context.Context, from string, links []string) (map[string]domain.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]domain.ResolvedLink{}
	for _, raw := range links {
		r := domain.ResolvedLink{Link: domain.Link{Target: notesdomain.ParseAddress(raw)}, From: from}
		v.resolve(notes, from, &r)
		if r.To != "" {
			out[raw] = r
		}
	}
	return out, nil
}
