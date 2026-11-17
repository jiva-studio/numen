package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetHeads returns the type and the stored frontmatter of the notes at the paths, in one question.
// A path with no indexed note is absent from the result. Whole numbers of the frontmatter come back as int
// and the others as float64, the way the YAML of the file gives them.
func (q *Queries) GetHeads(ctx context.Context, vaultID domain.VaultID, paths []string) (_ map[string]frontmatter.Head, err error) {
	out := map[string]frontmatter.Head{}
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
	wanted, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, headsAtSQL, vault, string(wanted))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	for rows.Next() {
		var path, noteType string
		var stored sql.NullString
		if err := rows.Scan(&path, &noteType, &stored); err != nil {
			return nil, err
		}
		values, err := readHeadValues(stored)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[path] = frontmatter.Head{Type: domain.NoteType(noteType), Values: values}
	}
	return out, rows.Err()
}

func readHeadValues(stored sql.NullString) (map[string]any, error) {
	if !stored.Valid {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(stored.String))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("read the stored frontmatter: %w", err)
	}
	for key, value := range values {
		values[key] = convertNumbers(value)
	}
	return values, nil
}

func convertNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		if whole, err := v.Int64(); err == nil {
			return int(whole)
		}
		if decimal, err := v.Float64(); err == nil {
			return decimal
		}
	case map[string]any:
		for key, one := range v {
			v[key] = convertNumbers(one)
		}
	case []any:
		for i, one := range v {
			v[i] = convertNumbers(one)
		}
	}
	return value
}
