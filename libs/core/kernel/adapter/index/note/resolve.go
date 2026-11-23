package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func (q *Queries) ListLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, q.cache, vaultID, nil, func(r *resolution) ([]domain.ResolvedLink, error) {
		return r.getLinks(ctx, from)
	})
}

func (r *resolution) getLinks(ctx context.Context, from string) (_ []domain.ResolvedLink, err error) {
	var row int64
	err = r.statements.QueryRowContext(ctx, identifySQL, r.vault, from).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.statements.QueryContext(ctx, linksOfSQL, row)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.ResolvedLink
	for rows.Next() {
		var l domain.ResolvedLink
		var role, marker string
		if err := rows.Scan(&l.Target.Scheme, &l.Target.Value, &role,
			&l.Type, &l.Why, &l.Label, &marker); err != nil {
			return nil, err
		}
		l.Role = domain.LinkRole(role)
		l.Marker = domain.LinkMarker(marker)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.prefetchCandidates(ctx, out); err != nil {
		return nil, err
	}

	for i := range out {
		out[i].From = from
		if err := r.resolve(ctx, from, &out[i]); err != nil {
			return nil, err
		}
	}
	return dedupe(out), nil
}

func dedupe(links []domain.ResolvedLink) []domain.ResolvedLink {
	observed := map[string]int{}
	var out []domain.ResolvedLink
	for _, l := range links {
		key := l.To
		if key == "" {
			// Unresolved links are equal only when written identically; their target is
			// unknown here.
			key = "\x00" + l.Target.String()
		}
		pos, recognized := observed[key]
		if !recognized {
			observed[key] = len(out)
			out = append(out, l)
			continue
		}
		if out[pos].Role == domain.RoleRef && l.Role != domain.RoleRef {
			// A mention in prose gives way to the record that carries a role.
			out[pos] = l
		}
	}
	return out
}

// ResolveLinks returns the note each address written in one note resolves to,
// keyed by the written text. Resolution priority is the one every name uses, so
// a wikilink that was not recorded as a link resolves as a link does.
func (q *Queries) ResolveLinks(
	ctx context.Context, vaultID domain.VaultID, from string, content []string,
) (map[string]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, q.cache, vaultID, nil, func(r *resolution) (map[string]domain.ResolvedLink, error) {
		return r.resolveLinks(ctx, from, content)
	})
}

func (r *resolution) resolveLinks(ctx context.Context, from string, content []string) (map[string]domain.ResolvedLink, error) {
	request := make(map[string]bool, len(content))
	out := make(map[string]domain.ResolvedLink, len(content))
	for _, raw := range content {
		if request[raw] {
			continue
		}
		request[raw] = true
		one := domain.ResolvedLink{Link: domain.Link{Target: domain.ParseAddress(raw)}, From: from}
		if err := r.resolve(ctx, from, &one); err != nil {
			return nil, err
		}
		if one.To != "" {
			out[raw] = one
		}
	}
	return out, nil
}

func (r *resolution) resolve(ctx context.Context, from string, l *domain.ResolvedLink) error {
	switch l.Target.Scheme {
	case domain.SchemeNote:
		if found, ok := r.identified[l.Target.Value]; ok {
			l.ToVault, l.To = found.vault, found.path
			return nil
		}
		// An identifier names one note globally, so this lookup is not scoped to a
		// vault. It is the one deliberate exception to the rule that a query without a
		// vault must not read another vault's notes.
		err := r.statements.QueryRowContext(ctx, noteByIdentifierSQL, l.Target.Value).
			Scan(&l.ToVault, &l.To)
		if errors.Is(err, sql.ErrNoRows) {
			// Neither dangling nor resolved: no connected vault holds this note. The
			// index cannot distinguish a deleted note from one in a vault that is not
			// connected, and reporting it as broken would be wrong on a machine where both
			// vaults are open.
			r.identified[l.Target.Value] = identifiedNote{}
			return nil
		}
		if err == nil {
			r.identified[l.Target.Value] = identifiedNote{vault: l.ToVault, path: l.To}
		}
		return err
	case domain.SchemeName:
		candidates, err := r.getCandidates(ctx, l.Target.Value)
		if err != nil {
			return err
		}
		l.To, l.IsAmbiguous = domain.ChooseMatch(from, l.Target.Value, candidates)
		if l.To != "" {
			// A name means something only inside the vault it was written in.
			l.ToVault = r.vaultID
		}
		return nil
	default:
		// An asset or a URL: not a note, so no note resolves it.
		return nil
	}
}

func (r *resolution) getCandidates(ctx context.Context, name string) ([]string, error) {
	if found, ok := r.named[name]; ok {
		return found, nil
	}
	r.cache.copyNamedInto(r.vaultID, []string{name}, r.named)
	if found, ok := r.named[name]; ok {
		return found, nil
	}
	found, err := r.queryCandidates(ctx, name)
	if err != nil {
		return nil, err
	}
	r.named[name] = found
	r.cache.setNamed(r.generation, r.vaultID, []string{name}, r.named)
	return found, nil
}

func (r *resolution) queryCandidates(ctx context.Context, name string) (_ []string, err error) {
	// Three shapes of the same question, so one query answers all of them: the
	// name as a path, as a path with an extension added, and as a filename.
	withExt := name + domain.NoteExtension
	base := domain.NormalizeName(domain.LinkName(name))

	rows, err := r.statements.QueryContext(ctx, candidatesSQL,
		r.vault, name, r.vault, withExt, r.vault, base)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *resolution) prefetchCandidates(ctx context.Context, links []domain.ResolvedLink) (err error) {
	if err := r.prefetchIdentifiers(ctx, links); err != nil {
		return err
	}
	requested := getRequestedNames(links, r.named)
	r.cache.copyNamedInto(r.vaultID, requested, r.named)
	missing := getMissingNames(requested, r.named)
	if len(missing) == 0 {
		return nil
	}
	if len(missing) == 1 {
		_, err := r.getCandidates(ctx, missing[0])
		return err
	}
	return r.readCandidateNames(ctx, missing)
}

func getRequestedNames(links []domain.ResolvedLink, named map[string][]string) []string {
	var requested []string
	seen := map[string]bool{}
	for _, l := range links {
		if l.Target.Scheme != domain.SchemeName {
			continue
		}
		name := l.Target.Value
		if _, ok := named[name]; !ok && !seen[name] {
			seen[name] = true
			requested = append(requested, name)
		}
	}
	return requested
}

func getMissingNames(requested []string, named map[string][]string) []string {
	var missing []string
	for _, name := range requested {
		if _, ok := named[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func (r *resolution) readCandidateNames(ctx context.Context, missing []string) (err error) {
	candidates := createCandidateNames(missing, r.named)
	pathsJSON, err := json.Marshal(candidates.paths)
	if err != nil {
		return err
	}
	basesJSON, err := json.Marshal(candidates.bases)
	if err != nil {
		return err
	}

	rows, err := r.statements.QueryContext(ctx, candidatesAtSQL,
		r.vault, string(pathsJSON), r.vault, string(basesJSON))
	if err != nil {
		return err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	if err := appendCandidateNames(rows, candidates, r.named); err != nil {
		return err
	}
	r.cache.setNamed(r.generation, r.vaultID, missing, r.named)
	return nil
}

type candidateNames struct {
	paths  []string
	bases  []string
	byPath map[string][]string
	byBase map[string][]string
}

func createCandidateNames(missing []string, named map[string][]string) candidateNames {
	candidates := candidateNames{
		paths: make([]string, 0, len(missing)*2), bases: make([]string, 0, len(missing)),
		byPath: make(map[string][]string, len(missing)*2), byBase: make(map[string][]string, len(missing)),
	}
	for _, name := range missing {
		named[name] = nil
		candidates.add(name)
	}
	return candidates
}

func (c *candidateNames) add(name string) {
	withExt := name + domain.NoteExtension
	base := domain.NormalizeName(domain.LinkName(name))
	c.paths = append(c.paths, name, withExt)
	c.byPath[name] = append(c.byPath[name], name)
	c.byPath[withExt] = append(c.byPath[withExt], name)
	if _, exists := c.byBase[base]; !exists {
		c.bases = append(c.bases, base)
	}
	c.byBase[base] = append(c.byBase[base], name)
}

func appendCandidateNames(rows *sql.Rows, candidates candidateNames, named map[string][]string) error {
	for rows.Next() {
		var path, folded string
		if err := rows.Scan(&path, &folded); err != nil {
			return err
		}
		appendCandidatePath(path, folded, candidates, named)
	}
	return rows.Err()
}

func appendCandidatePath(path, folded string, candidates candidateNames, named map[string][]string) {
	for _, name := range candidates.byPath[path] {
		named[name] = append(named[name], path)
	}
	for _, name := range candidates.byBase[folded] {
		if path != name && path != name+domain.NoteExtension {
			named[name] = append(named[name], path)
		}
	}
}

func (r *resolution) prefetchIdentifiers(ctx context.Context, links []domain.ResolvedLink) (err error) {
	var missing []string
	seen := map[string]bool{}
	for _, l := range links {
		if l.Target.Scheme != domain.SchemeNote {
			continue
		}
		id := l.Target.Value
		if _, ok := r.identified[id]; !ok && !seen[id] {
			seen[id] = true
			missing = append(missing, id)
		}
	}
	if len(missing) <= 1 {
		return nil
	}
	for _, id := range missing {
		r.identified[id] = identifiedNote{}
	}
	idsJSON, err := json.Marshal(missing)
	if err != nil {
		return err
	}
	rows, err := r.statements.QueryContext(ctx, notesByIdentifierAtSQL, string(idsJSON))
	if err != nil {
		return err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	for rows.Next() {
		var id, path string
		var vault domain.VaultID
		if err := rows.Scan(&id, &vault, &path); err != nil {
			return err
		}
		if r.identified[id].path == "" {
			r.identified[id] = identifiedNote{vault: vault, path: path}
		}
	}
	return rows.Err()
}

// ListBacklinks returns the notes that point at one note.
// A link is a backlink because it resolves here, not because its text looks
// like this note: a path link is also filed under its last segment, and a bare
// name may resolve to a nearer note of the same name. Every candidate goes
// through the forward resolution and only those that land here are kept.
func (q *Queries) ListBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, q.cache, vaultID, nil, func(r *resolution) ([]domain.ResolvedLink, error) {
		return r.getBacklinks(ctx, to)
	})
}

func (r *resolution) getBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	var row int64
	var identifier, base string
	err := r.statements.QueryRowContext(ctx, addressingSQL, r.vault, to).
		Scan(&row, &identifier, &base)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	candidates, err := r.getBacklinkCandidates(ctx, identifier, base)
	if err != nil {
		return nil, err
	}
	if err := r.prefetchCandidates(ctx, candidates); err != nil {
		return nil, err
	}

	var out []domain.ResolvedLink
	for _, c := range candidates {
		if err := r.resolve(ctx, c.From, &c); err != nil {
			return nil, err
		}
		// A candidate belongs here when it landed on this path in this vault.
		// An identifier resolves without regard to vault, and two vaults may
		// file a note at the same path.
		if c.To == to && c.ToVault == r.vaultID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *resolution) getBacklinkCandidates(ctx context.Context, identifier, base string) (_ []domain.ResolvedLink, err error) {
	rows, err := r.statements.QueryContext(ctx, backlinkCandidatesSQL,
		identifier, r.vault, base, r.vault)
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
