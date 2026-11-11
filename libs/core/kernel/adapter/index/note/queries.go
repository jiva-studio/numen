package note

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Queries answers queries about notes in shapes other than notes: what the
// index records about each file, what a name resolves to, and how many there are.
type Queries struct{ db *sql.DB }

func NewQueries(db *sql.DB) *Queries { return &Queries{db: db} }

// GetFileStats is what the index records about each note of the vault, keyed by path.
func (q *Queries) GetFileStats(ctx context.Context, vaultID domain.VaultID) (map[string]domain.FileStat, error) {
	return chunk.NewFiles(q.db).GetFileStats(ctx, vaultID, kind)
}

// Has reports whether the index carries this vault at all.
func (q *Queries) Has(ctx context.Context, vaultID domain.VaultID) (bool, error) {
	_, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (q *Queries) GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error) {
	var s domain.VaultSummary
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = q.db.QueryRowContext(ctx, summarySQL, vault, vault).Scan(&s.Notes, &s.Headings)
	return s, err
}

// GetNamedPaths is every note filed under one name. A name that answers for
// more than one note is what makes a link written by that name ambiguous, and
// is worth saying out loud before it surprises anyone.
func (q *Queries) GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) (_ []string, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, namedSQL, vault, domain.FoldName(name))
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

// mostPerNote is how many headings of one note an answer carries.
const mostPerNote = 2

// GetNameMatches returns the note titles and the headings inside notes that match the words typed.
// Titles are ranked apart and come first, so the limit cuts a heading of a note
// already named or a weaker name match. What one note may contribute is
// settled before the limit applies.
// note may contribute is settled before the limit is applied.
func (q *Queries) GetNameMatches(ctx context.Context, vaultID domain.VaultID, query string, limit int) (_ []domain.NameMatch, err error) {
	if limit <= 0 {
		// The result count is the caller's decision, not a database adapter's; a
		// missing limit is a caller error.
		return nil, fmt.Errorf("titles limit must be positive, got %d", limit)
	}

	words := chunk.Expression(query, true)
	if words == "" {
		return nil, nil
	}

	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, titlesSQL,
		opening, closing, words, vault, limit,
		opening, closing, words, vault, mostPerNote, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	out := make([]domain.NameMatch, 0, limit)
	for rows.Next() {
		var kind, line int
		var marked, target string
		var score float64
		var m domain.NameMatch
		if err := rows.Scan(&kind, &m.Path, &m.Title, &target, &line, &marked, &score); err != nil {
			return nil, err
		}
		m.Type = domain.NoteType(target)
		name, pos := parseMarkedName(marked)
		m.Ranges = pos
		if kind != 0 {
			m.Heading, m.Line = name, line
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
