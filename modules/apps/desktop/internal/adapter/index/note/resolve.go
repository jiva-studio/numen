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
	return dedupe(out), nil
}

// dedupe folds links that turn out to point at the same note, which the same
// note written two ways does: [[notes/Entropy]] in the links block and
// [[Entropy]] in prose are one link, and the described one wins (ADR-0003).
//
// It happens here rather than at parse time because only resolution knows that
// two different strings mean one note.
func dedupe(links []domain.ResolvedLink) []domain.ResolvedLink {
	seen := map[string]int{}
	var out []domain.ResolvedLink
	for _, l := range links {
		key := l.To
		if key == "" {
			// Unresolved links are only the same when written the same: nothing
			// here knows what they would have meant.
			key = "\x00" + l.Target.String()
		}
		at, known := seen[key]
		if !known {
			seen[key] = len(out)
			out = append(out, l)
			continue
		}
		if out[at].Role == domain.RoleRef && l.Role != domain.RoleRef {
			// A mention in prose gives way to the record that carries a role.
			out[at] = l
		}
	}
	return out
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
		vaultID, name, vaultID, withExt, vaultID, base)
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
		if strings.EqualFold(c, wanted) {
			return c, false
		}
	}

	relative := path.Join(path.Dir(from), wanted)
	for _, c := range candidates {
		if strings.EqualFold(c, relative) {
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

// Backlinks returns the notes that point at one note.
//
// A link is a backlink because it *resolves* here, not because its text looks
// like this note. Those differ in both directions: a link written as a path
// never matches by name, and a link written as a bare name may resolve to a
// nearer note of the same name. So every candidate goes through the same
// resolution the forward direction uses, and only the ones that land here are
// kept.
func (q *Queries) Backlinks(ctx context.Context, vaultID, to string) ([]domain.ResolvedLink, error) {
	var noteID, base string
	err := q.db.QueryRowContext(ctx, stmt.Get("addressing"), vaultID, to).Scan(&noteID, &base)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("backlink_candidates"),
		vaultID, noteID, vaultID, base)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []domain.ResolvedLink
	for rows.Next() {
		var r domain.ResolvedLink
		var role string
		var position int
		if err := rows.Scan(&r.From, &r.Target.Scheme, &r.Target.Value, &role,
			&r.Type, &r.Note, &r.Label, &position); err != nil {
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
		if err := q.resolve(ctx, vaultID, c.From, &c); err != nil {
			return nil, err
		}
		if c.To == to {
			out = append(out, c)
		}
	}
	return out, nil
}
