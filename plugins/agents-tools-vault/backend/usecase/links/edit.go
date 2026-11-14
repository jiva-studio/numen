package links

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/linkrule"
)

// LinkChange is a link to write into the frontmatter of a note, or the change of the links a note has
// to the target at Address.
type LinkChange struct {
	From        string
	Fingerprint string
	Address     string
	Role        string
	Type        string
	Label       string
	Note        string
}

// AddResult is the outcome of writing one link: the fingerprint of the note after the write, or the
// error that kept the link from being written.
type AddResult struct {
	Fingerprint string
	Err         error
}

// NoteEdit is the note a link edit wrote and its fingerprint after the write.
type NoteEdit struct {
	Path        string
	Fingerprint string
}

// Add writes the links into the frontmatter of their notes. A link that is invalid or stale gets an
// error in its result and does not stop the others.
func (u *UseCase) Add(ctx context.Context, links []LinkChange) ([]AddResult, error) {
	if len(links) == 0 {
		return nil, errors.New("links is required")
	}
	results := make([]AddResult, len(links))
	additions := make([]host.LinkAddition, 0, len(links))
	positions := make([]int, 0, len(links))
	for i, link := range links {
		addition, err := newAddition(link)
		if err != nil {
			results[i].Err = err
			continue
		}
		additions = append(additions, addition)
		positions = append(positions, i)
	}
	if len(additions) == 0 {
		return results, nil
	}
	added, err := u.editor.AddLinks(ctx, additions)
	if err != nil {
		return nil, fmt.Errorf("write the links: %w", err)
	}
	if len(added) != len(additions) {
		return nil, fmt.Errorf("the host returned %d of %d links", len(added), len(additions))
	}
	for j, one := range added {
		i := positions[j]
		if one.Err != nil {
			results[i].Err = describeWriteError(links[i].From, one.Err)
			continue
		}
		results[i].Fingerprint = fingerprint.Format(one.Stat)
	}
	return results, nil
}

// Update changes the role, type, label or note of the links a note has to a target.
func (u *UseCase) Update(ctx context.Context, in LinkChange) (NoteEdit, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return NoteEdit{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := linkrule.CheckRole(role); err != nil {
		return NoteEdit{}, err
	}
	baseline, err := newBaseline(in.From, in.Fingerprint)
	if err != nil {
		return NoteEdit{}, err
	}
	change := domain.Link{Role: role, Type: in.Type, Why: in.Note, Label: in.Label, Marker: domain.MarkerNone}
	stat, err := u.editor.UpdateLinks(ctx, in.From, target, change, baseline)
	if err != nil {
		return NoteEdit{}, describeWriteError(in.From, err)
	}
	return NoteEdit{Path: in.From, Fingerprint: fingerprint.Format(stat)}, nil
}

// Remove removes the links a note has to a target, or only the link of one role. Remove reads From,
// Fingerprint, Address and Role of the change.
func (u *UseCase) Remove(ctx context.Context, in LinkChange) (NoteEdit, error) {
	target := domain.ParseAddress(in.Address)
	if target.Value == "" {
		return NoteEdit{}, errors.New("address is required")
	}
	role := domain.LinkRole(in.Role)
	if err := linkrule.CheckRole(role); err != nil {
		return NoteEdit{}, err
	}
	baseline, err := newBaseline(in.From, in.Fingerprint)
	if err != nil {
		return NoteEdit{}, err
	}
	stat, err := u.editor.RemoveLinks(ctx, in.From, target, role, baseline)
	if err != nil {
		return NoteEdit{}, describeWriteError(in.From, err)
	}
	return NoteEdit{Path: in.From, Fingerprint: fingerprint.Format(stat)}, nil
}

func newAddition(in LinkChange) (host.LinkAddition, error) {
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
