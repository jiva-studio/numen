package note

import (
	"context"
	"database/sql"
	"errors"
	"path"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Links returns the links written in one note, resolved.
func (q *Queries) Links(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error) {
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
	defer rows.Close()

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

// dedupe folds links that turn out to point at the same note, which the same
// note written two ways does: [[notes/Entropy]] in the links block and
// [[Entropy]] in prose are one link, and the described one wins.
//
// It happens here because only resolution knows that two different strings
// mean one note.
func dedupe(links []domain.ResolvedLink) []domain.ResolvedLink {
	seen := map[string]int{}
	var out []domain.ResolvedLink
	for _, l := range links {
		key := l.To
		if key == "" {
			// Unresolved links are only the same when written the same:
			// nothing here knows what they meant.
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

// Resolve answers where addresses written in one note land, keyed by what was
// written. The priority is the one every name resolves by, so a wikilink
// nobody recorded as a link is answered as a link is.
func (q *Queries) Resolve(
	ctx context.Context, vaultID domain.VaultID, from string, written []string,
) (map[string]domain.ResolvedLink, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	asked := make(map[string]bool, len(written))
	out := make(map[string]domain.ResolvedLink, len(written))
	for _, raw := range written {
		if asked[raw] {
			continue
		}
		asked[raw] = true
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
		// An identifier names one note in the world, so this lookup is not
		// scoped to a vault — the one deliberate exception to the rule that a
		// query without a vault reads someone else's notes.
		err := q.db.QueryRowContext(ctx, stmt.Get("note_by_identifier"), r.Target.Value).
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
		candidates, err := q.candidates(ctx, vault, r.Target.Value)
		if err != nil {
			return err
		}
		r.To, r.IsAmbiguous = pick(from, r.Target.Value, candidates)
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

func (q *Queries) candidates(ctx context.Context, vault int64, name string) ([]string, error) {
	// Three shapes of the same question, so one query answers all of them: the
	// name as a path, as a path with an extension added, and as a filename.
	withExt := name + domain.NoteExtension
	base := domain.FoldName(domain.LinkName(name))

	rows, err := q.db.QueryContext(ctx, stmt.Get("candidates"),
		vault, name, vault, withExt, vault, base)
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

// pick applies the priority a name resolves by: an exact path from the root,
// then a path relative to the note the link is written in, then a single match
// by name. Several matches are ambiguous: the link resolves to the nearest one
// in the tree, and the ambiguity is shown.
//
// A path is matched under domain.FoldName: every folder on the way to a note
// answers the way the note's own name does.
func pick(from, name string, candidates []string) (chosen string, ambiguous bool) {
	if len(candidates) == 0 {
		return "", false
	}

	folded := make([]string, len(candidates))
	for i, c := range candidates {
		folded[i] = domain.FoldName(c)
	}

	// A name is written with an extension or without one, and both spellings
	// name the same file.
	wanted := []string{name, name + domain.NoteExtension}
	for _, w := range wanted {
		if at := where(folded, domain.FoldName(w)); at >= 0 {
			return candidates[at], false
		}
	}

	for _, w := range wanted {
		relative := domain.FoldName(path.Join(path.Dir(from), w))
		if at := where(folded, relative); at >= 0 {
			return candidates[at], false
		}
	}

	if len(candidates) == 1 {
		return candidates[0], false
	}

	// Nearest by how much of the path they have in common with the note the
	// link was written in — the neighbour is likelier than the stranger.
	best, bestShared := candidates[0], -1
	for _, c := range candidates {
		if shared := countSharedPrefix(from, c); shared > bestShared {
			best, bestShared = c, shared
		}
	}
	return best, true
}

// where one key stands among many, and -1 when none of them is it.
func where(keys []string, want string) int {
	for i, key := range keys {
		if key == want {
			return i
		}
	}
	return -1
}

func countSharedPrefix(a, b string) int {
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
// like this note. Those differ in both directions: a link written as a path is
// filed under its last segment and so is offered by name as well, and a link
// written as a bare name may resolve to a nearer note of the same name. So
// every candidate goes through the same resolution the forward direction uses,
// and only the ones that land here are kept.
func (q *Queries) Backlinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error) {
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
	defer rows.Close()

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
