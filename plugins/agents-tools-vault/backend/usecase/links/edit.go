package links

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/linkrule"
)

// Add writes the links into the frontmatter of their notes. A link that is invalid or stale gets an
// error in its result and does not stop the others.
func (u *UseCase) Add(ctx context.Context, in api.LinkAddInput) (api.LinkAddOutput, error) {
	if len(in.Links) == 0 {
		return api.LinkAddOutput{}, errors.New("links is required")
	}
	out := api.LinkAddOutput{Results: make([]api.LinkAddItem, len(in.Links))}
	additions := make([]host.LinkAddition, 0, len(in.Links))
	positions := make([]int, 0, len(in.Links))
	for i, link := range in.Links {
		out.Results[i] = api.LinkAddItem{From: link.From, Address: link.Address, Fingerprint: "", Error: ""}
		addition, err := newAddition(link)
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
	added, err := u.editor.AddLinks(ctx, additions)
	if err != nil {
		return api.LinkAddOutput{}, fmt.Errorf("write the links: %w", err)
	}
	if len(added) != len(additions) {
		return api.LinkAddOutput{}, fmt.Errorf("the host returned %d of %d links", len(added), len(additions))
	}
	for j, one := range added {
		i := positions[j]
		if one.Err != nil {
			out.Results[i].Error = describeWriteError(in.Links[i].From, one.Err).Error()
			continue
		}
		out.Results[i].Fingerprint = fingerprint.Format(one.Stat)
	}
	return out, nil
}

// Update changes the role, type, label or note of the links a note has to a target.
func (u *UseCase) Update(ctx context.Context, in api.LinkUpdateInput) (api.LinkEditOutput, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return api.LinkEditOutput{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := linkrule.CheckRole(role); err != nil {
		return api.LinkEditOutput{}, err
	}
	baseline, err := newBaseline(in.From, in.Fingerprint)
	if err != nil {
		return api.LinkEditOutput{}, err
	}
	change := domain.Link{Role: role, Type: in.Type, Why: in.Note, Label: in.Label, Marker: domain.MarkerNone}
	stat, err := u.editor.UpdateLinks(ctx, in.From, target, change, baseline)
	if err != nil {
		return api.LinkEditOutput{}, describeWriteError(in.From, err)
	}
	return api.LinkEditOutput{Path: in.From, Fingerprint: fingerprint.Format(stat)}, nil
}

// Remove removes the links a note has to a target, or only the link of one role.
func (u *UseCase) Remove(ctx context.Context, in api.LinkRemoveInput) (api.LinkEditOutput, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return api.LinkEditOutput{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := linkrule.CheckRole(role); err != nil {
		return api.LinkEditOutput{}, err
	}
	baseline, err := newBaseline(in.From, in.Fingerprint)
	if err != nil {
		return api.LinkEditOutput{}, err
	}
	stat, err := u.editor.RemoveLinks(ctx, in.From, target, role, baseline)
	if err != nil {
		return api.LinkEditOutput{}, describeWriteError(in.From, err)
	}
	return api.LinkEditOutput{Path: in.From, Fingerprint: fingerprint.Format(stat)}, nil
}

func newAddition(in api.LinkAddition) (host.LinkAddition, error) {
	link := domain.Link{
		Target: domain.ParseAddress(in.Address),
		Role:   domain.LinkRole(in.Role),
		Type:   in.Type,
		Why:    in.Note,
		Label:  in.Label,
		Marker: domain.MarkerNone,
	}
	if err := linkrule.CheckWritable(link); err != nil {
		return host.LinkAddition{}, err
	}
	baseline, err := fingerprint.Parse(in.Fingerprint)
	if err != nil {
		return host.LinkAddition{}, err
	}
	baseline.Path = in.From
	return host.LinkAddition{From: in.From, Baseline: baseline, Link: link}, nil
}

func newBaseline(notePath, token string) (domain.FileStat, error) {
	if strings.TrimSpace(notePath) == "" {
		return domain.FileStat{}, errors.New("from is required")
	}
	baseline, err := fingerprint.Parse(token)
	if err != nil {
		return domain.FileStat{}, err
	}
	baseline.Path = notePath
	return baseline, nil
}

func describeWriteError(notePath string, err error) error {
	if errors.Is(err, host.ErrStale) {
		return fmt.Errorf("%s changed since note_read returned the fingerprint; read it again: %w", notePath, err)
	}
	return err
}
