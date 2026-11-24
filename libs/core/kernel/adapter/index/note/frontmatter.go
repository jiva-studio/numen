package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListFrontmatter returns the stored frontmatter of the notes the query names, in path order. A note
// without frontmatter has no values; a path with no note is left out.
func (q *QueryReads) ListFrontmatter(ctx context.Context, vaultID domain.VaultID, query frontmatter.Query) ([]frontmatter.Note, error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(query.Paths) == 0 {
		return q.listFrontmatterAfter(ctx, vault, query)
	}
	return q.listFrontmatterAt(ctx, vault, query)
}

func (q *QueryReads) listFrontmatterAfter(ctx context.Context, vault int64, query frontmatter.Query) (_ []frontmatter.Note, err error) {
	rows, err := q.db.QueryContext(ctx, frontmatterAfterSQL, vault, query.After, query.Limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []frontmatter.Note
	for rows.Next() {
		var source int64
		var path string
		var stored sql.NullString
		if err := rows.Scan(&source, &path, &stored); err != nil {
			return nil, err
		}
		values, err := readValues(stored, query.Keys)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, frontmatter.Note{SourceID: filechange.SourceID(source), Path: path, Values: values})
	}
	return out, rows.Err()
}

func (q *QueryReads) listFrontmatterAt(ctx context.Context, vault int64, query frontmatter.Query) (_ []frontmatter.Note, err error) {
	statement, err := prepareStatement(ctx, q.db, frontmatterAtSQL)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, statement.Close()) }()
	paths := slices.Compact(slices.Sorted(slices.Values(query.Paths)))
	var out []frontmatter.Note
	for _, path := range paths {
		if path <= query.After {
			continue
		}
		if len(out) >= query.Limit {
			break
		}
		var source int64
		var stored sql.NullString
		err := statement.QueryRowContext(ctx, vault, path).Scan(&source, &path, &stored)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		values, err := readValues(stored, query.Keys)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, frontmatter.Note{SourceID: filechange.SourceID(source), Path: path, Values: values})
	}
	return out, nil
}

func readValues(stored sql.NullString, keys []string) (map[string]any, error) {
	values := map[string]any{}
	if !stored.Valid {
		return values, nil
	}
	decoder := json.NewDecoder(strings.NewReader(stored.String))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("read the stored frontmatter: %w", err)
	}
	if len(keys) == 0 {
		return values, nil
	}
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

// FrontmatterChanges hands the paths of the notes whose stored frontmatter changed to the watchers of
// their vault, once the write that changed them is committed.
type FrontmatterChanges struct {
	mu       sync.Mutex
	next     int
	watchers map[int]frontmatterWatcher
}

type frontmatterWatcher struct {
	vault  domain.VaultID
	report func(paths []string)
}

// Watch hands report the paths of each write of the vault that changed frontmatter, until the
// function it returns is called. report runs inside the write's caller and does not block.
func (c *FrontmatterChanges) Watch(vaultID domain.VaultID, report func(paths []string)) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.watchers == nil {
		c.watchers = map[int]frontmatterWatcher{}
	}
	id := c.next
	c.next++
	c.watchers[id] = frontmatterWatcher{vault: vaultID, report: report}
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.watchers, id)
	}
}

func (c *FrontmatterChanges) publish(vaultID domain.VaultID, paths []string) {
	if c == nil || len(paths) == 0 {
		return
	}
	c.mu.Lock()
	var reports []func([]string)
	for _, one := range c.watchers {
		if one.vault == vaultID {
			reports = append(reports, one.report)
		}
	}
	c.mu.Unlock()
	for _, report := range reports {
		report(slices.Clone(paths))
	}
}
