package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetNotes returns what is needed to show each of the paths asked about. A path
// that is empty is left out.
func (q *Queries) GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
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

// GetTypes is what each of the paths asked about is. A path the index holds no
// note at is left out, and so a listing draws it as the file it is.
func (q *Queries) GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	out := map[string]domain.NoteType{}
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
	// The paths travel as a JSON array, so the statement is one the database
	// can keep whatever number of them arrive.
	wanted, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("types_at"), vault, string(wanted))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path, target string
		if err := rows.Scan(&path, &target); err != nil {
			return nil, err
		}
		out[path] = domain.NoteType(target)
	}
	return out, rows.Err()
}

// GetPathsOfType is every note of one type the vault holds, by path.
func (q *Queries) GetPathsOfType(ctx context.Context, vaultID domain.VaultID, of domain.NoteType) ([]string, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("notes_of_type"), vault, string(of))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// GetHeadings answers NoteQueries.GetHeadings.
func (q *Queries) GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error) {
	out := map[string][]domain.Heading{}
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
	statement, err := q.db.PrepareContext(ctx, stmt.Get("headings_of"))
	if err != nil {
		return nil, err
	}
	defer statement.Close()

	for _, path := range paths {
		found, err := getHeadingsAt(ctx, statement, vault, path)
		if err != nil {
			return nil, err
		}
		if len(found) > 0 {
			out[path] = found
		}
	}
	return out, nil
}

// getHeadingsAt is one note's headings, read off a statement already prepared.
func getHeadingsAt(
	ctx context.Context,
	statement *sql.Stmt,
	vault int64,
	path string,
) ([]domain.Heading, error) {
	rows, err := statement.QueryContext(ctx, vault, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Heading
	for rows.Next() {
		var h domain.Heading
		if err := rows.Scan(&h.Line, &h.Level, &h.Text); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetOpeningNote is the note a vault is shown at when nothing else has been chosen.
func (q *Queries) GetOpeningNote(ctx context.Context, vaultID domain.VaultID) (domain.NoteRef, bool, error) {
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
