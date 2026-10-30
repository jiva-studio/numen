package note

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GetLinks returns the links written in one note, resolved.
func (q *Queries) GetLinks(ctx context.Context, vaultID domain.VaultID, from string) (_ []domain.ResolvedLink, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var row int64
	err = q.db.QueryRowContext(ctx, stmt.Get("identify"), vault, from).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("links_of"), row)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.ResolvedLink
	for rows.Next() {
		var r domain.ResolvedLink
		var role string
		if err := rows.Scan(&r.Target.Scheme, &r.Target.Value, &role,
			&r.Type, &r.Why, &r.Label); err != nil {
			return nil, err
		}
		r.Role = domain.LinkRole(role)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		out[i].From = from
		if err := q.resolve(ctx, vault, vaultID, from, &out[i]); err != nil {
			return nil, err
		}
	}
	return dedupe(out), nil
}

// dedupe merges links that resolve to the same note, which happens when one
// note is written two ways: [[notes/Entropy]] in the links block and [[Entropy]]
// in prose are one link, and the described one wins.
//
// Deduplication runs after resolution, which determines that two different
// strings denote one note.
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
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	request := make(map[string]bool, len(content))
	out := make(map[string]domain.ResolvedLink, len(content))
	for _, raw := range content {
		if request[raw] {
			continue
		}
		request[raw] = true
		one := domain.ResolvedLink{Link: domain.Link{Target: domain.ParseAddress(raw)}, From: from}
		if err := q.resolve(ctx, vault, vaultID, from, &one); err != nil {
			return nil, err
		}
		if one.To != "" {
			out[raw] = one
		}
	}
	return out, nil
}

func (q *Queries) resolve(ctx context.Context, vault int64, vaultID domain.VaultID, from string, r *domain.ResolvedLink) error {
	switch r.Target.Scheme {
	case domain.SchemeNote:
		// An identifier names one note globally, so this lookup is not scoped to a
		// vault. It is the one deliberate exception to the rule that a query without a
		// vault must not read another vault's notes.
		err := q.db.QueryRowContext(ctx, stmt.Get("note_by_identifier"), r.Target.Value).
			Scan(&r.ToVault, &r.To)
		if errors.Is(err, sql.ErrNoRows) {
			// Neither dangling nor resolved: no connected vault holds this note. The
			// index cannot distinguish a deleted note from one in a vault that is not
			// connected, and reporting it as broken would be wrong on a machine where both
			// vaults are open.
			return nil
		}
		return err
	case domain.SchemeName:
		candidates, err := q.candidates(ctx, vault, r.Target.Value)
		if err != nil {
			return err
		}
		r.To, r.IsAmbiguous = domain.ChooseMatch(from, r.Target.Value, candidates)
		if r.To != "" {
			// A name means something only inside the vault it was written in.
			r.ToVault = vaultID
		}
		return nil
	default:
		// An asset or a URL: not a note, so no note resolves it.
		return nil
	}
}

func (q *Queries) candidates(ctx context.Context, vault int64, name string) (_ []string, err error) {
	// Three shapes of the same question, so one query answers all of them: the
	// name as a path, as a path with an extension added, and as a filename.
	withExt := name + domain.NoteExtension
	base := domain.FoldName(domain.LinkName(name))

	rows, err := q.db.QueryContext(ctx, stmt.Get("candidates"),
		vault, name, vault, withExt, vault, base)
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
//
// A link is a backlink because it *resolves* here, not because its text looks
// like this note. Those differ in both directions: a link written as a path is
// filed under its last segment and so is offered by name as well, and a link
// written as a bare name may resolve to a nearer note of the same name. So
// every candidate goes through the same resolution the forward direction uses,
// and only the ones that land here are kept.
func (q *Queries) GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) (_ []domain.ResolvedLink, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var row int64
	var identifier, base string
	err = q.db.QueryRowContext(ctx, stmt.Get("addressing"), vault, to).
		Scan(&row, &identifier, &base)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("backlink_candidates"),
		identifier, vault, base, vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var candidates []domain.ResolvedLink
	for rows.Next() {
		var r domain.ResolvedLink
		var role string
		var position int
		if err := rows.Scan(&r.From, &r.Target.Scheme, &r.Target.Value, &role,
			&r.Type, &r.Why, &r.Label, &position); err != nil {
			return nil, err
		}
		r.Role = domain.LinkRole(role)
		candidates = append(candidates, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []domain.ResolvedLink
	for _, c := range candidates {
		if err := q.resolve(ctx, vault, vaultID, c.From, &c); err != nil {
			return nil, err
		}
		// A candidate belongs here when it landed on this path in this vault.
		// An identifier resolves without regard to vault, and two vaults may
		// file a note at the same path.
		if c.To == to && c.ToVault == vaultID {
			out = append(out, c)
		}
	}
	return out, nil
}
