package note

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Problems answers what the index found wrong in the notes of a vault: parse problems, unreadable
// frontmatter and links that point nowhere or at several notes.
type Problems struct{ db *sql.DB }

// NewProblems creates the queries about problems over the read pool.
func NewProblems(db *sql.DB) *Problems { return &Problems{db: db} }

// GetParseProblems is what parsing each file turned up. The wording is the
// parser's and is carried through as it was written.
func (q *Problems) GetParseProblems(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error) {
	return q.getProblems(ctx, vaultID, problemsSQL)
}

// GetUnreadable is the notes whose frontmatter is not YAML.
func (q *Problems) GetUnreadable(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error) {
	return q.getProblems(ctx, vaultID, frontmatterErrorsSQL)
}

func (q *Problems) getProblems(ctx context.Context, vaultID domain.VaultID, query string) (_ []domain.VaultProblem, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, query, vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

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

// GetDanglingLinks returns every link that resolves to nothing.
//
// The query is the whole answer here: a link with no candidate resolves to
// nothing whichever priority rule is applied to it.
func (q *Problems) GetDanglingLinks(ctx context.Context, vaultID domain.VaultID) (_ []domain.ResolvedLink, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, danglingSQL, vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.ResolvedLink
	for rows.Next() {
		l, err := scanLinkAsWritten(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetAmbiguousLinks returns every link that more than one note matches.
// The query narrows to links written by a shared name; the same resolution a
// link goes through anywhere else decides ambiguity, so SQL does not duplicate
// that rule.
// diverge.
func (q *Problems) GetAmbiguousLinks(ctx context.Context, vaultID domain.VaultID) ([]domain.AmbiguousLink, error) {
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) ([]domain.AmbiguousLink, error) {
		return r.getAmbiguousLinks(ctx)
	})
}

func (r *resolution) getAmbiguousLinks(ctx context.Context) (_ []domain.AmbiguousLink, err error) {
	rows, err := r.reads.QueryContext(ctx, sharedNameLinksSQL, r.vault, r.vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var candidates []domain.ResolvedLink
	for rows.Next() {
		l, err := scanLinkAsWritten(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []domain.AmbiguousLink
	for _, l := range candidates {
		answering, err := r.getCandidates(ctx, l.Target.Value)
		if err != nil {
			return nil, err
		}
		l.To, l.IsAmbiguous = domain.ChooseMatch(l.From, l.Target.Value, answering)
		if !l.IsAmbiguous {
			continue
		}
		l.ToVault = r.vaultID
		out = append(out, domain.AmbiguousLink{ResolvedLink: l, Candidates: answering})
	}
	return out, nil
}

func scanLinkAsWritten(rows *sql.Rows) (domain.ResolvedLink, error) {
	var l domain.ResolvedLink
	var role, marker string
	if err := rows.Scan(&l.From, &l.Target.Scheme, &l.Target.Value, &role,
		&l.Type, &l.Why, &l.Label, &marker); err != nil {
		return domain.ResolvedLink{}, err
	}
	l.Role = domain.LinkRole(role)
	l.Marker = domain.LinkMarker(marker)
	return l, nil
}
