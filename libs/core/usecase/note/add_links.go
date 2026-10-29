package note

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// LinkAddition is one relationship to write, and the note it is written in.
// FileStat is that note as the caller read it.
type LinkAddition struct {
	From     string
	FileStat domain.FileStat
	Link     domain.Link

	// FileStatError is why the caller's file stat could not be read. It is
	FileStatError error
}

// LinkAdded is the outcome of one addition. Error is nil when the link was
// written, and FileStat is then the note's current state.
type LinkAdded struct {
	FileStat domain.FileStat
	Error    error
}

// AddLinks writes a batch of relationships, grouped by the note each is
// written in: the links of one note are one write against one file stat, and
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
		content, err := u.linking.Add(ctx, v, from, groups.stats[from], group[0], group[1:]...)
		for _, i := range groups.positions[from] {
			results[i].Error = err
			if err == nil {
				results[i].FileStat = content
			}
		}
	}
	return results, nil
}

// linkGroups holds the links of one call grouped by the note they are written
// in, in order of first appearance.
type linkGroups struct {
	order     []string
	links     map[string][]domain.Link
	stats     map[string]domain.FileStat
	positions map[string][]int
}

func newLinkGroups() *linkGroups {
	return &linkGroups{
		links:     map[string][]domain.Link{},
		stats:     map[string]domain.FileStat{},
		positions: map[string][]int{},
	}
}

// add groups the i-th addition under its note and returns the reason it was
func (g *linkGroups) add(i int, add LinkAddition) error {
	if err := CheckWritable(add.Link); err != nil {
		return err
	}
	if add.FileStatError != nil {
		return add.FileStatError
	}
	if first, grouped := g.stats[add.From]; grouped && first != add.FileStat {
		return fmt.Errorf("the links written into %s name two different fingerprints, and they are one write", add.From)
	}
	if _, grouped := g.links[add.From]; !grouped {
		g.order = append(g.order, add.From)
	}
	g.stats[add.From] = add.FileStat
	g.links[add.From] = append(g.links[add.From], add.Link)
	g.positions[add.From] = append(g.positions[add.From], i)
	return nil
}
