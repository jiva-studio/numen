package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

// QueryClock provides the current instant for measuring SQL query duration.
type QueryClock interface {
	Now() time.Time
}

// QueryLogFunc receives a structured log entry for an executed SQL query.
type QueryLogFunc func(logs.Entry)

// WithQueryLog records each SQL query executed on the read and write pools.
func WithQueryLog(clock QueryClock, onQuery QueryLogFunc) Option {
	return func(s *settings) {
		s.clock = clock
		s.handleQuery = onQuery
	}
}

func openSQLitePool(dsn, pool string, pos settings) (*sql.DB, error) {
	if pos.handleQuery == nil || pos.clock == nil {
		return sql.Open("sqlite", dsn)
	}
	return sql.OpenDB(&tracedConnector{
		base:        &sqlite.Driver{},
		dsn:         dsn,
		pool:        pool,
		clock:       pos.clock,
		handleQuery: pos.handleQuery,
	}), nil
}

type tracedConnector struct {
	base        driver.Driver
	dsn         string
	pool        string
	clock       QueryClock
	handleQuery func(logs.Entry)
}

func (c *tracedConnector) Connect(_ context.Context) (driver.Conn, error) {
	conn, err := c.base.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &tracedConn{
		conn:        conn,
		pool:        c.pool,
		clock:       c.clock,
		handleQuery: c.handleQuery,
	}, nil
}

func (c *tracedConnector) Driver() driver.Driver { return c.base }

type tracedConn struct {
	conn        driver.Conn
	pool        string
	clock       QueryClock
	handleQuery func(logs.Entry)
}

func (c *tracedConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &tracedStmt{stmt: stmt, query: query, conn: c}, nil
}

func (c *tracedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if prep, ok := c.conn.(driver.ConnPrepareContext); ok {
		stmt, err := prep.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &tracedStmt{stmt: stmt, query: query, conn: c}, nil
	}
	return c.Prepare(query)
}

func (c *tracedConn) Close() error { return c.conn.Close() }

func (c *tracedConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *tracedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.conn.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}
	return nil, driver.ErrSkip
}

func (c *tracedConn) CheckNamedValue(nv *driver.NamedValue) error {
	if checker, ok := c.conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

func (c *tracedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := c.clock.Now()
	res, err := execer.ExecContext(ctx, query, args)
	if !errors.Is(err, driver.ErrSkip) {
		c.record(start, query, args, err)
	}
	return res, err
}

func (c *tracedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := c.clock.Now()
	rows, err := queryer.QueryContext(ctx, query, args)
	if errors.Is(err, driver.ErrSkip) {
		return rows, err
	}
	if err != nil {
		c.record(start, query, args, err)
		return nil, err
	}
	return &tracedRows{rows: rows, conn: c, start: start, query: query, args: args}, nil
}

func (c *tracedConn) record(start time.Time, query string, args []driver.NamedValue, err error) {
	cleaned := stripSQLComments(query)
	if cleaned == "" {
		return
	}
	duration := c.clock.Now().Sub(start)
	level := logs.LevelDebug
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		level = logs.LevelError
	}
	sqlJSON, marshalErr := json.Marshal(cleaned)
	if marshalErr != nil {
		return
	}
	props := map[string]string{
		"pool":       strconv.Quote(c.pool),
		"sql":        string(sqlJSON),
		"args":       encodeNamedValues(args),
		"durationMs": formatDurationMs(duration),
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		props["error"] = strconv.Quote(err.Error())
	}
	c.handleQuery(logs.Entry{
		Time:       start,
		Level:      level,
		Source:     logs.SourceSQL,
		Message:    summarizeSQL(cleaned, duration),
		Properties: props,
	})
}

func stripSQLComments(query string) string {
	var bodyLines []string
	for line := range strings.SplitSeq(query, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n"))
}

func summarizeSQL(query string, duration time.Duration) string {
	var bodyLines []string
	for line := range strings.SplitSeq(query, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		bodyLines = append(bodyLines, trimmed)
	}
	summary := strings.Join(bodyLines, " ")
	if summary == "" {
		summary = "query"
	}
	const maxLen = 72
	if len(summary) > maxLen {
		summary = strings.TrimSpace(summary[:maxLen]) + "..."
	}
	return fmt.Sprintf("SQL: %s (%s ms)", summary, formatDurationMs(duration))
}

type tracedStmt struct {
	stmt  driver.Stmt
	query string
	conn  *tracedConn
}

func (s *tracedStmt) Close() error { return s.stmt.Close() }

func (s *tracedStmt) NumInput() int { return s.stmt.NumInput() }

func (s *tracedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), convertNamedValues(args))
}

func (s *tracedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := s.stmt.(driver.StmtExecContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := s.conn.clock.Now()
	res, err := execer.ExecContext(ctx, args)
	s.conn.record(start, s.query, args, err)
	return res, err
}

func (s *tracedStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), convertNamedValues(args))
}

func (s *tracedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := s.stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := s.conn.clock.Now()
	rows, err := queryer.QueryContext(ctx, args)
	if err != nil {
		s.conn.record(start, s.query, args, err)
		return nil, err
	}
	return &tracedRows{rows: rows, conn: s.conn, start: start, query: s.query, args: args}, nil
}

type tracedRows struct {
	rows     driver.Rows
	conn     *tracedConn
	start    time.Time
	query    string
	args     []driver.NamedValue
	recorded bool
	stepErr  error
}

func (r *tracedRows) Columns() []string { return r.rows.Columns() }

func (r *tracedRows) Next(dest []driver.Value) error {
	err := r.rows.Next(dest)
	if err != nil && !errors.Is(err, io.EOF) {
		r.stepErr = err
	}
	return err
}

func (r *tracedRows) Close() error {
	err := r.rows.Close()
	if !r.recorded {
		r.recorded = true
		recErr := r.stepErr
		if recErr == nil {
			recErr = err
		}
		r.conn.record(r.start, r.query, r.args, recErr)
	}
	return err
}

func convertNamedValues(args []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(args))
	for i, v := range args {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return out
}

func encodeNamedValues(args []driver.NamedValue) string {
	if len(args) == 0 {
		return "[]"
	}
	items := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.Value.(type) {
		case []byte:
			items[i] = fmt.Sprintf("[%d bytes]", len(v))
		default:
			items[i] = v
		}
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func formatDurationMs(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000.0
	return strconv.FormatFloat(ms, 'f', -1, 64)
}
