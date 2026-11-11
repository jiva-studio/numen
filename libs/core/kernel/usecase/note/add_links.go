package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// LinkAddition is one relationship to write, and the note it is written in.
// FileStat is that note as the caller read it.
type LinkAddition struct {
	From     string
	FileStat domain.FileStat
	Link     domain.Link

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
	linking linkAdding
}

// linkAdding is the write of relationships into one note that AddLinks repeats for each note.
type linkAdding interface {
	Add(
		ctx context.Context, v domain.Vault, from string, stat domain.FileStat,
		add domain.Link, more ...domain.Link,
	) (domain.FileStat, error)
}

// NewAddLinks writes batches of relationships through the editor of a note's
// relationships.
func NewAddLinks(linking linkAdding) AddLinks {
	return AddLinks{linking: linking}
}

// Execute writes the additions and answers for each one, in the order given. An
// addition that cannot be written is rejected on its own and costs the others
// nothing. When ctx is canceled, the links of every note not yet written get
// plugin.ErrNotAttempted, and the results are returned with a nil error.
func (u AddLinks) Execute(ctx context.Context, v domain.Vault, additions []LinkAddition) ([]LinkAdded, error) {
	results := make([]LinkAdded, len(additions))
	groups := newLinkGroups()
	for i, add := range additions {
		results[i].Error = groups.add(i, add)
	}
	for _, from := range groups.order {
		var content domain.FileStat
		err := ctx.Err()
		if err == nil {
			group := groups.links[from]
			content, err = u.linking.Add(ctx, v, from, groups.stats[from], group[0], group[1:]...)
		}
		if err != nil && ctx.Err() != nil && !errors.Is(err, edit.ErrIndexOutdated) {
			err = fmt.Errorf("the call was canceled before the links were added to %s: %w", from, plugin.ErrNotAttempted)
		}
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
