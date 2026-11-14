package note

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func (q *Queries) GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) ([]domain.ResolvedLink, error) {
		return r.getLinks(ctx, from)
	})
}

func (r *resolution) getLinks(ctx context.Context, from string) (_ []domain.ResolvedLink, err error) {
	var row int64
	err = r.reads.QueryRowContext(ctx, identifySQL, r.vault, from).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.reads.QueryContext(ctx, linksOfSQL, row)
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
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) (map[string]domain.ResolvedLink, error) {
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
		// An identifier names one note globally, so this lookup is not scoped to a
		// vault. It is the one deliberate exception to the rule that a query without a
		// vault must not read another vault's notes.
		err := r.reads.QueryRowContext(ctx, noteByIdentifierSQL, l.Target.Value).
			Scan(&l.ToVault, &l.To)
		if errors.Is(err, sql.ErrNoRows) {
			// Neither dangling nor resolved: no connected vault holds this note. The
			// index cannot distinguish a deleted note from one in a vault that is not
			// connected, and reporting it as broken would be wrong on a machine where both
			// vaults are open.
			return nil
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
	found, err := r.queryCandidates(ctx, name)
	if err != nil {
		return nil, err
	}
	r.named[name] = found
	return found, nil
}

func (r *resolution) queryCandidates(ctx context.Context, name string) (_ []string, err error) {
	// Three shapes of the same question, so one query answers all of them: the
	// name as a path, as a path with an extension added, and as a filename.
	withExt := name + domain.NoteExtension
	base := domain.FoldName(domain.LinkName(name))

	rows, err := r.reads.QueryContext(ctx, candidatesSQL,
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

// GetBacklinks returns the notes that point at one note.
// A link is a backlink because it resolves here, not because its text looks
// like this note: a path link is also filed under its last segment, and a bare
// name may resolve to a nearer note of the same name. Every candidate goes
// through the forward resolution and only those that land here are kept.
func (q *Queries) GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) ([]domain.ResolvedLink, error) {
		return r.getBacklinks(ctx, to)
	})
}

func (r *resolution) getBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	var row int64
	var identifier, base string
	err := r.reads.QueryRowContext(ctx, addressingSQL, r.vault, to).
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
	rows, err := r.reads.QueryContext(ctx, backlinkCandidatesSQL,
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
