package note

import (
	"context"
	"database/sql"
	"errors"
	"maps"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type rawVaultLink struct {
	link   domain.ResolvedLink
	folded string
}

type vaultGraph struct {
	notes          map[string]cachedNoteRef
	types          map[string]cachedNoteType
	rawFrontmatter map[string]sql.NullString
	linksOf        map[string][]domain.ResolvedLink
	backlinksOf    map[string][]domain.ResolvedLink
	named          map[string][]string
}

type vaultLinkResolver struct {
	cachedResolution
	idByPath    map[string]string
	backlinksOf map[string][]domain.ResolvedLink
}

func getGraphVault(
	ctx context.Context, cache *Cache, db *sql.DB, gen uint64, vaultID domain.VaultID,
) (int64, bool, error) {
	if vault, ok := cache.getVaultRow(vaultID); ok {
		return vault, true, nil
	}
	vault, err := getVaultRow(ctx, db, vaultID)
	if errors.Is(err, errNoVault) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	cache.setVaultRow(gen, vaultID, vault)
	return vault, true, nil
}

func loadVaultGraph(ctx context.Context, db *sql.DB, vault int64, vaultID domain.VaultID) (vaultGraph, error) {
	found, err := loadVaultNotes(ctx, db, vault)
	if err != nil {
		return vaultGraph{}, err
	}
	links, err := loadVaultLinks(ctx, db, vault)
	if err != nil {
		return vaultGraph{}, err
	}
	identified, err := loadIdentifiedNotes(ctx, db, links.noteIDs)
	if err != nil {
		return vaultGraph{}, err
	}
	resolver := vaultLinkResolver{
		cachedResolution: cachedResolution{vaultID: vaultID, byPath: found.byPath, byFolded: found.byFolded, named: make(map[string][]string, len(found.byFolded)), identified: identified},
		idByPath:         found.idByPath,
		backlinksOf:      make(map[string][]domain.ResolvedLink, len(found.byPath)),
	}
	linksOf, err := resolver.resolve(ctx, links.order, links.grouped)
	if err != nil {
		return vaultGraph{}, err
	}
	return vaultGraph{notes: found.notes, types: found.types, rawFrontmatter: found.rawFrontmatter, linksOf: linksOf, backlinksOf: resolver.backlinksOf, named: resolver.named}, nil
}

func (r *vaultLinkResolver) resolve(
	ctx context.Context, order []string, grouped map[string][]rawVaultLink,
) (map[string][]domain.ResolvedLink, error) {
	linksOf := make(map[string][]domain.ResolvedLink, len(r.byPath))
	for _, from := range order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		links := r.resolveAt(from, grouped[from])
		if len(links) > 0 {
			linksOf[from] = links
		}
	}
	return linksOf, nil
}

func (r *vaultLinkResolver) resolveAt(from string, raws []rawVaultLink) []domain.ResolvedLink {
	links := make([]domain.ResolvedLink, len(raws))
	for i, raw := range raws {
		link := raw.link
		r.resolveLink(from, &link)
		links[i] = link
		if link.ToVault == r.vaultID && link.To != "" && isBacklinkMatch(link, raw.folded, r.idByPath[link.To], r.byPath[link.To]) {
			r.backlinksOf[link.To] = append(r.backlinksOf[link.To], link)
		}
	}
	return dedupe(links)
}

func setVaultGraph(cache *Cache, gen uint64, vaultID domain.VaultID, vault int64, graph vaultGraph) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.generation != gen {
		return
	}
	v := cache.getOrCreateVault(vaultID)
	v.row, v.hasRow, v.isGraphLoaded = vault, true, true
	v.linksOf, v.backlinksOf = graph.linksOf, graph.backlinksOf
	maps.Copy(v.named, graph.named)
	v.notes, v.types, v.rawFrontmatter = graph.notes, graph.types, graph.rawFrontmatter
}
