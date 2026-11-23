package note

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListLinksOf reads the links of every path in one query and resolves them with one set of prepared statements.
func (q *QueryReads) ListLinksOf(ctx context.Context, vaultID domain.VaultID, from []string) (map[string][]domain.ResolvedLink, error) {
	if len(from) == 0 {
		return map[string][]domain.ResolvedLink{}, nil
	}
	if err := q.cache.ensureGraphLoaded(ctx, q.db, vaultID); err != nil {
		return nil, err
	}
	gen, out, missing := q.cache.splitLinksOf(vaultID, from)
	if len(missing) == 0 {
		return out, nil
	}
	fetched, err := runResolution(ctx, q.db, q.cache, vaultID, map[string][]domain.ResolvedLink{}, func(r *resolution) (map[string][]domain.ResolvedLink, error) {
		return r.getLinksOf(ctx, missing)
	})
	if err != nil {
		return nil, err
	}
	q.cache.setLinksOf(gen, vaultID, missing, fetched)
	maps.Copy(out, fetched)
	return out, nil
}

func (r *resolution) getLinksOf(ctx context.Context, from []string) (_ map[string][]domain.ResolvedLink, err error) {
	grouped, order, allLinks, err := r.readLinksAt(ctx, from)
	if err != nil {
		return nil, err
	}
	if err := r.preloadCandidates(ctx, allLinks); err != nil {
		return nil, err
	}
	return r.resolveLinksOf(ctx, grouped, order)
}

func (r *resolution) readLinksAt(ctx context.Context, from []string) (_ map[string][]domain.ResolvedLink, _ []string, _ []domain.ResolvedLink, err error) {
	wanted, err := json.Marshal(from)
	if err != nil {
		return nil, nil, nil, err
	}
	rows, err := r.statements.QueryContext(ctx, linksAtSQL, r.vault, string(wanted))
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	grouped := make(map[string][]domain.ResolvedLink, len(from))
	var order []string
	var allLinks []domain.ResolvedLink
	for rows.Next() {
		var path, role, marker string
		var l domain.ResolvedLink
		if err := rows.Scan(&path, &l.Target.Scheme, &l.Target.Value, &role,
			&l.Type, &l.Why, &l.Label, &marker); err != nil {
			return nil, nil, nil, err
		}
		l.From = path
		l.Role = domain.LinkRole(role)
		l.Marker = domain.LinkMarker(marker)
		if _, seen := grouped[path]; !seen {
			order = append(order, path)
		}
		grouped[path] = append(grouped[path], l)
		allLinks = append(allLinks, l)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	return grouped, order, allLinks, nil
}

func (r *resolution) resolveLinksOf(ctx context.Context, grouped map[string][]domain.ResolvedLink, order []string) (map[string][]domain.ResolvedLink, error) {
	out := make(map[string][]domain.ResolvedLink, len(grouped))
	for _, path := range order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		links := grouped[path]
		for i := range links {
			if err := r.resolve(ctx, path, &links[i]); err != nil {
				return nil, err
			}
		}
		if deduped := dedupe(links); len(deduped) > 0 {
			out[path] = deduped
		}
	}
	return out, nil
}

// ListBacklinksOf reads the backlinks of every path in one batch and resolves them with one set of prepared statements.
func (q *QueryReads) ListBacklinksOf(ctx context.Context, vaultID domain.VaultID, to []string) (map[string][]domain.ResolvedLink, error) {
	if len(to) == 0 {
		return map[string][]domain.ResolvedLink{}, nil
	}
	if err := q.cache.ensureGraphLoaded(ctx, q.db, vaultID); err != nil {
		return nil, err
	}
	gen, out, missing := q.cache.splitBacklinksOf(vaultID, to)
	if len(missing) == 0 {
		return out, nil
	}
	fetched, err := runResolution(ctx, q.db, q.cache, vaultID, map[string][]domain.ResolvedLink{}, func(r *resolution) (map[string][]domain.ResolvedLink, error) {
		return r.getBacklinksOf(ctx, missing)
	})
	if err != nil {
		return nil, err
	}
	q.cache.setBacklinksOf(gen, vaultID, missing, fetched)
	maps.Copy(out, fetched)
	return out, nil
}

func (r *resolution) getBacklinksOf(ctx context.Context, to []string) (_ map[string][]domain.ResolvedLink, err error) {
	wanted, identifiers, bases, err := r.readAddressingAt(ctx, to)
	if err != nil || len(wanted) == 0 {
		return map[string][]domain.ResolvedLink{}, err
	}
	candidates, err := r.getBacklinkCandidatesAt(ctx, identifiers, bases)
	if err != nil {
		return nil, err
	}
	if err := r.preloadCandidates(ctx, candidates); err != nil {
		return nil, err
	}

	out := make(map[string][]domain.ResolvedLink, len(wanted))
	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := r.resolve(ctx, c.From, &c); err != nil {
			return nil, err
		}
		if c.ToVault == r.vaultID && wanted[c.To] {
			out[c.To] = append(out[c.To], c)
		}
	}
	return out, nil
}

func (r *resolution) readAddressingAt(ctx context.Context, to []string) (wanted map[string]bool, identifiers, bases []string, err error) {
	pathsJSON, err := json.Marshal(to)
	if err != nil {
		return nil, nil, nil, err
	}
	rows, err := r.statements.QueryContext(ctx, addressingAtSQL, r.vault, string(pathsJSON))
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	wanted = make(map[string]bool, len(to))
	seenID := map[string]bool{}
	seenBase := map[string]bool{}
	for rows.Next() {
		var path, id, base string
		if err := rows.Scan(&path, &id, &base); err != nil {
			return nil, nil, nil, err
		}
		wanted[path] = true
		if id != "" && !seenID[id] {
			seenID[id] = true
			identifiers = append(identifiers, id)
		}
		if !seenBase[base] {
			seenBase[base] = true
			bases = append(bases, base)
		}
	}
	return wanted, identifiers, bases, rows.Err()
}

func (r *resolution) getBacklinkCandidatesAt(ctx context.Context, identifiers, bases []string) (_ []domain.ResolvedLink, err error) {
	idsJSON, err := json.Marshal(identifiers)
	if err != nil {
		return nil, err
	}
	basesJSON, err := json.Marshal(bases)
	if err != nil {
		return nil, err
	}
	rows, err := r.statements.QueryContext(ctx, backlinkCandidatesAtSQL,
		string(idsJSON), r.vault, string(basesJSON), r.vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var candidates []domain.ResolvedLink
	for rows.Next() {
		var l domain.ResolvedLink
		var role, marker string
		var position int
		if err := rows.Scan(&l.From, &l.Target.Scheme, &l.Target.Value, &role,
			&l.Type, &l.Why, &l.Label, &marker, &position); err != nil {
			return nil, err
		}
		l.Role = domain.LinkRole(role)
		l.Marker = domain.LinkMarker(marker)
		candidates = append(candidates, l)
	}
	return candidates, rows.Err()
}
