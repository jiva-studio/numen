package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetNotes returns the data needed to display each of the given paths. An empty
// path is skipped.
func (q *Queries) GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	return queryEachPath(ctx, q.db, vaultID, paths, notesAtSQL, func(statement *sql.Stmt, vault int64, path string) (domain.NoteRef, bool, error) {
		var ref domain.NoteRef
		err := statement.QueryRowContext(ctx, vault, path).Scan(&ref.Path, &ref.Title, &ref.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return ref, false, nil
		}
		return ref, err == nil, err
	})
}

func queryEachPath[T any](
	ctx context.Context, db *sql.DB, vaultID domain.VaultID, paths []string, query string,
	read func(statement *sql.Stmt, vault int64, path string) (T, bool, error),
) (_ map[string]T, err error) {
	out := map[string]T{}
	if len(paths) == 0 {
		return out, nil
	}
	vault, err := vaultRow(ctx, db, vaultID)
	if errors.Is(err, errNoVault) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}

	statement, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, statement.Close()) }()

	for _, path := range paths {
		found, ok, err := read(statement, vault, path)
		if err != nil {
			return nil, err
		}
		if ok {
			out[path] = found
		}
	}
	return out, nil
}

// GetTypes returns the type of each of the given paths. A path with no indexed
// note is omitted, so a listing shows it as a plain file.
func (q *Queries) GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (_ map[string]domain.NoteType, err error) {
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

	rows, err := q.db.QueryContext(ctx, typesAtSQL, vault, string(wanted))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

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
func (q *Queries) GetPathsOfType(ctx context.Context, vaultID domain.VaultID, of domain.NoteType) (_ []string, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, notesOfTypeSQL, vault, string(of))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

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

func (q *Queries) GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error) {
	return queryEachPath(ctx, q.db, vaultID, paths, headingsOfSQL, func(statement *sql.Stmt, vault int64, path string) ([]domain.Heading, bool, error) {
		found, err := getHeadingsAt(ctx, statement, vault, path)
		return found, len(found) > 0, err
	})
}

func getHeadingsAt(
	ctx context.Context,
	statement *sql.Stmt,
	vault int64,
	path string,
) (_ []domain.Heading, err error) {
	rows, err := statement.QueryContext(ctx, vault, path)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

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

	err = q.db.QueryRowContext(ctx, openingSQL, vault).
		Scan(&ref.Path, &ref.Title, &ref.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return ref, false, nil
	}
	return ref, err == nil, err
}
