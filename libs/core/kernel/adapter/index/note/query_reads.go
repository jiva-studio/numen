package note

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// QueryReads answers the reads of the search query language over the notes: what their frontmatter and headings hold.
type QueryReads struct {
	db    *sql.DB
	cache *Cache
}

func NewQueryReads(db *sql.DB) *QueryReads { return &QueryReads{db: db} }

// WithCache returns a copy of QueryReads using cache for read lookups.
func (q *QueryReads) WithCache(cache *Cache) *QueryReads {
	return &QueryReads{db: q.db, cache: cache}
}

// ListPropertyValues returns the path and the value of every note of the vault whose stored frontmatter has the
// top-level key. A note without frontmatter, or whose frontmatter could not be read, is absent.
func (q *QueryReads) ListPropertyValues(ctx context.Context, vaultID domain.VaultID, key string) (_ []searchquery.PropertyEntry, err error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, propertyValuesSQL, vault, key)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []searchquery.PropertyEntry
	for rows.Next() {
		var path, kind string
		var value any
		if err := rows.Scan(&path, &kind, &value); err != nil {
			return nil, err
		}
		decoded, err := decodePropertyValue(kind, value)
		if err != nil {
			return nil, fmt.Errorf("read the frontmatter key %q of %s: %w", key, path, err)
		}
		out = append(out, searchquery.PropertyEntry{Path: path, Value: decoded})
	}
	return out, rows.Err()
}

func decodePropertyValue(kind string, value any) (searchquery.PropertyValue, error) {
	switch kind {
	case "null":
		return searchquery.PropertyValue{Kind: searchquery.PropertyNull}, nil
	case "true", "false":
		return searchquery.PropertyValue{Kind: searchquery.PropertyBool, Bool: kind == "true"}, nil
	case "integer":
		number, ok := value.(int64)
		if !ok {
			return searchquery.PropertyValue{}, fmt.Errorf("an integer arrived as %T", value)
		}
		return searchquery.PropertyValue{Kind: searchquery.PropertyInt, Int: number}, nil
	case "real":
		number, ok := value.(float64)
		if !ok {
			return searchquery.PropertyValue{}, fmt.Errorf("a number arrived as %T", value)
		}
		return searchquery.PropertyValue{Kind: searchquery.PropertyDouble, Double: number}, nil
	case "text":
		return searchquery.PropertyValue{Kind: searchquery.PropertyString, Text: getText(value)}, nil
	}
	return decodeJSON([]byte(getText(value)))
}

func getText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return ""
}

func decodeJSON(raw []byte) (searchquery.PropertyValue, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return searchquery.PropertyValue{}, err
	}
	return convertJSON(value), nil
}

func convertJSON(value any) searchquery.PropertyValue {
	switch v := value.(type) {
	case bool:
		return searchquery.PropertyValue{Kind: searchquery.PropertyBool, Bool: v}
	case json.Number:
		return convertNumber(v)
	case string:
		return searchquery.PropertyValue{Kind: searchquery.PropertyString, Text: v}
	case []any:
		items := make([]searchquery.PropertyValue, len(v))
		for i, item := range v {
			items[i] = convertJSON(item)
		}
		return searchquery.PropertyValue{Kind: searchquery.PropertyList, Items: items}
	case map[string]any:
		return searchquery.PropertyValue{Kind: searchquery.PropertyMap}
	}
	return searchquery.PropertyValue{Kind: searchquery.PropertyNull}
}

func convertNumber(number json.Number) searchquery.PropertyValue {
	if !strings.ContainsAny(number.String(), ".eE") {
		if whole, err := strconv.ParseInt(number.String(), 10, 64); err == nil {
			return searchquery.PropertyValue{Kind: searchquery.PropertyInt, Int: whole}
		}
	}
	fraction, err := number.Float64()
	if err != nil {
		return searchquery.PropertyValue{Kind: searchquery.PropertyString, Text: number.String()}
	}
	return searchquery.PropertyValue{Kind: searchquery.PropertyDouble, Double: fraction}
}

// FindHeadings returns the headings of the notes of the vault whose words hold every word of the text, as one
// phrase of the heading index, with the path of their note; every heading of the vault when the text has no
// word.
func (q *QueryReads) FindHeadings(ctx context.Context, vaultID domain.VaultID, text string) (_ []searchquery.HeadingEntry, err error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	statement, args := allHeadingsSQL, []any{vault}
	phrase := `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
	found, err := hasTokens(ctx, phrase, text)
	if err != nil {
		return nil, err
	}
	if found {
		statement, args = headingsLikeSQL, []any{phrase, vault}
	}
	rows, err := q.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []searchquery.HeadingEntry
	for rows.Next() {
		var h searchquery.HeadingEntry
		if err := rows.Scan(&h.Path, &h.Text); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func hasTokens(ctx context.Context, phrase, text string) (bool, error) {
	matched, err := wordmatch.NewMatcher().MatchWords(ctx, phrase, []string{text})
	if err != nil {
		return false, err
	}
	return matched[0], nil
}

func (q *QueryReads) CountUnreadableFrontmatter(ctx context.Context, vaultID domain.VaultID) (int, error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var count int
	err = q.db.QueryRowContext(ctx, countFrontmatterErrorsSQL, vault).Scan(&count)
	return count, err
}
