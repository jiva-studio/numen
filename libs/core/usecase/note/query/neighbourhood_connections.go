package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// linkCache reads the links of a note once per query.
type linkCache struct {
	queries noteport.LinkQueries
	vault   domain.Vault
	byPath  map[string][]domain.ResolvedLink
}

// get returns the links of the note at path.
func (c *linkCache) get(ctx context.Context, path string) ([]domain.ResolvedLink, error) {
	if links, ok := c.byPath[path]; ok {
		return links, nil
	}
	links, err := c.queries.GetLinks(ctx, c.vault.ID, path)
	if err != nil {
		return nil, err
	}
	if c.byPath == nil {
		c.byPath = map[string][]domain.ResolvedLink{}
	}
	c.byPath[path] = links
	return links, nil
}

// isNavigable reports whether a link is a record of a connection inside the vault.
func (c *linkCache) isNavigable(l domain.ResolvedLink) bool {
	return l.ToVault == c.vault.ID && domain.IsConnectionRole(l.Role)
}

// record is a link as the note at one end of a pair wrote it.
type record struct {
	author string
	link   domain.ResolvedLink
}

// pairKey is a pair of notes, the earlier one in the drawing order first.
type pairKey [2]string

// getConnections returns one connection for every pair of drawn notes that a
// navigable record joins, in the order the pairs are first found: the records
// of each note in drawing order, then in the order they are written. Every line
// the plex draws comes from one of these, whether it ends at the note in focus
// or joins two other notes.
func getConnections(ctx context.Context, cache *linkCache, paths []string) ([]domain.Connection, error) {
	pairs, records, err := collectRecords(ctx, cache, paths)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Connection, 0, len(pairs))
	for _, key := range pairs {
		out = append(out, createConnection(key, records[key]))
	}
	return out, nil
}

// pairRecords groups the records between drawn notes by pair.
type pairRecords struct {
	position map[string]int
	pairs    []pairKey
	byPair   map[pairKey][]record
}

// add files a record under its pair; a record to a note that is not drawn, or to
// its own author, joins no pair.
func (p *pairRecords) add(author string, l domain.ResolvedLink) {
	target, isDrawn := p.position[l.To]
	if !isDrawn || l.To == author {
		return
	}
	key := pairKey{author, l.To}
	if target < p.position[author] {
		key = pairKey{l.To, author}
	}
	if _, exists := p.byPair[key]; !exists {
		p.pairs = append(p.pairs, key)
	}
	p.byPair[key] = append(p.byPair[key], record{author: author, link: l})
}

// collectRecords groups the navigable records between the given notes by pair.
func collectRecords(
	ctx context.Context, cache *linkCache, paths []string,
) ([]pairKey, map[pairKey][]record, error) {
	found := &pairRecords{position: make(map[string]int, len(paths)), byPair: map[pairKey][]record{}}
	for i, path := range paths {
		found.position[path] = i
	}
	for _, author := range paths {
		links, err := cache.get(ctx, author)
		if err != nil {
			return nil, nil, err
		}
		for _, l := range links {
			if cache.isNavigable(l) {
				found.add(author, l)
			}
		}
	}
	return found.pairs, found.byPair, nil
}

// getUpper returns the note a hierarchical record makes the parent of the pair,
// and an empty string for a jump.
func getUpper(key pairKey, r record) string {
	if r.link.Role == domain.RoleChild {
		return r.author
	}
	if r.link.Role != domain.RoleParent {
		return ""
	}
	if r.author == key[0] {
		return key[1]
	}
	return key[0]
}

// getParent returns which note of a pair is the parent and which the child.
// The later drawn note is the parent when each is the other's parent, so a
// neighbour outranks the focus as a parent. Notes joined by jumps only keep
// the same order. The last result is true when the pair is hierarchical.
func getParent(key pairKey, records []record) (parent, child string, isHierarchical bool) {
	earlier, later := key[0], key[1]
	hasEarlierParent, hasLaterParent := false, false
	for _, r := range records {
		upper := getUpper(key, r)
		hasEarlierParent = hasEarlierParent || upper == earlier
		hasLaterParent = hasLaterParent || upper == later
	}
	if hasEarlierParent && !hasLaterParent {
		return earlier, later, true
	}
	return later, earlier, hasLaterParent
}

// createConnection joins the records of one pair. A hierarchical connection
// runs from the parent to the child; a connection of jumps runs from the later
// drawn note to the earlier. A record of another kind or the other direction is
// left out, and of two equal records the first written stays.
func createConnection(key pairKey, records []record) domain.Connection {
	parent, child, isHierarchical := getParent(key, records)
	roleOfChild, roleOfParent := domain.RoleJump, domain.RoleJump
	if isHierarchical {
		roleOfChild, roleOfParent = domain.RoleChild, domain.RoleParent
	}

	connection := domain.Connection{A: parent, B: child}
	for _, r := range records {
		side := &domain.Side{
			Role: r.link.Role, Type: r.link.Type, Why: r.link.Why, Label: r.link.Label, Marker: r.link.Marker,
		}
		switch {
		case r.author == parent && r.link.Role == roleOfChild && connection.AB == nil:
			connection.AB = side
		case r.author == child && r.link.Role == roleOfParent && connection.BA == nil:
			connection.BA = side
		}
	}
	return connection
}
