package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Problems returns what a scan could not act on and did not guess at: a link
// with no role, a role nobody decided on, frontmatter that will not parse.
//
// The two sources are deliberate. A broken frontmatter block belongs to the note
// it broke, so it is stored on the note; everything else is a list. Reading them
// together is what makes them one view.
func (q *Queries) Problems(ctx context.Context, vaultID string) ([]domain.VaultProblem, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("problems"), vault, vault)
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
