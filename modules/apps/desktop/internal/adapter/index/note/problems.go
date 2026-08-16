package note

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Noted is what parsing each file turned up. The wording is the parser's and is
// carried through as it was written.
func (q *Queries) Noted(ctx context.Context, vaultID string) ([]domain.VaultProblem, error) {
	return q.said(ctx, vaultID, "noted")
}

// Unreadable is the notes whose frontmatter is not YAML.
func (q *Queries) Unreadable(ctx context.Context, vaultID string) ([]domain.VaultProblem, error) {
	return q.said(ctx, vaultID, "unreadable")
}

// said reads the two questions that come back as a path and a line of text. What
// the line means is the caller's to say: a query answers what is stored and does
// not compose sentences about it.
func (q *Queries) said(ctx context.Context, vaultID, statement string) ([]domain.VaultProblem, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get(statement), vault)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.VaultProblem
	for rows.Next() {
		var p domain.VaultProblem
		if err := rows.Scan(&p.Path, &p.Detail); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Dangling is every link that reaches nothing.
//
// The query is the whole answer here: a link with no candidate resolves to
// nothing whichever priority rule is applied to it.
func (q *Queries) Dangling(ctx context.Context, vaultID string) ([]domain.ResolvedLink, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("dangling"), vault)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ResolvedLink
	for rows.Next() {
		l, err := scanWritten(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Ambiguous is every link that more than one note answers to.
//
// The query narrows to links written by a shared name; whether one of those is
// really ambiguous is settled by the same resolution a link goes through
// anywhere else. Writing that rule a second time in SQL is how two answers to
// one question start disagreeing.
func (q *Queries) Ambiguous(ctx context.Context, vaultID string) ([]domain.AmbiguousLink, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("shared_name_links"), vault, vault)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []domain.ResolvedLink
	for rows.Next() {
		l, err := scanWritten(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// What a name could mean is a property of the name, not of the link, and
	// these links are here because they share one. Asking once per name rather
	// than once per link is the difference between a question per link and a
	// question per duplicate.
	answers := map[string][]string{}
	var out []domain.AmbiguousLink
	for _, l := range candidates {
		answering, known := answers[l.Target.Value]
		if !known {
			if answering, err = q.candidates(ctx, vault, l.Target.Value); err != nil {
				return nil, err
			}
			answers[l.Target.Value] = answering
		}
		l.To, l.Ambiguous = pick(l.From, l.Target.Value, answering)
		if !l.Ambiguous {
			continue
		}
		l.ToVault = vaultID
		out = append(out, domain.AmbiguousLink{ResolvedLink: l, Candidates: answering})
	}
	return out, nil
}

// scanWritten reads one link as it was written, before anything asks where it
// currently goes.
func scanWritten(rows *sql.Rows) (domain.ResolvedLink, error) {
	var l domain.ResolvedLink
	var role string
	if err := rows.Scan(&l.From, &l.Target.Scheme, &l.Target.Value, &role,
		&l.Type, &l.Note, &l.Label); err != nil {
		return domain.ResolvedLink{}, err
	}
	l.Role = domain.LinkRole(role)
	return l, nil
}
