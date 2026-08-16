package note

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Notes returns what is needed to show each of the paths asked about. A path
// that names nothing is left out.
func (q *Queries) Notes(ctx context.Context, vaultID string, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	if len(paths) == 0 {
		return out, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}

	// One prepared statement, asked repeatedly: the query's text is the same
	// whatever number of paths arrive.
	statement, err := q.db.PrepareContext(ctx, stmt.Get("notes_at"))
	if err != nil {
		return nil, err
	}
	defer statement.Close()

	for _, path := range paths {
		var ref domain.NoteRef
		err := statement.QueryRowContext(ctx, vault, path).Scan(&ref.Path, &ref.Title, &ref.ID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[ref.Path] = ref
	}
	return out, nil
}

// Opening is the note a vault is shown at when nothing else has been chosen.
func (q *Queries) Opening(ctx context.Context, vaultID string) (domain.NoteRef, bool, error) {
	var ref domain.NoteRef
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return ref, false, nil
	}
	if err != nil {
		return ref, false, err
	}

	err = q.db.QueryRowContext(ctx, stmt.Get("opening"), vault).
		Scan(&ref.Path, &ref.Title, &ref.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return ref, false, nil
	}
	return ref, err == nil, err
}
