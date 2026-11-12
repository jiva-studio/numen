package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// linkCache reads the links of a note once per query.
type linkCache struct {
	queries noteport.LinkQueries
	vault   domain.Vault
	byPath  map[string][]domain.ResolvedLink
}

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

func getUpperName(key pairKey, r record) string {
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

func getParent(key pairKey, records []record) (parent, child string, isHierarchical bool) {
	earlier, later := key[0], key[1]
	hasEarlierParent, hasLaterParent := false, false
	for _, r := range records {
		upper := getUpperName(key, r)
		hasEarlierParent = hasEarlierParent || upper == earlier
		hasLaterParent = hasLaterParent || upper == later
	}
	if hasEarlierParent && !hasLaterParent {
		return earlier, later, true
	}
	return later, earlier, hasLaterParent
}

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
