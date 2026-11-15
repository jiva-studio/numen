// Package wordmatch matches FTS5 queries against texts held in memory, with the tokenizer of the
// index, so a query matches a text here exactly where it matches the same text in the index.
package wordmatch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

// createTableSQL declares the table with the tokenizer options of chunks_fts: the default unicode61.
const createTableSQL = `CREATE VIRTUAL TABLE temp.texts USING fts5 (text)`

const insertSQL = `INSERT INTO temp.texts (rowid, text) VALUES (?, ?)`

const matchSQL = `SELECT rowid FROM temp.texts WHERE texts MATCH ?`

// Matcher matches FTS5 queries against texts in an in-memory database opened for each call.
type Matcher struct{}

// NewMatcher returns the matcher.
func NewMatcher() Matcher { return Matcher{} }

// MatchWords reports for each text whether it matches the FTS5 expression, in the order of the
// texts. An expression FTS5 cannot parse returns source.ErrInvalidWords.
func (Matcher) MatchWords(ctx context.Context, expression string, texts []string) (_ []bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return []bool{}, nil
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open the word matcher: %w", err)
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open the word matcher: %w", err)
	}
	defer func() { err = errors.Join(err, conn.Close()) }()

	if err := insertTexts(ctx, conn, texts); err != nil {
		return nil, err
	}
	matched, err := readMatches(ctx, conn, expression, len(texts))
	if isQueryError(err) {
		return nil, fmt.Errorf("%w: %w", source.ErrInvalidWords, err)
	}
	return matched, err
}

func insertTexts(ctx context.Context, conn *sql.Conn, texts []string) (err error) {
	if _, err := conn.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("create the word table: %w", err)
	}
	insert, err := conn.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("prepare the word table: %w", err)
	}
	defer func() { err = errors.Join(err, insert.Close()) }()
	for i, text := range texts {
		if _, err := insert.ExecContext(ctx, i+1, text); err != nil {
			return fmt.Errorf("fill the word table: %w", err)
		}
	}
	return nil
}

func readMatches(ctx context.Context, conn *sql.Conn, expression string, count int) (_ []bool, err error) {
	rows, err := conn.QueryContext(ctx, matchSQL, expression)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	matched := make([]bool, count)
	for rows.Next() {
		var row int
		if err := rows.Scan(&row); err != nil {
			return nil, err
		}
		if row >= 1 && row <= count {
			matched[row-1] = true
		}
	}
	return matched, rows.Err()
}

func isQueryError(err error) bool {
	// The statement is fixed and the expression is bound, so SQLITE_ERROR comes from the expression.
	var response *sqlite.Error
	return errors.As(err, &response) && response.Code()&0xff == sqlite3.SQLITE_ERROR
}
