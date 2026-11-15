package note

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetNoteCandidates returns the paths of the notes of one vault an address could mean, ordered by path: for
// a note address the notes of the vault with that identifier, for a name the candidates the link resolution
// reads.
func (q *Queries) GetNoteCandidates(ctx context.Context, vaultID domain.VaultID, address domain.Address) ([]string, error) {
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) ([]string, error) {
		switch address.Scheme {
		case domain.SchemeNote:
			return r.queryPaths(ctx, noteInVaultByIdentifierSQL, address.Value, r.vault)
		case domain.SchemeName:
			return r.getCandidates(ctx, address.Value)
		}
		return nil, nil
	})
}

// GetIndexedPaths returns those of the paths that are sources of one vault, ordered by path.
func (q *Queries) GetIndexedPaths(ctx context.Context, vaultID domain.VaultID, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(paths)
	if err != nil {
		return nil, fmt.Errorf("encode the paths: %w", err)
	}
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) ([]string, error) {
		return r.queryPaths(ctx, indexedPathsSQL, r.vault, string(encoded))
	})
}

// ListLinksByName returns the links written in the notes of one vault whose target's last segment folds to
// the name, unresolved, with the path of the note each is written in.
func (q *Queries) ListLinksByName(ctx context.Context, vaultID domain.VaultID, foldedName string) ([]domain.ResolvedLink, error) {
	return runResolution(ctx, q.db, vaultID, nil, func(r *resolution) (_ []domain.ResolvedLink, err error) {
		rows, err := r.statements.QueryContext(ctx, linksByNameSQL, foldedName, r.vault)
		if err != nil {
			return nil, err
		}
		defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
		var out []domain.ResolvedLink
		for rows.Next() {
			var l domain.ResolvedLink
			var role, marker string
			if err := rows.Scan(&l.From, &l.Target.Scheme, &l.Target.Value, &role, &l.Type, &l.Why, &l.Label, &marker); err != nil {
				return nil, err
			}
			l.Role = domain.LinkRole(role)
			l.Marker = domain.LinkMarker(marker)
			out = append(out, l)
		}
		return out, rows.Err()
	})
}

func (r *resolution) queryPaths(ctx context.Context, query string, args ...any) (_ []string, err error) {
	rows, err := r.statements.QueryContext(ctx, query, args...)
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
