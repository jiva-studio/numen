package note

import (
	"context"
	"database/sql"
	"errors"
	"path"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Links returns the links written in one note, resolved.
func (q *Queries) Links(ctx context.Context, vaultID, from string) ([]domain.ResolvedLink, error) {
	rows, err := q.db.QueryContext(ctx, stmt.Get("links_of"), vaultID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ResolvedLink
	for rows.Next() {
		var r domain.ResolvedLink
		var role string
		if err := rows.Scan(&r.Target.Scheme, &r.Target.Value, &role,
			&r.Type, &r.Note, &r.Label); err != nil {
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
		if err := q.resolve(ctx, vaultID, from, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (q *Queries) resolve(ctx context.Context, vaultID, from string, r *domain.ResolvedLink) error {
	switch r.Target.Scheme {
	case domain.SchemeNote:
		// An identifier names one note in the world, so this lookup is not
		// scoped to a vault — the one deliberate exception to the rule that a
		// query without a vault is a leak (ADR-0011).
		err := q.db.QueryRowContext(ctx, stmt.Get("note_by_id"), r.Target.Value).
			Scan(&r.ToVault, &r.To)
		if errors.Is(err, sql.ErrNoRows) {
			// Not dangling and not resolved: no connected vault holds this note.
			// Whether it was deleted or simply lives in a vault the user has not
			// added, nothing here can tell, and guessing would report a link as
			// broken that is fine on the machine where both vaults are open.
			return nil
		}
		return err
	case domain.SchemeName:
		candidates, err := q.candidates(ctx, vaultID, r.Target.Value)
		if err != nil {
			return err
		}
		r.To, r.Ambiguous = pick(from, r.Target.Value, candidates)
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

func (q *Queries) candidates(ctx context.Context, vaultID, name string) ([]string, error) {
	// Four shapes of the same question, so one query answers all of them: the
	// name as a path with and without an extension, and as a filename.
	withExt := name
	if path.Ext(withExt) == "" {
		withExt += ".md"
	}
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))

	rows, err := q.db.QueryContext(ctx, stmt.Get("candidates"),
		vaultID, name, withExt, base, path.Base(name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// pick applies the priority in ADR-0011: an exact path from the root, then a
// path relative to the note the link is written in, then a single match by
// name. Several matches are ambiguous rather than dangling — the link resolves
// to the nearest one in the tree, and the ambiguity is something to show.
func pick(from, name string, candidates []string) (chosen string, ambiguous bool) {
	if len(candidates) == 0 {
		return "", false
	}

	wanted := name
	if path.Ext(wanted) == "" {
		wanted += ".md"
	}
	for _, c := range candidates {
		if c == wanted {
			return c, false
		}
	}

	relative := path.Join(path.Dir(from), wanted)
	for _, c := range candidates {
		if c == relative {
			return c, false
		}
	}

	if len(candidates) == 1 {
		return candidates[0], false
	}

	// Nearest by how much of the path they have in common with the note the
	// link was written in — the neighbour is likelier than the stranger.
	best, bestShared := candidates[0], -1
	for _, c := range candidates {
		if shared := sharedPrefix(from, c); shared > bestShared {
			best, bestShared = c, shared
		}
	}
	return best, true
}

func sharedPrefix(a, b string) int {
	as, bs := strings.Split(path.Dir(a), "/"), strings.Split(path.Dir(b), "/")
	n := 0
	for n < len(as) && n < len(bs) && as[n] == bs[n] {
		n++
	}
	return n
}

// Backlinks returns the notes pointing at one note, by whichever form was
// written: its identifier, or a name that resolves to it.
func (q *Queries) Backlinks(ctx context.Context, vaultID, to string) ([]domain.ResolvedLink, error) {
	var noteID string
	var basename string
	if err := q.db.QueryRowContext(ctx,
		stmt.Get("addressing"),
		vaultID, to).Scan(&noteID, &basename); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("backlinks"), vaultID, noteID, basename, basename)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ResolvedLink
	for rows.Next() {
		var r domain.ResolvedLink
		var role string
		if err := rows.Scan(&r.From, &role, &r.Type); err != nil {
			return nil, err
		}
		r.Role = domain.LinkRole(role)
		out = append(out, r)
	}
	return out, rows.Err()
}
