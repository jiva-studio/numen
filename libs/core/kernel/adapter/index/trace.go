package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
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
		s.onQuery = onQuery
	}
}

func openSQLitePool(dsn, pool string, pos settings) (*sql.DB, error) {
	if pos.onQuery == nil || pos.clock == nil {
		return sql.Open("sqlite", dsn)
	}
	return sql.OpenDB(&tracedConnector{
		base:    &sqlite.Driver{},
		dsn:     dsn,
		pool:    pool,
		clock:   pos.clock,
		onQuery: pos.onQuery,
	}), nil
}

type tracedConnector struct {
	base    driver.Driver
	dsn     string
	pool    string
	clock   QueryClock
	onQuery func(logs.Entry)
}

func (c *tracedConnector) Connect(_ context.Context) (driver.Conn, error) {
	conn, err := c.base.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &tracedConn{
		conn:    conn,
		pool:    c.pool,
		clock:   c.clock,
		onQuery: c.onQuery,
	}, nil
}

func (c *tracedConnector) Driver() driver.Driver { return c.base }

type tracedConn struct {
	conn    driver.Conn
	pool    string
	clock   QueryClock
	onQuery func(logs.Entry)
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
	if !errors.Is(err, driver.ErrSkip) {
		c.record(start, query, args, err)
	}
	return rows, err
}

func (c *tracedConn) record(start time.Time, query string, args []driver.NamedValue, err error) {
	cleaned := strings.TrimSpace(query)
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
	c.onQuery(logs.Entry{
		Time:       start,
		Level:      level,
		Source:     logs.SourceSQL,
		Message:    "SQL query",
		Properties: props,
	})
}

type tracedStmt struct {
	stmt  driver.Stmt
	query string
	conn  *tracedConn
}

func (s *tracedStmt) Close() error { return s.stmt.Close() }

func (s *tracedStmt) NumInput() int { return s.stmt.NumInput() }

func (s *tracedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), toNamedValues(args))
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
	return s.QueryContext(context.Background(), toNamedValues(args))
}

func (s *tracedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := s.stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	start := s.conn.clock.Now()
	rows, err := queryer.QueryContext(ctx, args)
	s.conn.record(start, s.query, args, err)
	return rows, err
}

func toNamedValues(args []driver.Value) []driver.NamedValue {
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
