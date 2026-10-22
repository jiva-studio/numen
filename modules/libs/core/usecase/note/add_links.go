package note

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// LinkAddition is one relationship to write, and the note it is written in.
// Fingerprint is that note as the caller read it.
type LinkAddition struct {
	From        string
	Fingerprint domain.Fingerprint
	Link        domain.Link

	// FingerprintError is why the caller's fingerprint could not be read. It is
	FingerprintError error
}

// LinkAdded is what happened to one addition. Error is nil when it was
// written, and Fingerprint is then the note as it now stands.
type LinkAdded struct {
	Fingerprint domain.Fingerprint
	Error       error
}

// AddLinks writes a batch of relationships, grouped by the note each is
// written in: the links of one note are one write against one fingerprint, and
// succeed or fail together.
type AddLinks struct {
	linking EditLinks
}

// NewAddLinks writes batches of relationships through the editor of a note's
// relationships.
func NewAddLinks(linking EditLinks) AddLinks {
	return AddLinks{linking: linking}
}

// Execute writes the additions and answers for each one, in the order given. An
// addition that cannot be written is rejected on its own and costs the others
// nothing.
func (u AddLinks) Execute(ctx context.Context, v domain.Vault, additions []LinkAddition) ([]LinkAdded, error) {
	results := make([]LinkAdded, len(additions))
	groups := newLinkGroups()
	for i, add := range additions {
		results[i].Error = groups.add(i, add)
	}
	for _, from := range groups.order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		group := groups.links[from]
		content, err := u.linking.Add(ctx, v, from, groups.fingerprints[from], group[0], group[1:]...)
		for _, i := range groups.positions[from] {
			results[i].Error = err
			if err == nil {
				results[i].Fingerprint = content
			}
		}
	}
	return results, nil
}

// linkGroups holds the links of one call grouped by the note they are written
// in, in order of first appearance.
type linkGroups struct {
	order        []string
	links        map[string][]domain.Link
	fingerprints map[string]domain.Fingerprint
	positions    map[string][]int
}

func newLinkGroups() *linkGroups {
	return &linkGroups{
		links:        map[string][]domain.Link{},
		fingerprints: map[string]domain.Fingerprint{},
		positions:    map[string][]int{},
	}
}

// add groups the i-th addition under its note and returns the reason it was
func (g *linkGroups) add(i int, add LinkAddition) error {
	if err := CheckWritable(add.Link); err != nil {
		return err
	}
	if add.FingerprintError != nil {
		return add.FingerprintError
	}
	if first, grouped := g.fingerprints[add.From]; grouped && first != add.Fingerprint {
		return fmt.Errorf("the links written into %s name two different fingerprints, and they are one write", add.From)
	}
	if _, grouped := g.links[add.From]; !grouped {
		g.order = append(g.order, add.From)
	}
	g.fingerprints[add.From] = add.Fingerprint
	g.links[add.From] = append(g.links[add.From], add.Link)
	g.positions[add.From] = append(g.positions[add.From], i)
	return nil
}
