package hosttest

import (
	"context"
	"errors"
	"fmt"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
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
		if err := checkWritable(one, additions, groups[one.From]); err != nil {
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

// checkWritable returns the reason the addition cannot be written: a link the notes library rejects, or a
// baseline other than the one of the links before it into the same note.
func checkWritable(one host.LinkAddition, additions []host.LinkAddition, grouped []int) error {
	switch {
	case !domain.IsKnownRole(one.Link.Role):
		return fmt.Errorf("%q is not a role a link can have", one.Link.Role)
	case !domain.IsWritableMarker(one.Link.Marker):
		return fmt.Errorf("%q is not a marker a link can have", one.Link.Marker)
	case one.Link.Target.Value == "":
		return errors.New("link target is empty")
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

// editLinks makes one change to the note at from in one read and one write, against the baseline
// when one is given, and returns the stat of the note written. The caller has the lock.
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
